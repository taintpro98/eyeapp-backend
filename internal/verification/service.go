package verification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/pkg/email"
	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/internal/repositories"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

var (
	ErrInvalidVerificationToken  = errors.New("invalid verification token")
	ErrVerificationTokenExpired  = errors.New("verification token expired")
	ErrTokenAlreadyConsumed      = errors.New("verification token already used")
)

type Service struct {
	log           logger.Logger
	repo          repositories.VerificationRepository
	identityRepo  repositories.IdentityRepository
	emailSender   email.Sender
	cfg           *config.Config
}

func NewService(
	cfg *config.Config,
	log logger.Logger,
	repo repositories.VerificationRepository,
	identityRepo repositories.IdentityRepository,
	emailSender email.Sender,
) *Service {
	return &Service{
		log:          log,
		repo:         repo,
		identityRepo: identityRepo,
		emailSender:  emailSender,
		cfg:          cfg,
	}
}

func (s *Service) VerifyToken(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return ErrInvalidVerificationToken
	}

	tokenHash := crypto.HashToken(rawToken)

	token, err := s.repo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repositories.ErrVerificationTokenNotFound) {
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

	if err := s.repo.MarkConsumed(ctx, token.ID); err != nil {
		return err
	}

	identities, err := s.identityRepo.GetByUserID(ctx, token.UserID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, ident := range identities {
		if ident.Provider == models.IdentityProviderPassword {
			return s.identityRepo.UpdateVerifiedAt(ctx, ident.ID, &now)
		}
	}

	return nil
}

func (s *Service) CreateAndSendToken(ctx context.Context, userID, emailAddr string) error {
	rawToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return err
	}

	tokenHash := crypto.HashToken(rawToken)
	expiresAt := time.Now().Add(s.cfg.EmailVerificationTTL)

	token := &models.VerificationToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.Create(ctx, token); err != nil {
		return err
	}

	verifyURL := fmt.Sprintf("%s?token=%s", strings.TrimSuffix(s.cfg.AppVerifyURLBase, "/"), rawToken)

	if err := s.emailSender.SendVerificationEmail(ctx, emailAddr, verifyURL); err != nil {
		return err
	}

	return nil
}

func (s *Service) ResendVerification(ctx context.Context, emailAddr string) error {
	emailAddr = normalizeEmail(emailAddr)

	ident, err := s.identityRepo.GetByProviderAndEmail(ctx, models.IdentityProviderPassword, emailAddr)
	if err != nil {
		if errors.Is(err, repositories.ErrIdentityNotFound) {
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
