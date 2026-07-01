package socks5

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"c2/pkg/protocol"
)

// RelayConn implements net.Conn by bridging through implant tunnel channels.
type RelayConn struct {
	tunnelID      string
	target        string
	readCh        chan []byte
	sendMsg       func(msgType uint32, body interface{}) error
	closeCh       chan struct{}
	onClose       func(tunnelID string)
	closed        int32
	readDeadline  time.Time
	writeDeadline time.Time
	mu            sync.Mutex
	readBuf       []byte
	readBufPos    int
}

// NewRelayConn creates a new relay connection.
func NewRelayConn(tunnelID, target string, readCh chan []byte, sendMsg func(msgType uint32, body interface{}) error, onClose func(tunnelID string)) *RelayConn {
	return &RelayConn{
		tunnelID: tunnelID,
		target:   target,
		readCh:   readCh,
		sendMsg:  sendMsg,
		closeCh:  make(chan struct{}),
		onClose:  onClose,
	}
}

func (c *RelayConn) Read(b []byte) (int, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return 0, net.ErrClosed
	}

	// If we have buffered data from a previous read, return it first
	c.mu.Lock()
	if c.readBufPos < len(c.readBuf) {
		n := copy(b, c.readBuf[c.readBufPos:])
		c.readBufPos += n
		if c.readBufPos >= len(c.readBuf) {
			c.readBuf = nil
			c.readBufPos = 0
		}
		c.mu.Unlock()
		return n, nil
	}
	c.mu.Unlock()

	var timeout <-chan time.Time
	c.mu.Lock()
	if !c.readDeadline.IsZero() {
		timeout = time.After(time.Until(c.readDeadline))
	}
	c.mu.Unlock()

	select {
	case data, ok := <-c.readCh:
		if !ok {
			return 0, net.ErrClosed
		}
		if len(data) == 0 {
			// Empty data signals tunnel close from implant
			return 0, fmt.Errorf("tunnel closed by implant")
		}
		n := copy(b, data)
		// Buffer remaining data for next read
		if n < len(data) {
			c.mu.Lock()
			c.readBuf = data
			c.readBufPos = n
			c.mu.Unlock()
		}
		return n, nil
	case <-c.closeCh:
		return 0, net.ErrClosed
	case <-timeout:
		return 0, fmt.Errorf("read timeout")
	}
}

func (c *RelayConn) Write(b []byte) (int, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return 0, net.ErrClosed
	}

	c.mu.Lock()
	if !c.writeDeadline.IsZero() && time.Now().After(c.writeDeadline) {
		c.mu.Unlock()
		return 0, fmt.Errorf("write timeout")
	}
	c.mu.Unlock()

	dataMsg := protocol.TunnelData{TunnelID: c.tunnelID, Data: b}
	if err := c.sendMsg(protocol.MsgTunnelData, dataMsg); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *RelayConn) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil // already closed
	}

	select {
	case <-c.closeCh:
		// already closed
	default:
		close(c.closeCh)
	}

	// Send close message to implant
	closeMsg := protocol.TunnelClose{TunnelID: c.tunnelID}
	c.sendMsg(protocol.MsgTunnelClose, closeMsg) // best effort

	if c.onClose != nil {
		c.onClose(c.tunnelID)
	}
	return nil
}

func (c *RelayConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (c *RelayConn) RemoteAddr() net.Addr {
	return &addr{network: "tcp", address: c.target}
}

func (c *RelayConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *RelayConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	c.readDeadline = t
	c.mu.Unlock()
	return nil
}

func (c *RelayConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	c.writeDeadline = t
	c.mu.Unlock()
	return nil
}

// addr implements net.Addr for tunnel remote addresses.
type addr struct {
	network string
	address string
}

func (a *addr) Network() string { return a.network }
func (a *addr) String() string  { return a.address }
