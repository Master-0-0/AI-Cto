package listener

import (
	"context"
	"net"
)

// Listener defines the interface for all C2 listener types.
type Listener interface {
	// Start begins accepting connections. Blocks until the context is cancelled
	// or an error occurs. Should be run in a goroutine.
	Start(ctx context.Context) error

	// Stop gracefully stops the listener and closes all connections.
	Stop() error

	// ID returns the listener's unique identifier.
	ID() string

	// Type returns the protocol type (e.g., "tcp", "ws", "dns").
	Type() string

	// Status returns the current status ("stopped", "running", "error").
	Status() string

	// Addr returns the listener's network address.
	Addr() string
}

// AgentCallback is called when a new agent connection is accepted.
// listenerID identifies which listener accepted the connection.
type AgentCallback func(listenerID string, conn net.Conn)
