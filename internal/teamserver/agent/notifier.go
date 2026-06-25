package agent

import (
	"sync"
)

// EventType categorizes agent events.
type EventType string

const (
	EventConnected     EventType = "connected"
	EventDisconnected  EventType = "disconnected"
	EventStatusChange  EventType = "status_change"
	EventTaskComplete  EventType = "task_complete"
)

// Event represents an agent-related event.
type Event struct {
	AgentID string      `json:"agent_id"`
	Type    EventType   `json:"type"`
	Data    interface{} `json:"data,omitempty"`
}

// Notifier implements a pub/sub system for agent events.
type Notifier struct {
	subscribers map[string]map[chan Event]struct{} // agentID -> set of channels
	mu          sync.RWMutex
}

// NewNotifier creates a new Notifier.
func NewNotifier() *Notifier {
	return &Notifier{
		subscribers: make(map[string]map[chan Event]struct{}),
	}
}

// Subscribe returns a channel that receives events for the given agent.
// The caller is responsible for calling Unsubscribe when done.
func (n *Notifier) Subscribe(agentID string) chan Event {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan Event, 16)
	if n.subscribers[agentID] == nil {
		n.subscribers[agentID] = make(map[chan Event]struct{})
	}
	n.subscribers[agentID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber channel.
func (n *Notifier) Unsubscribe(agentID string, ch chan Event) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if subs, ok := n.subscribers[agentID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(n.subscribers, agentID)
		}
	}
}

// Notify sends an event to all subscribers for a given agent.
func (n *Notifier) Notify(event Event) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	subs, ok := n.subscribers[event.AgentID]
	if !ok {
		return
	}

	for ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, drop event to avoid blocking
		}
	}
}
