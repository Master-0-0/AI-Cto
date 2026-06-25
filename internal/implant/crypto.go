package implant

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	nonceSize = 12
	tagSize   = 16
)

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// Returns: nonce (12 bytes) + ciphertext + tag (16 bytes).
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided key.
// Expects input: nonce (12 bytes) + ciphertext + tag (16 bytes).
func Decrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	if len(data) < nonceSize+tagSize {
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(data))
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// DeriveAgentKey derives a per-agent AES-256 key from the master key and agent ID
// using HKDF-SHA256. This must match the server-side derivation in crypto/keys.go.
func DeriveAgentKey(masterKey []byte, agentID string) ([]byte, error) {
	agentKey := make([]byte, 32)
	reader := hkdf.New(sha256.New, masterKey, []byte(agentID), []byte("c2-session-key"))
	if _, err := io.ReadFull(reader, agentKey); err != nil {
		return nil, fmt.Errorf("deriving agent key: %w", err)
	}
	return agentKey, nil
}
