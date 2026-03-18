package verification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/email"
	"github.com/alumieye/eyeapp-backend/internal/identity"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

var (
	ErrInvalidVerificationToken  = errors.New("invalid verification token")
	ErrVerificationTokenExpired  = errors.New("verification token expired")
	ErrTokenAlreadyConsumed      = errors.New("verification token already used")
)

// Service handles email verification logic
type Service struct {
	log           logger.Logger
	repo          Repository
	identityRepo  identity.Repository
	emailSender   email.Sender
	ttl           time.Duration
	verifyURLBase string
}

// NewService creates a new verification service
func NewService(
	log logger.Logger,
	repo Repository,
	identityRepo identity.Repository,
	emailSender email.Sender,
	ttl time.Duration,
	verifyURLBase string,
) *Service {
	return &Service{
		log:           log,
		repo:          repo,
		identityRepo:  identityRepo,
		emailSender:   emailSender,
		ttl:           ttl,
		verifyURLBase: strings.TrimSuffix(verifyURLBase, "/"),
	}
}

// VerifyToken validates a raw token and marks the identity as verified
func (s *Service) VerifyToken(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return ErrInvalidVerificationToken
	}

	tokenHash := crypto.HashToken(rawToken)

	token, err := s.repo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return ErrInvalidVerificationToken
		}
		return err
	}

	if token.IsConsumed() {
		return ErrTokenAlreadyConsumed
	}

	if token.IsExpired() {
		return ErrVerificationTokenExpired
	}

	// Mark token as consumed
	if err := s.repo.MarkConsumed(ctx, token.ID); err != nil {
		return err
	}

	// Mark password identity as verified
	identities, err := s.identityRepo.GetByUserID(ctx, token.UserID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, ident := range identities {
		if ident.Provider == identity.ProviderPassword {
			return s.identityRepo.UpdateVerifiedAt(ctx, ident.ID, &now)
		}
	}

	return nil
}

// CreateAndSendToken creates a verification token and sends the email
func (s *Service) CreateAndSendToken(ctx context.Context, userID, emailAddr string) error {
	rawToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return err
	}

	tokenHash := crypto.HashToken(rawToken)
	expiresAt := time.Now().Add(s.ttl)

	token := &Token{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Create(ctx, token); err != nil {
		return err
	}

	verifyURL := fmt.Sprintf("%s?token=%s", s.verifyURLBase, rawToken)

	if s.emailSender != nil {
		if err := s.emailSender.SendVerificationEmail(ctx, emailAddr, verifyURL); err != nil {
			// Log but don't fail - token is stored, user can resend
			return err
		}
	}

	return nil
}

// ResendVerification sends a new verification email if the account exists and is not verified
// Returns nil in all cases to avoid leaking account existence
func (s *Service) ResendVerification(ctx context.Context, emailAddr string) error {
	emailAddr = normalizeEmail(emailAddr)

	ident, err := s.identityRepo.GetByProviderAndEmail(ctx, identity.ProviderPassword, emailAddr)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return nil // Don't leak
		}
		return err
	}

	if ident.VerifiedAt != nil {
		return nil // Already verified, generic success
	}

	return s.CreateAndSendToken(ctx, ident.UserID, emailAddr)
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}
