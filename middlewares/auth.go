package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/internal/auth"
)

// Auth returns JWT authentication middleware
func Auth(tokenService *auth.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				apierrors.Unauthorized(w, "Missing authorization header")
				return
			}

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

			userID, err := tokenService.ValidateAccessToken(token)
			if err != nil {
				switch err {
				case auth.ErrExpiredToken:
					apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeUnauthorized, "Token has expired")
				case auth.ErrInvalidTokenType:
					apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeUnauthorized, "Invalid token type")
				default:
					apierrors.Unauthorized(w, "Invalid token")
				}
				return
			}

			ctx := context.WithValue(r.Context(), auth.UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
