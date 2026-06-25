package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	session, err := NewCipherSession(key)
	if err != nil {
		t.Fatalf("NewCipherSession failed: %v", err)
	}

	plaintext := []byte("sensitive implant communication data")

	ciphertext, err := session.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify nonce + ciphertext + tag format
	if len(ciphertext) < NonceSize+TagSize {
		t.Fatalf("ciphertext too short: %d bytes (minimum %d)", len(ciphertext), NonceSize+TagSize)
	}

	decrypted, err := session.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted data doesn't match original")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	key := make([]byte, KeySize)
	session, _ := NewCipherSession(key)

	_, err := session.Decrypt([]byte("corrupted"))
	if err == nil {
		t.Error("expected error for corrupted data")
	}
}

func TestInvalidKeySize(t *testing.T) {
	_, err := NewCipherSession([]byte("short"))
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestEncryptRandomNonce(t *testing.T) {
	key := make([]byte, KeySize)
	session, _ := NewCipherSession(key)
	plaintext := []byte("test")

	c1, _ := session.Encrypt(plaintext)
	c2, _ := session.Encrypt(plaintext)

	// Same plaintext encrypted twice should produce different ciphertexts
	if bytes.Equal(c1, c2) {
		t.Error("same plaintext produced identical ciphertexts (nonce reuse?)")
	}
}
