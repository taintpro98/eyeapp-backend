package models

import "time"

// VerificationToken represents an email verification token record
type VerificationToken struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// IsConsumed returns true if the token has been used
func (t *VerificationToken) IsConsumed() bool {
	return t.ConsumedAt != nil
}

// IsExpired returns true if the token has expired
func (t *VerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
