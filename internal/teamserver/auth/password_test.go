package auth

import (
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("got empty hash")
	}

	// Verify correct password
	if !CheckPassword(hash, password) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Verify wrong password
	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashPasswordUnique(t *testing.T) {
	password := "same-password"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// bcrypt generates unique salts, hashes should differ
	if hash1 == hash2 {
		t.Error("same password should produce unique hashes due to random salt")
	}
}

func TestCheckPasswordEmpty(t *testing.T) {
	hash, _ := HashPassword("test")
	if CheckPassword(hash, "") {
		t.Error("empty password should not match")
	}
}
