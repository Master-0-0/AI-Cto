package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// KeySize is the size of AES-256 keys in bytes.
const KeySize = 32

// GenerateMasterKey creates a new random 32-byte master key.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generating master key: %w", err)
	}
	return key, nil
}

// DeriveAgentKey derives a per-agent AES key from the master key and agent ID
// using HKDF-SHA256.
func DeriveAgentKey(masterKey []byte, agentID string) ([]byte, error) {
	agentKey := make([]byte, KeySize)
	reader := hkdf.New(sha256.New, masterKey, []byte(agentID), []byte("c2-session-key"))
	if _, err := io.ReadFull(reader, agentKey); err != nil {
		return nil, fmt.Errorf("deriving agent key: %w", err)
	}
	return agentKey, nil
}

// EncodeKey returns the hex encoding of a key.
func EncodeKey(key []byte) string {
	return hex.EncodeToString(key)
}

// DecodeKey decodes a hex-encoded key.
func DecodeKey(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decoding key: %w", err)
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d (expected %d)", len(key), KeySize)
	}
	return key, nil
}
