package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	"github.com/alumieye/eyeapp-backend/internal/repositories"
	"github.com/alumieye/eyeapp-backend/internal/verification"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
	"github.com/lib/pq"
)

var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrEmailNotVerified     = errors.New("email not verified")
	ErrUserBlocked          = errors.New("user account is blocked")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrSessionExpired       = errors.New("session has expired")
	ErrSessionRevoked       = errors.New("session has been revoked")
	ErrValidationFailed     = errors.New("validation failed")
)

type Service struct {
	userRepo        repositories.UserRepository
	identityRepo    repositories.IdentityRepository
	sessionRepo     repositories.SessionRepository
	tokenService    *TokenService
	verificationSvc *verification.Service
	cfg             *config.Config
	log             logger.Logger
}

func NewService(
	cfg *config.Config,
	log logger.Logger,
	userRepo repositories.UserRepository,
	identityRepo repositories.IdentityRepository,
	sessionRepo repositories.SessionRepository,
	tokenService *TokenService,
	verificationSvc *verification.Service,
) *Service {
	return &Service{
		userRepo:        userRepo,
		identityRepo:    identityRepo,
		sessionRepo:     sessionRepo,
		tokenService:    tokenService,
		verificationSvc: verificationSvc,
		cfg:             cfg,
		log:             log,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, _ RequestContext) (*RegisterResponse, error) {
	s.log.Info(ctx, "Register", logger.Str("service", "auth"), logger.Str("email", req.Email), logger.Str("display_name", req.DisplayName))

	emailAddr := normalizeEmail(req.Email)

	exists, err := s.userRepo.EmailExists(ctx, emailAddr)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	passwordHash, err := crypto.HashPassword(req.Password, nil)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
		Email:       emailAddr,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Status:      models.UserStatusActive,
		Role:        models.UserRoleUser,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	ident := &models.Identity{
		UserID:       newUser.ID,
		Provider:     models.IdentityProviderPassword,
		Email:        emailAddr,
		PasswordHash: &passwordHash,
	}
	if err := s.identityRepo.Create(ctx, ident); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	_ = s.verificationSvc.CreateAndSendToken(ctx, newUser.ID, emailAddr)

	return &RegisterResponse{
		Message: "Registration successful. Please verify your email before logging in.",
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, reqCtx RequestContext) (*AuthResponse, error) {
	s.log.Info(ctx, "Login", logger.Str("service", "auth"), logger.Str("email", req.Email))

	email := normalizeEmail(req.Email)

	ident, err := s.identityRepo.GetByProviderAndEmail(ctx, models.IdentityProviderPassword, email)
	if err != nil {
		if errors.Is(err, repositories.ErrIdentityNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if ident.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	valid, err := crypto.VerifyPassword(req.Password, *ident.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	if ident.VerifiedAt == nil {
		return nil, ErrEmailNotVerified
	}

	usr, err := s.userRepo.GetByID(ctx, ident.UserID)
	if err != nil {
		return nil, err
	}

	// Check user status
	if usr.Status == models.UserStatusBlocked {
		return nil, ErrUserBlocked
	}

	now := time.Now()
	if err := s.userRepo.UpdateLastLogin(ctx, usr.ID, now); err != nil {
		return nil, err
	}

	return s.createSessionAndTokens(ctx, usr, reqCtx)
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest, reqCtx RequestContext) (*AuthResponse, error) {
	s.log.Info(ctx, "Refresh", logger.Str("service", "auth"), logger.Bool("refresh_token_present", req.RefreshToken != ""))

	tokenHash := crypto.HashToken(req.RefreshToken)

	sess, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repositories.ErrSessionNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if sess.IsRevoked() {
		return nil, ErrSessionRevoked
	}
	if sess.IsExpired() {
		return nil, ErrSessionExpired
	}

	usr, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}

	if usr.Status == models.UserStatusBlocked {
		return nil, ErrUserBlocked
	}

	newRefreshToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}
	newTokenHash := crypto.HashToken(newRefreshToken)
	newExpiresAt := time.Now().Add(s.cfg.RefreshTokenTTL)

	if err := s.sessionRepo.UpdateRefreshToken(ctx, sess.ID, newTokenHash, newExpiresAt); err != nil {
		return nil, err
	}

	accessToken, err := s.tokenService.GenerateAccessToken(usr.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User: usr.ToResponse(),
		Tokens: &TokensResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    s.tokenService.GetAccessTokenTTL(),
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, req LogoutRequest) error {
	s.log.Info(ctx, "Logout", logger.Str("service", "auth"), logger.Bool("refresh_token_present", req.RefreshToken != ""))

	if req.RefreshToken == "" {
		return nil
	}

	tokenHash := crypto.HashToken(req.RefreshToken)

	sess, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repositories.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	return s.sessionRepo.Revoke(ctx, sess.ID)
}

func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*models.User, error) {
	s.log.Info(ctx, "GetCurrentUser", logger.Str("service", "auth"), logger.Str("user_id", userID))

	return s.userRepo.GetByID(ctx, userID)
}

func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	s.log.Info(ctx, "VerifyEmail", logger.Str("service", "auth"), logger.Bool("token_present", rawToken != ""))

	return s.verificationSvc.VerifyToken(ctx, rawToken)
}

func (s *Service) ResendVerificationEmail(ctx context.Context, emailAddr string) error {
	s.log.Info(ctx, "ResendVerificationEmail", logger.Str("service", "auth"), logger.Str("email", emailAddr))

	return s.verificationSvc.ResendVerification(ctx, emailAddr)
}

func (s *Service) createSessionAndTokens(ctx context.Context, usr *models.User, reqCtx RequestContext) (*AuthResponse, error) {
	const platform = "web"

	refreshToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	tokenHash := crypto.HashToken(refreshToken)

	sess := &models.Session{
		UserID:           usr.ID,
		RefreshTokenHash: tokenHash,
		Platform:         platform,
		ExpiresAt:        time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if reqCtx.UserAgent != "" {
		sess.UserAgent = &reqCtx.UserAgent
	}
	if reqCtx.IPAddress != "" {
		sess.IPAddress = &reqCtx.IPAddress
	}

	tx, err := s.sessionRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.RevokeActiveSessionsByPlatform(ctx, usr.ID, platform); err != nil {
		return nil, err
	}
	if err := tx.Create(ctx, sess); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	accessToken, err := s.tokenService.GenerateAccessToken(usr.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User: usr.ToResponse(),
		Tokens: &TokensResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    s.tokenService.GetAccessTokenTTL(),
		},
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

