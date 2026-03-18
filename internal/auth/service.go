package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/identity"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	"github.com/alumieye/eyeapp-backend/internal/session"
	"github.com/alumieye/eyeapp-backend/internal/user"
	"github.com/alumieye/eyeapp-backend/internal/verification"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
	"github.com/alumieye/eyeapp-backend/pkg/trace"
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

// Service handles authentication business logic
type Service struct {
	userRepo         user.Repository
	identityRepo     identity.Repository
	sessionRepo      session.Repository
	tokenService     *TokenService
	verificationSvc   *verification.Service
	refreshTokenTTL  time.Duration
	log              *logger.Logger
}

// NewService creates a new auth service
func NewService(
	log *logger.Logger,
	userRepo user.Repository,
	identityRepo identity.Repository,
	sessionRepo session.Repository,
	tokenService *TokenService,
	verificationSvc *verification.Service,
	refreshTokenTTL time.Duration,
) *Service {
	return &Service{
		userRepo:        userRepo,
		identityRepo:    identityRepo,
		sessionRepo:     sessionRepo,
		tokenService:    tokenService,
		verificationSvc: verificationSvc,
		refreshTokenTTL: refreshTokenTTL,
		log:             log,
	}
}

// Register creates a new user account with email/password.
// Does not auto-login; user must verify email first.
func (s *Service) Register(ctx context.Context, req *RegisterRequest, _ *RequestContext) (*RegisterResponse, error) {
	s.logInput(ctx, "Register", "email", req.Email, "display_name", req.DisplayName)

	// Validate input
	if err := s.validateRegisterRequest(req); err != nil {
		return nil, err
	}

	// Normalize email
	emailAddr := normalizeEmail(req.Email)

	// Check if email already exists
	exists, err := s.userRepo.EmailExists(ctx, emailAddr)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	passwordHash, err := crypto.HashPassword(req.Password, nil)
	if err != nil {
		return nil, err
	}

	// Create user
	newUser := &user.User{
		Email:       emailAddr,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Status:      user.StatusActive,
		Role:        user.RoleUser,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	// Create password identity (verified_at = null for new users)
	ident := &identity.Identity{
		UserID:       newUser.ID,
		Provider:     identity.ProviderPassword,
		Email:        emailAddr,
		PasswordHash: &passwordHash,
	}
	if err := s.identityRepo.Create(ctx, ident); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	// Generate verification token and send email
	_ = s.verificationSvc.CreateAndSendToken(ctx, newUser.ID, emailAddr)

	return &RegisterResponse{
		Message: "Registration successful. Please verify your email before logging in.",
	}, nil
}

// Login authenticates a user with email/password
func (s *Service) Login(ctx context.Context, req *LoginRequest, reqCtx *RequestContext) (*AuthResponse, error) {
	s.logInput(ctx, "Login", "email", req.Email)

	// Validate input
	if err := s.validateLoginRequest(req); err != nil {
		return nil, err
	}

	// Normalize email
	email := normalizeEmail(req.Email)

	// Find password identity
	ident, err := s.identityRepo.GetByProviderAndEmail(ctx, identity.ProviderPassword, email)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	if ident.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	valid, err := crypto.VerifyPassword(req.Password, *ident.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	// Require email verification
	if ident.VerifiedAt == nil {
		return nil, ErrEmailNotVerified
	}

	// Get user
	usr, err := s.userRepo.GetByID(ctx, ident.UserID)
	if err != nil {
		return nil, err
	}

	// Check user status
	if usr.Status == user.StatusBlocked {
		return nil, ErrUserBlocked
	}

	// Update last login
	now := time.Now()
	if err := s.userRepo.UpdateLastLogin(ctx, usr.ID, now); err != nil {
		return nil, err
	}

	// Create session and generate tokens
	return s.createSessionAndTokens(ctx, usr, reqCtx)
}

// Refresh validates a refresh token and issues new tokens
func (s *Service) Refresh(ctx context.Context, req *RefreshRequest, reqCtx *RequestContext) (*AuthResponse, error) {
	s.logInput(ctx, "Refresh", "refresh_token_present", req.RefreshToken != "")

	if req.RefreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	// Hash the refresh token
	tokenHash := crypto.HashToken(req.RefreshToken)

	// Find session by token hash
	sess, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	// Validate session
	if sess.IsRevoked() {
		return nil, ErrSessionRevoked
	}
	if sess.IsExpired() {
		return nil, ErrSessionExpired
	}

	// Get user
	usr, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}

	// Check user status
	if usr.Status == user.StatusBlocked {
		return nil, ErrUserBlocked
	}

	// Implement refresh token rotation: generate new refresh token
	newRefreshToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}
	newTokenHash := crypto.HashToken(newRefreshToken)
	newExpiresAt := time.Now().Add(s.refreshTokenTTL)

	// Update session with new refresh token
	if err := s.sessionRepo.UpdateRefreshToken(ctx, sess.ID, newTokenHash, newExpiresAt); err != nil {
		return nil, err
	}

	// Generate new access token
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

// Logout revokes a session
func (s *Service) Logout(ctx context.Context, req *LogoutRequest) error {
	s.logInput(ctx, "Logout", "refresh_token_present", req.RefreshToken != "")

	if req.RefreshToken == "" {
		return nil // No-op if no token provided
	}

	// Hash the refresh token
	tokenHash := crypto.HashToken(req.RefreshToken)

	// Find session
	sess, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil // Already logged out or invalid token
		}
		return err
	}

	// Revoke the session
	return s.sessionRepo.Revoke(ctx, sess.ID)
}

