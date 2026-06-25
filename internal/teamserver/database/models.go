package database

// Operator represents an authenticated user of the C2 platform.
type Operator struct {
	ID           string  `json:"id"`
	Username     string  `json:"username"`
	PasswordHash string  `json:"-"` // bcrypt hash, never serialized to JSON
	Role         string  `json:"role"` // "admin" | "operator"
	CreatedAt    string  `json:"created_at"`
	LastLogin    *string `json:"last_login,omitempty"`
	Active       bool    `json:"active"`
}

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID         string `json:"id"`
	OperatorID string `json:"operator_id"`
	TokenHash  string `json:"-"` // SHA-256 of the opaque token
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
	Revoked    bool   `json:"revoked"`
}

// Listener represents a C2 listener endpoint.
type Listener struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Protocol   string  `json:"protocol"` // "tcp"
	Host       string  `json:"host"`
	Port       int     `json:"port"`
	Status     string  `json:"status"` // "stopped", "running", "error"
	ConfigJSON *string `json:"config_json,omitempty"`
	CreatedAt  string  `json:"created_at"`
	StartedAt  *string `json:"started_at,omitempty"`
}

// Agent represents a connected implant.
type Agent struct {
	ID                 string  `json:"id"`
	ListenerID         *string `json:"listener_id,omitempty"`
	Hostname           string  `json:"hostname"`
	OS                 string  `json:"os"`
	Arch               string  `json:"arch"`
	Username           string  `json:"username"`
	PID                int     `json:"pid"`
	LocalIPs           *string `json:"local_ips,omitempty"` // JSON array
	Version            string  `json:"version"`
	FirstSeen          string  `json:"first_seen"`
	LastSeen           string  `json:"last_seen"`
	HeartbeatInterval  int     `json:"heartbeat_interval"`
	Status             string  `json:"status"` // "active", "inactive", "dead"
	SessionKey         *string `json:"-"`      // encrypted AES key (hex), never exposed
	MetadataJSON       *string `json:"metadata_json,omitempty"`
}

// Task represents a command task queued for an agent.
type Task struct {
	ID           string  `json:"id"`
	AgentID      string  `json:"agent_id"`
	OperatorID   *string `json:"operator_id,omitempty"`
	Command      string  `json:"command"`
	Arguments    string  `json:"arguments"`
	Status       string  `json:"status"` // "pending", "sent", "completed", "error", "cancelled"
	Output       *string `json:"output,omitempty"`        // base64-encoded
	ErrorMessage *string `json:"error_message,omitempty"`
	TimeoutSec   int     `json:"timeout_sec"`
	CreatedAt    string  `json:"created_at"`
	SentAt       *string `json:"sent_at,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
	DurationMs   *int    `json:"duration_ms,omitempty"`
}

// AuditLog represents an audited action.
type AuditLog struct {
	ID           string  `json:"id"`
	OperatorID   *string `json:"operator_id,omitempty"`
	Action       string  `json:"action"`
	ResourceType string  `json:"resource_type"`
	ResourceID   *string `json:"resource_id,omitempty"`
	DetailsJSON  *string `json:"details_json,omitempty"`
	IPAddress    *string `json:"ip_address,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// DashboardStats holds aggregated counts for the dashboard.
type DashboardStats struct {
	AgentCount      int `json:"agent_count"`
	ActiveAgents    int `json:"active_agents"`
	ListenerCount   int `json:"listener_count"`
	RunningListeners int `json:"running_listeners"`
	TaskCount       int `json:"task_count"`
	TasksToday      int `json:"tasks_today"`
}
