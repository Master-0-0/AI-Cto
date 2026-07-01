package socks5

import (
	"context"
	"fmt"
	"sync"

	"c2/pkg/protocol"
)

// ProxyRegistry manages SOCKS proxies for all agents.
type ProxyRegistry struct {
	proxies map[string]*SocksManager // agentID -> manager
	mu      sync.RWMutex
}

// NewProxyRegistry creates a new proxy registry.
func NewProxyRegistry() *ProxyRegistry {
	return &ProxyRegistry{
		proxies: make(map[string]*SocksManager),
	}
}

// Start starts a SOCKS proxy for the given agent on the specified port.
func (r *ProxyRegistry) Start(agentID string, port int, sendMsg func(msgType uint32, body interface{}) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.proxies[agentID]; ok {
		status := existing.Status()
		if status.Active {
			return fmt.Errorf("SOCKS proxy already running on port %d for agent %s", status.Port, agentID)
		}
	}

	mgr := NewSocksManager(agentID, port, sendMsg)
	if err := mgr.Start(context.Background()); err != nil {
		return err
	}

	r.proxies[agentID] = mgr
	return nil
}

// Stop stops the SOCKS proxy for the given agent.
func (r *ProxyRegistry) Stop(agentID string) error {
	r.mu.Lock()
	mgr, ok := r.proxies[agentID]
	if ok {
		delete(r.proxies, agentID)
	}
	r.mu.Unlock()

	if ok {
		return mgr.Stop()
	}
	return fmt.Errorf("no SOCKS proxy found for agent %s", agentID)
}

// Status returns the SOCKS proxy status for the given agent.
func (r *ProxyRegistry) Status(agentID string) (SocksStatus, bool) {
	r.mu.RLock()
	mgr, ok := r.proxies[agentID]
	r.mu.RUnlock()
	if !ok {
		return SocksStatus{Active: false}, false
	}
	return mgr.Status(), true
}

// GetManager returns the SocksManager for the given agent, or nil.
func (r *ProxyRegistry) GetManager(agentID string) *SocksManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.proxies[agentID]
}

// HandleTunnelConnected routes a TunnelConnected message to the correct agent's manager.
func (r *ProxyRegistry) HandleTunnelConnected(agentID string, msg protocol.TunnelConnected) {
	mgr := r.GetManager(agentID)
	if mgr != nil {
		mgr.HandleTunnelConnected(msg)
	}
}

// RouteData routes tunnel data to the correct agent's manager.
func (r *ProxyRegistry) RouteData(agentID, tunnelID string, data []byte) {
	mgr := r.GetManager(agentID)
	if mgr != nil {
		mgr.RouteData(tunnelID, data)
	}
}

// HandleTunnelClose routes a tunnel close message to the correct agent's manager.
func (r *ProxyRegistry) HandleTunnelClose(agentID string, msg protocol.TunnelClose) {
	mgr := r.GetManager(agentID)
	if mgr != nil {
		mgr.HandleTunnelClose(msg)
	}
}

// StopAll stops all active SOCKS proxies.
func (r *ProxyRegistry) StopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for agentID, mgr := range r.proxies {
		mgr.Stop()
		delete(r.proxies, agentID)
	}
}