// GetCurrentUser returns the user for the given user ID
func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*user.User, error) {
	s.logInput(ctx, "GetCurrentUser", "user_id", userID)

	return s.userRepo.GetByID(ctx, userID)
}

// VerifyEmail validates a verification token and marks the identity as verified
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	s.logInput(ctx, "VerifyEmail", "token_present", rawToken != "")

	return s.verificationSvc.VerifyToken(ctx, rawToken)
}

// ResendVerificationEmail sends a new verification email if the account exists and is not verified
func (s *Service) ResendVerificationEmail(ctx context.Context, emailAddr string) error {
	s.logInput(ctx, "ResendVerificationEmail", "email", emailAddr)

	return s.verificationSvc.ResendVerification(ctx, emailAddr)
}

// createSessionAndTokens creates a new session and returns auth tokens
func (s *Service) createSessionAndTokens(ctx context.Context, usr *user.User, reqCtx *RequestContext) (*AuthResponse, error) {
	// Generate refresh token
	refreshToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	// Hash refresh token for storage
	tokenHash := crypto.HashToken(refreshToken)

	// Create session
	sess := &session.Session{
		UserID:           usr.ID,
		RefreshTokenHash: tokenHash,
		ExpiresAt:        time.Now().Add(s.refreshTokenTTL),
	}
	if reqCtx != nil {
		if reqCtx.UserAgent != "" {
			sess.UserAgent = &reqCtx.UserAgent
		}
		if reqCtx.IPAddress != "" {
			sess.IPAddress = &reqCtx.IPAddress
		}
	}

	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, err
	}

	// Generate access token
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

// validateRegisterRequest validates the registration request
func (s *Service) validateRegisterRequest(req *RegisterRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if !isValidEmail(req.Email) {
		return errors.New("invalid email format")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

// validateLoginRequest validates the login request
func (s *Service) validateLoginRequest(req *LoginRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// normalizeEmail normalizes an email address
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// logInput logs service input at debug level with trace_id. No-op if log is nil (e.g. in tests).
func (s *Service) logInput(ctx context.Context, method string, kv ...interface{}) {
	if s.log == nil {
		return
	}
	ev := s.log.Debug().Str("service", "auth").Str("method", method)
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		ev = ev.Str("trace_id", traceID)
	}
	for i := 0; i < len(kv)-1; i += 2 {
		if key, ok := kv[i].(string); ok {
			ev = ev.Interface(key, kv[i+1])
		}
	}
	ev.Msg("service input")
}

// isUniqueViolation returns true if the error is a PostgreSQL unique constraint violation
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	atIndex := strings.Index(email, "@")
	if atIndex < 1 || atIndex >= len(email)-1 {
		return false
	}
	dotIndex := strings.LastIndex(email[atIndex:], ".")
	return dotIndex > 1 && dotIndex < len(email[atIndex:])-1
}
