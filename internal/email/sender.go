package email

import "context"

// Sender sends transactional emails
type Sender interface {
	SendVerificationEmail(ctx context.Context, to string, verifyURL string) error
}
