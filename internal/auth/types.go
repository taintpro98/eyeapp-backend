package auth

import (
	"errors"
	"strings"

	"github.com/alumieye/eyeapp-backend/internal/models"
)

type RegisterRequest struct {
	Email       string `json:"email" example:"user@example.com"`
	Password    string `json:"password" example:"strong_password123"`
	DisplayName string `json:"display_name" example:"John Doe"`
}

func (r RegisterRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if !isValidEmail(r.Email) {
		return errors.New("invalid email format")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"strong_password123"`
}

func (r LoginRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."`
}

func (r RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return errors.New("refresh token is required")
	}
	return nil
}

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
	User   *models.UserResponse `json:"user"`
	Tokens *TokensResponse    `json:"tokens"`
}

// RegisterResponse represents the response for registration (no tokens, verify email first)
type RegisterResponse struct {
	Message string `json:"message"`
}

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

func (r VerifyEmailRequest) Validate() error {
	if r.Token == "" {
		return errors.New("token is required")
	}
	return nil
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

func (r ResendVerificationRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

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

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

// MeResponse represents the response for the /me endpoint
type MeResponse struct {
	User *models.UserResponse `json:"user"`
}

// RequestContext holds request-specific information
type RequestContext struct {
	UserAgent string
	IPAddress string
}
