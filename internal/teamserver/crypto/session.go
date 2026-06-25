package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// NonceSize is the AES-GCM nonce size (12 bytes).
	NonceSize = 12
	// TagSize is the AES-GCM authentication tag size (16 bytes).
	TagSize = 16
)

// CipherSession handles per-agent AES-256-GCM encryption/decryption.
type CipherSession struct {
	key   []byte
	aead  cipher.AEAD
}

// NewCipherSession creates a new CipherSession with the given 32-byte key.
func NewCipherSession(key []byte) (*CipherSession, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d (expected %d)", len(key), KeySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return &CipherSession{
		key:  key,
		aead: aead,
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns: nonce (12 bytes) + ciphertext + tag (16 bytes)
func (cs *CipherSession) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	// Seal appends ciphertext to nonce and returns: nonce + ciphertext + tag
	ciphertext := cs.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// Expects input format: nonce (12 bytes) + ciphertext + tag (16 bytes)
func (cs *CipherSession) Decrypt(data []byte) ([]byte, error) {
	if len(data) < NonceSize+TagSize {
		return nil, fmt.Errorf("ciphertext too short: %d bytes (minimum %d)", len(data), NonceSize+TagSize)
	}

	nonce := data[:NonceSize]
	ciphertext := data[NonceSize:]

	plaintext, err := cs.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptNoNonce is used by the implant side where nonce and ciphertext are
// separate in the envelope format. This encrypts and returns the full blob.
// Alias for Encrypt to maintain API clarity.
func (cs *CipherSession) EncryptNoNonce(plaintext []byte) ([]byte, error) {
	return cs.Encrypt(plaintext)
}

// Key returns the session key.
func (cs *CipherSession) Key() []byte {
	return cs.key
}
