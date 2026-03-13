package identity

import (
	"time"
)

type Provider string

const (
	ProviderPassword Provider = "password"
	ProviderGoogle   Provider = "google" // For future use
)

type Identity struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Provider       Provider   `json:"provider"`
	ProviderUserID *string    `json:"provider_user_id,omitempty"`
	Email          string     `json:"email"`
	PasswordHash   *string    `json:"-"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
