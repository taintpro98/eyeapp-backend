package verification

import "time"

// Token represents an email verification token record
type Token struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// IsConsumed returns true if the token has been used
func (t *Token) IsConsumed() bool {
	return t.ConsumedAt != nil
}

// IsExpired returns true if the token has expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
