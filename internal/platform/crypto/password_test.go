package crypto

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "test_password_123"

	hash, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	// Hash should start with $argon2id$
	if len(hash) < 10 || hash[:10] != "$argon2id$" {
		t.Errorf("Hash does not have correct prefix: %s", hash[:20])
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "test_password_123"

	hash, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Test correct password
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !valid {
		t.Error("VerifyPassword returned false for correct password")
	}

	// Test incorrect password
	valid, err = VerifyPassword("wrong_password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if valid {
		t.Error("VerifyPassword returned true for incorrect password")
	}
}

func TestHashPasswordUniqueness(t *testing.T) {
	password := "same_password"

	hash1, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	hash2, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Same password should produce different hashes (different salts)
	if hash1 == hash2 {
		t.Error("HashPassword produced identical hashes for same password")
	}

	// Both should still verify correctly
	valid1, _ := VerifyPassword(password, hash1)
	valid2, _ := VerifyPassword(password, hash2)
	if !valid1 || !valid2 {
		t.Error("Verification failed for unique hashes")
	}
}

func TestVerifyPasswordInvalidHash(t *testing.T) {
	_, err := VerifyPassword("password", "invalid_hash")
	if err == nil {
		t.Error("VerifyPassword should return error for invalid hash")
	}
}
