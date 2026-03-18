package auth

import (
	"testing"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/config"
)

func tokenTestConfig(jwtSecret string, accessTTL time.Duration) *config.Config {
	return &config.Config{
		JWTSecret:      jwtSecret,
		AccessTokenTTL: accessTTL,
	}
}

func TestTokenService_GenerateAccessToken(t *testing.T) {
	service := NewTokenService(tokenTestConfig("test_secret_key", 15*time.Minute))

	userID := "user-123-uuid"
	token, err := service.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateAccessToken returned empty token")
	}
}

func TestTokenService_ValidateAccessToken(t *testing.T) {
	service := NewTokenService(tokenTestConfig("test_secret_key", 15*time.Minute))

	userID := "user-123-uuid"
	token, err := service.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	extractedUserID, err := service.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if extractedUserID != userID {
		t.Errorf("UserID mismatch: got %s, expected %s", extractedUserID, userID)
	}
}

func TestTokenService_ValidateAccessToken_InvalidToken(t *testing.T) {
	service := NewTokenService(tokenTestConfig("test_secret_key", 15*time.Minute))

	_, err := service.ValidateAccessToken("invalid.token.here")
	if err == nil {
		t.Error("ValidateAccessToken should return error for invalid token")
	}
}

func TestTokenService_ValidateAccessToken_WrongSecret(t *testing.T) {
	service1 := NewTokenService(tokenTestConfig("secret_key_1", 15*time.Minute))
	service2 := NewTokenService(tokenTestConfig("secret_key_2", 15*time.Minute))

	token, _ := service1.GenerateAccessToken("user-123")

	// Validate with different secret should fail
	_, err := service2.ValidateAccessToken(token)
	if err == nil {
		t.Error("ValidateAccessToken should fail with wrong secret")
	}
}

func TestTokenService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	service := NewTokenService(tokenTestConfig("test_secret_key", 1*time.Nanosecond))

	token, _ := service.GenerateAccessToken("user-123")

	time.Sleep(10 * time.Millisecond)

	_, err := service.ValidateAccessToken(token)
	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got: %v", err)
	}
}

func TestTokenService_GetAccessTokenTTL(t *testing.T) {
	service := NewTokenService(tokenTestConfig("test_secret_key", 15*time.Minute))

	ttl := service.GetAccessTokenTTL()
	expected := 900 // 15 minutes in seconds

	if ttl != expected {
		t.Errorf("TTL mismatch: got %d, expected %d", ttl, expected)
	}
}
