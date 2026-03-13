package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
)

// Middleware provides JWT authentication middleware
type Middleware struct {
	tokenService *TokenService
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(tokenService *TokenService) *Middleware {
	return &Middleware{
		tokenService: tokenService,
	}
}

// Authenticate is middleware that validates JWT access tokens
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			apierrors.Unauthorized(w, "Missing authorization header")
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			apierrors.Unauthorized(w, "Invalid authorization header format")
			return
		}

		token := parts[1]
		if token == "" {
			apierrors.Unauthorized(w, "Missing token")
			return
		}

		// Validate token
		userID, err := m.tokenService.ValidateAccessToken(token)
		if err != nil {
			switch err {
			case ErrExpiredToken:
				apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeUnauthorized, "Token has expired")
			case ErrInvalidTokenType:
				apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeUnauthorized, "Invalid token type")
			default:
				apierrors.Unauthorized(w, "Invalid token")
			}
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}
