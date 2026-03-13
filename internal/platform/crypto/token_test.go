package crypto

import (
	"testing"
)

func TestGenerateRandomToken(t *testing.T) {
	token, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("GenerateRandomToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateRandomToken returned empty token")
	}

	// Token should be base64 encoded, so longer than 32
	if len(token) < 32 {
		t.Errorf("Token length too short: %d", len(token))
	}
}

func TestGenerateRandomTokenUniqueness(t *testing.T) {
	token1, _ := GenerateRandomToken(32)
	token2, _ := GenerateRandomToken(32)

	if token1 == token2 {
		t.Error("GenerateRandomToken produced identical tokens")
	}
}

func TestHashToken(t *testing.T) {
	token := "test_token_123"

	hash := HashToken(token)
	if hash == "" {
		t.Fatal("HashToken returned empty hash")
	}

	// SHA-256 produces 64 character hex string
	if len(hash) != 64 {
		t.Errorf("Hash length incorrect: %d, expected 64", len(hash))
	}

	// Same input should produce same hash
	hash2 := HashToken(token)
	if hash != hash2 {
		t.Error("HashToken produced different hashes for same input")
	}

	// Different input should produce different hash
	hash3 := HashToken("different_token")
	if hash == hash3 {
		t.Error("HashToken produced same hash for different inputs")
	}
}
