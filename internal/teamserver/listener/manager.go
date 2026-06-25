package listener

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"c2/internal/teamserver/database"
)

// Manager manages the lifecycle of all listeners.
type Manager struct {
	db            *sql.DB
	listeners     map[string]Listener
	mu            sync.RWMutex
	agentCallback AgentCallback
}

// NewManager creates a new ListenerManager.
func NewManager(db *sql.DB, callback AgentCallback) *Manager {
	return &Manager{
		db:            db,
		listeners:     make(map[string]Listener),
		agentCallback: callback,
	}
}

// Create creates a new listener and persists it to the database.
func (m *Manager) Create(name, protocol, host string, port int) (*Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for port conflicts
	for _, l := range m.listeners {
		if l.Addr() == fmt.Sprintf("%s:%d", host, port) {
			return nil, fmt.Errorf("address %s:%d is already in use", host, port)
		}
	}

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Persist to DB
	_, err := m.db.Exec(
		"INSERT INTO listeners (id, name, protocol, host, port, status, created_at) VALUES (?, ?, ?, ?, ?, 'stopped', ?)",
		id, name, protocol, host, port, now,
	)
	if err != nil {
		return nil, fmt.Errorf("persisting listener: %w", err)
	}

	// Create the appropriate listener type
	var l Listener
	switch protocol {
	case "tcp":
		l = NewTCPListener(id, host, port, m.agentCallback)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	m.listeners[id] = l
	return &l, nil
}

// Start starts a listener by ID.
func (m *Manager) Start(ctx context.Context, id string) error {
	m.mu.RLock()
	l, ok := m.listeners[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("listener %s not found", id)
	}

	if l.Status() == "running" {
		return fmt.Errorf("listener %s is already running", id)
	}

	// Start the listener in a goroutine
	go func() {
		if err := l.Start(ctx); err != nil {
			log.Printf("[ERROR] Listener %s error: %v", id, err)
			m.updateStatus(id, "error")
		}
	}()

	// Update status in DB
	m.updateStatus(id, "running")
	now := time.Now().UTC().Format(time.RFC3339)
	m.db.Exec("UPDATE listeners SET status = 'running', started_at = ? WHERE id = ?", now, id)

	return nil
}

// Stop stops a listener by ID.
func (m *Manager) Stop(id string) error {
	m.mu.RLock()
	l, ok := m.listeners[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("listener %s not found", id)
	}

	if l.Status() != "running" {
		return fmt.Errorf("listener %s is not running", id)
	}

	if err := l.Stop(); err != nil {
		return fmt.Errorf("stopping listener: %w", err)
	}

	m.updateStatus(id, "stopped")
	m.db.Exec("UPDATE listeners SET status = 'stopped' WHERE id = ?", id)

	return nil
}

// Delete removes a listener. The listener must be stopped first.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	l, ok := m.listeners[id]
	if !ok {
		return fmt.Errorf("listener %s not found", id)
	}

	if l.Status() == "running" {
		return fmt.Errorf("cannot delete running listener %s", id)
	}

	delete(m.listeners, id)
	m.db.Exec("DELETE FROM listeners WHERE id = ?", id)

	return nil
}

// List returns all listeners.
func (m *Manager) List() []database.Listener {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []database.Listener
	for _, l := range m.listeners {
		dl := m.toDBListener(l)
		result = append(result, *dl)
	}
	return result
}

// Get returns a single listener by ID.
func (m *Manager) Get(id string) (*database.Listener, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	l, ok := m.listeners[id]
	if !ok {
		return nil, fmt.Errorf("listener %s not found", id)
	}

	return m.toDBListener(l), nil
}

// Recover restarts all listeners that were running when the server was shut down.
func (m *Manager) Recover(ctx context.Context) {
	rows, err := m.db.Query("SELECT id, name, protocol, host, port FROM listeners WHERE status = 'running'")
	if err != nil {
		log.Printf("[ERROR] Recovering listeners: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, protocol, host string
		var port int
		if err := rows.Scan(&id, &name, &protocol, &host, &port); err != nil {
			log.Printf("[ERROR] Scanning listener row: %v", err)
			continue
		}

		var l Listener
		switch protocol {
		case "tcp":
			l = NewTCPListener(id, host, port, m.agentCallback)
		default:
			continue
		}

		m.mu.Lock()
		m.listeners[id] = l
		m.mu.Unlock()

		go func(lid string) {
			log.Printf("[INFO] Recovering listener %s (%s:%d)", lid, host, port)
			if err := l.Start(ctx); err != nil {
				log.Printf("[ERROR] Recovered listener %s error: %v", lid, err)
			}
		}(id)
	}
}

func (m *Manager) updateStatus(id, status string) {
	m.mu.RLock()
	l, ok := m.listeners[id]
	m.mu.RUnlock()

	if ok {
		// Update in-memory status (listener handles its own status tracking)
		_ = l
	}
}

func (m *Manager) toDBListener(l Listener) *database.Listener {
	// Parse host:port from Addr()
	addr := l.Addr()
	host := ""
	port := 0
	if addr != "" {
		fmt.Sscanf(addr, "%[^:]:%d", &host, &port)
	}

	// Get details from the actual listener implementation
	switch t := l.(type) {
	case *TCPListener:
		return &database.Listener{
			ID:       l.ID(),
			Name:     "", // Name is stored only in DB, not in memory for simplicity
			Protocol: l.Type(),
			Host:     t.host,
			Port:     t.port,
			Status:   l.Status(),
		}
	default:
		return &database.Listener{
			ID:       l.ID(),
			Protocol: l.Type(),
			Host:     host,
			Port:     port,
			Status:   l.Status(),
		}
	}
}
