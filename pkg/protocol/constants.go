package protocol

// Protocol version
const ProtocolVersion = 1

// Message types
const (
	MsgHandshakeRequest  uint32 = 0x01 // Implant → Server
	MsgHandshakeResponse uint32 = 0x02 // Server → Implant
	MsgHeartbeat         uint32 = 0x03 // Implant → Server (fire-and-forget)
	MsgTaskRequest       uint32 = 0x04 // Implant → Server (poll for tasks)
	MsgTaskResponse      uint32 = 0x05 // Server → Implant (list of tasks)
	MsgTaskResult        uint32 = 0x06 // Implant → Server (task output)
	MsgTunnelConnect     uint32 = 0x10 // Server → Implant (request tunnel to target)
	MsgTunnelConnected   uint32 = 0x11 // Implant → Server (tunnel connect result)
	MsgTunnelData        uint32 = 0x12 // Bidirectional (data chunk on tunnel)
	MsgTunnelClose       uint32 = 0x13 // Bidirectional (close tunnel session)
	MsgError             uint32 = 0xFF // Either direction
)

// MaxMessageSize is the maximum allowed size of a single protocol message (10 MB).
const MaxMessageSize = 10 * 1024 * 1024

// AES-GCM parameters
const (
	KeySize   = 32 // AES-256
	NonceSize = 12
	TagSize   = 16
)

// Envelope header sizes
const (
	LengthPrefixSize = 4 // uint32 big-endian
	TypePrefixSize   = 4 // uint32 big-endian
	EnvelopeOverhead = LengthPrefixSize + TypePrefixSize
)
