package verification

import (
	"context"
	"testing"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/email"
	"github.com/alumieye/eyeapp-backend/internal/identity"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

func TestVerifyToken_EmptyToken(t *testing.T) {
	svc := NewService(logger.NewNop(), &mockRepo{}, nil, nil, 24*time.Hour, "http://localhost:5173/verify-email")

	err := svc.VerifyToken(context.Background(), "")
	if err != ErrInvalidVerificationToken {
		t.Errorf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerifyToken_NotFound(t *testing.T) {
	svc := NewService(logger.NewNop(), &mockRepo{notFound: true}, nil, nil, 24*time.Hour, "http://localhost:5173/verify-email")

	err := svc.VerifyToken(context.Background(), "any-token")
	if err != ErrInvalidVerificationToken {
		t.Errorf("expected ErrInvalidVerificationToken for unknown token, got %v", err)
	}
}

type mockRepo struct {
	notFound bool
}

func (m *mockRepo) Create(ctx context.Context, token *Token) error {
	return nil
}

func (m *mockRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*Token, error) {
	if m.notFound {
		return nil, ErrTokenNotFound
	}
	return &Token{ID: "1", UserID: "u1", TokenHash: tokenHash, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (m *mockRepo) MarkConsumed(ctx context.Context, id string) error {
	return nil
}

func TestTokenHash_LookupConsistency(t *testing.T) {
	rawToken := "test-token-abc123"
	hash1 := crypto.HashToken(rawToken)
	hash2 := crypto.HashToken(rawToken)

	if hash1 != hash2 {
		t.Error("same token should produce same hash")
	}
	if hash1 == "" {
		t.Error("hash should not be empty")
	}
}

func TestResendVerification_NoopWhenNotFound(t *testing.T) {
	// Resend should not error when identity not found (no leak)
	svc := NewService(logger.NewNop(), &mockRepo{}, &mockIdentityRepo{notFound: true}, &email.NoopSender{}, 24*time.Hour, "http://localhost:3000")

	err := svc.ResendVerification(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Errorf("resend should return nil when not found (no leak), got %v", err)
	}
}

type mockIdentityRepo struct {
	notFound bool
}

func (m *mockIdentityRepo) Create(ctx context.Context, identity *identity.Identity) error {
	return nil
}

func (m *mockIdentityRepo) GetByProviderAndEmail(ctx context.Context, provider identity.Provider, email string) (*identity.Identity, error) {
	if m.notFound {
		return nil, identity.ErrIdentityNotFound
	}
	return &identity.Identity{UserID: "user-1", Email: email, Provider: identity.ProviderPassword}, nil
}

func (m *mockIdentityRepo) GetByUserID(ctx context.Context, userID string) ([]*identity.Identity, error) {
	return nil, nil
}

func (m *mockIdentityRepo) UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error {
	return nil
}

func (m *mockIdentityRepo) UpdateVerifiedAt(ctx context.Context, id string, verifiedAt *time.Time) error {
	return nil
}
