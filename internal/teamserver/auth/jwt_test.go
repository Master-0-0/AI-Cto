package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!!!"
	expiry := 15 * time.Minute

	token, err := GenerateAccessToken("op-123", "admin", "admin", secret, expiry)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("got empty token")
	}

	claims, err := ValidateAccessToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.OperatorID != "op-123" {
		t.Errorf("expected OperatorID 'op-123', got '%s'", claims.OperatorID)
	}
	if claims.Username != "admin" {
		t.Errorf("expected Username 'admin', got '%s'", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("expected Role 'admin', got '%s'", claims.Role)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	secret := "test-secret-key"

	_, err := ValidateAccessToken("invalid.token.here", secret)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateWrongSecret(t *testing.T) {
	secret := "correct-secret-key-for-testing!"
	wrongSecret := "wrong-secret-key-for-testing!!"
	expiry := 15 * time.Minute

	token, err := GenerateAccessToken("op-1", "user", "operator", secret, expiry)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = ValidateAccessToken(token, wrongSecret)
	if err == nil {
		t.Error("expected error with wrong secret")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64-char hex token, got %d chars", len(token))
	}
}

func TestHashToken(t *testing.T) {
	token := "test-refresh-token"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("same token produced different hashes")
	}

	if len(hash1) != 64 { // SHA-256 = 64 hex chars
		t.Errorf("expected 64-char hash, got %d", len(hash1))
	}
}
