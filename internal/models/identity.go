package models

import "time"

type IdentityProvider string

const (
	IdentityProviderPassword IdentityProvider = "password"
	IdentityProviderGoogle   IdentityProvider = "google" // For future use
)

type Identity struct {
	ID             string           `json:"id"`
	UserID         string           `json:"user_id"`
	Provider       IdentityProvider `json:"provider"`
	ProviderUserID *string          `json:"provider_user_id,omitempty"`
	Email          string           `json:"email"`
	PasswordHash   *string          `json:"-"`
	VerifiedAt     *time.Time       `json:"verified_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}
