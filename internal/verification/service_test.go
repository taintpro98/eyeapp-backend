package verification

import (
	"context"
	"testing"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/pkg/email"
	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/internal/repositories"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

func testConfig() *config.Config {
	return &config.Config{
		EmailVerificationTTL: 24 * time.Hour,
		AppVerifyURLBase:     "http://localhost:5173/verify-email",
	}
}

func TestVerifyToken_EmptyToken(t *testing.T) {
	svc := NewService(testConfig(), logger.NewNop(), &mockRepo{}, nil, nil)

	err := svc.VerifyToken(context.Background(), "")
	if err != ErrInvalidVerificationToken {
		t.Errorf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerifyToken_NotFound(t *testing.T) {
	svc := NewService(testConfig(), logger.NewNop(), &mockRepo{notFound: true}, nil, nil)

	err := svc.VerifyToken(context.Background(), "any-token")
	if err != ErrInvalidVerificationToken {
		t.Errorf("expected ErrInvalidVerificationToken for unknown token, got %v", err)
	}
}

type mockRepo struct {
	notFound bool
}

func (m *mockRepo) Create(ctx context.Context, token *models.VerificationToken) error {
	return nil
}

func (m *mockRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*models.VerificationToken, error) {
	if m.notFound {
		return nil, repositories.ErrVerificationTokenNotFound
	}
	return &models.VerificationToken{ID: "1", UserID: "u1", TokenHash: tokenHash, ExpiresAt: time.Now().Add(time.Hour)}, nil
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
	cfg := testConfig()
	cfg.AppVerifyURLBase = "http://localhost:3000"
	svc := NewService(cfg, logger.NewNop(), &mockRepo{}, &mockIdentityRepo{notFound: true}, &email.NoopSender{})

	err := svc.ResendVerification(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Errorf("resend should return nil when not found (no leak), got %v", err)
	}
}

type mockIdentityRepo struct {
	notFound bool
}

func (m *mockIdentityRepo) Create(ctx context.Context, identity *models.Identity) error {
	return nil
}

func (m *mockIdentityRepo) GetByProviderAndEmail(ctx context.Context, provider models.IdentityProvider, email string) (*models.Identity, error) {
	if m.notFound {
		return nil, repositories.ErrIdentityNotFound
	}
	return &models.Identity{UserID: "user-1", Email: email, Provider: models.IdentityProviderPassword}, nil
}

func (m *mockIdentityRepo) GetByUserID(ctx context.Context, userID string) ([]*models.Identity, error) {
	return nil, nil
}

func (m *mockIdentityRepo) UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error {
	return nil
}

func (m *mockIdentityRepo) UpdateVerifiedAt(ctx context.Context, id string, verifiedAt *time.Time) error {
	return nil
}
