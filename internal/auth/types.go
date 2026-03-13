package auth

import "github.com/alumieye/eyeapp-backend/internal/user"

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Email       string `json:"email" example:"user@example.com"`
	Password    string `json:"password" example:"strong_password123"`
	DisplayName string `json:"display_name" example:"John Doe"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"strong_password123"`
}

// RefreshRequest represents the refresh token request payload
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."`
}

// LogoutRequest represents the logout request payload
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."`
}

// TokensResponse represents the tokens in an auth response
type TokensResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."`
	ExpiresIn    int    `json:"expires_in" example:"900"`
}

// AuthResponse represents the response for auth endpoints
type AuthResponse struct {
	User   *user.UserResponse `json:"user"`
	Tokens *TokensResponse    `json:"tokens"`
}

// MeResponse represents the response for the /me endpoint
type MeResponse struct {
	User *user.UserResponse `json:"user"`
}

// RequestContext holds request-specific information
type RequestContext struct {
	UserAgent string
	IPAddress string
}
