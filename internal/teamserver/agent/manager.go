package agent

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"c2/internal/teamserver/crypto"
	"c2/internal/teamserver/database"
)

// AgentState holds the runtime state for a connected agent.
type AgentState struct {
	Agent    database.Agent
	Conn     net.Conn
	Cipher   *crypto.CipherSession
	LastSeen time.Time
	mu       sync.RWMutex
}

// Manager manages all connected agents.
type Manager struct {
	db       *sql.DB
	agents   map[string]*AgentState
	mu       sync.RWMutex
	notifier *Notifier
}

// NewManager creates a new AgentManager.
func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db:       db,
		agents:   make(map[string]*AgentState),
		notifier: NewNotifier(),
	}
}

// Notifier returns the agent event notifier.
func (m *Manager) Notifier() *Notifier {
	return m.notifier
}

// Register adds or updates an agent in the registry and database.
func (m *Manager) Register(agent *database.Agent, conn net.Conn, cipher *crypto.CipherSession) {
	m.mu.Lock()

	existing, exists := m.agents[agent.ID]
	if exists {
		// Reconnect: close old connection, update with new
		if existing.Conn != nil {
			existing.Conn.Close()
		}
		existing.Conn = conn
		existing.Cipher = cipher
		existing.Agent = *agent
		existing.LastSeen = time.Now()
		m.mu.Unlock()

		// Update database
		m.updateDB(agent)
		m.notifier.Notify(Event{AgentID: agent.ID, Type: EventStatusChange, Data: "active"})
		log.Printf("[INFO] Agent %s reconnected (%s@%s)", agent.ID, agent.Username, agent.Hostname)
		return
	}

	state := &AgentState{
		Agent:    *agent,
		Conn:     conn,
		Cipher:   cipher,
		LastSeen: time.Now(),
	}
	m.mu.Unlock()

	// Insert into database — must succeed before adding to memory
	if err := m.insertDB(agent); err != nil {
		log.Printf("[ERROR] Agent %s DB insert failed: %v", agent.ID, err)
		conn.Close()
		return
	}

	m.mu.Lock()
	m.agents[agent.ID] = state
	m.mu.Unlock()

	m.notifier.Notify(Event{AgentID: agent.ID, Type: EventConnected, Data: agent})
	log.Printf("[INFO] Agent %s registered (%s@%s, %s/%s)", agent.ID, agent.Username, agent.Hostname, agent.OS, agent.Arch)
}

// Unregister removes an agent from the registry.
func (m *Manager) Unregister(agentID string) {
	m.mu.Lock()
	state, ok := m.agents[agentID]
	if !ok {
		m.mu.Unlock()
		return
	}

	if state.Conn != nil {
		state.Conn.Close()
	}
	delete(m.agents, agentID)
	m.mu.Unlock()

	m.notifier.Notify(Event{AgentID: agentID, Type: EventDisconnected, Data: nil})
	log.Printf("[INFO] Agent %s unregistered", agentID)
}

// UpdateHeartbeat updates the last seen timestamp for an agent.
func (m *Manager) UpdateHeartbeat(agentID string) {
	m.mu.RLock()
	state, ok := m.agents[agentID]
	m.mu.RUnlock()

	if ok {
		state.mu.Lock()
		state.LastSeen = time.Now()
		state.mu.Unlock()
	}
}

// Get returns an agent's runtime state by ID.
func (m *Manager) Get(agentID string) (*AgentState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.agents[agentID]
	return state, ok
}

// List returns all agents, optionally filtered by status.
func (m *Manager) List(statusFilter string) []*database.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*database.Agent
	for _, state := range m.agents {
		a := state.Agent
		if statusFilter != "" && a.Status != statusFilter {
			continue
		}
		// Update last_seen from memory
		a.LastSeen = state.LastSeen.UTC().Format(time.RFC3339)
		result = append(result, &a)
	}
	return result
}

// MarkDead marks agents as dead if they haven't been seen within the timeout.
func (m *Manager) MarkDead(timeout time.Duration) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, state := range m.agents {
		state.mu.RLock()
		lastSeen := state.LastSeen
		state.mu.RUnlock()

		if now.Sub(lastSeen) > timeout {
			state.Agent.Status = "dead"
			m.markDeadDB(id)

			m.notifier.Notify(Event{AgentID: id, Type: EventStatusChange, Data: "dead"})
			log.Printf("[INFO] Agent %s marked as dead (last seen: %v ago)", id, now.Sub(lastSeen).Round(time.Second))
		}
	}
}

// DeadCheckLoop periodically checks for dead agents.
func (m *Manager) DeadCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.MarkDead(5 * time.Minute) // Default: 5 minutes timeout
		}
	}
}

// GetCipher returns the cipher session for an agent.
func (m *Manager) GetCipher(agentID string) (*crypto.CipherSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}
	return state.Cipher, nil
}

func (m *Manager) insertDB(agent *database.Agent) error {
	now := time.Now().UTC().Format(time.RFC3339)
	agent.FirstSeen = now
	agent.LastSeen = now

	_, err := m.db.Exec(
		`INSERT INTO agents (id, listener_id, hostname, os, arch, username, pid, local_ips, version, exe_name, remote_ip, first_seen, last_seen, heartbeat_interval, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID, agent.ListenerID, agent.Hostname, agent.OS, agent.Arch,
		agent.Username, agent.PID, agent.LocalIPs, agent.Version,
		agent.ExeName, agent.RemoteIP,
		agent.FirstSeen, agent.LastSeen, agent.HeartbeatInterval, agent.Status,
	)
	if err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}
	return nil
}

func (m *Manager) updateDB(agent *database.Agent) {
	now := time.Now().UTC().Format(time.RFC3339)
	agent.LastSeen = now

	_, err := m.db.Exec(
		`UPDATE agents SET hostname=?, os=?, arch=?, username=?, pid=?, local_ips=?, exe_name=?, remote_ip=?, last_seen=?, status='active'
		 WHERE id=?`,
		agent.Hostname, agent.OS, agent.Arch, agent.Username, agent.PID, agent.LocalIPs,
		agent.ExeName, agent.RemoteIP,
		now, agent.ID,
	)
	if err != nil {
		log.Printf("[ERROR] Updating agent %s: %v", agent.ID, err)
	}
}

func (m *Manager) markDeadDB(agentID string) {
	m.db.Exec("UPDATE agents SET status = 'dead' WHERE id = ?", agentID)
}
