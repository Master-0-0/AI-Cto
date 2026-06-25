package listener

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPListener implements the Listener interface for plain TCP.
type TCPListener struct {
	id            string
	host          string
	port          int
	listener      net.Listener
	status        string
	mu            sync.Mutex
	agentCallback AgentCallback
}

// NewTCPListener creates a new TCP listener instance.
func NewTCPListener(id, host string, port int, callback AgentCallback) *TCPListener {
	return &TCPListener{
		id:            id,
		host:          host,
		port:          port,
		status:        "stopped",
		agentCallback: callback,
	}
}

// Start begins accepting TCP connections. Blocks until the context is cancelled.
func (t *TCPListener) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", t.host, t.port)

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		t.setStatus("error")
		return fmt.Errorf("binding to %s: %w", addr, err)
	}

	t.mu.Lock()
	t.listener = ln
	t.status = "running"
	t.mu.Unlock()

	log.Printf("[INFO] TCP listener %s started on %s", t.id, addr)

	// Accept loop
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Printf("[ERROR] TCP listener %s accept error: %v", t.id, err)
					t.setStatus("error")
					return
				}
			}

			log.Printf("[INFO] TCP listener %s: new connection from %s", t.id, conn.RemoteAddr())

			// Pass connection to the agent handler with listener ID
			if t.agentCallback != nil {
				go t.agentCallback(t.id, conn)
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return t.Stop()
}

// Stop closes the TCP listener.
func (t *TCPListener) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.listener != nil {
		if err := t.listener.Close(); err != nil {
			return err
		}
		t.listener = nil
	}

	t.status = "stopped"
	log.Printf("[INFO] TCP listener %s stopped", t.id)
	return nil
}

// ID returns the listener's unique identifier.
func (t *TCPListener) ID() string {
	return t.id
}

// Type returns "tcp".
func (t *TCPListener) Type() string {
	return "tcp"
}

// Status returns the current status.
func (t *TCPListener) Status() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status
}

// Addr returns the listener's address string.
func (t *TCPListener) Addr() string {
	return fmt.Sprintf("%s:%d", t.host, t.port)
}

func (t *TCPListener) setStatus(status string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = status
}
