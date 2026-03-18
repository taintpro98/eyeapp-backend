package email

import "context"

// NoopSender is a no-op email sender for when email is not configured (e.g. tests)
type NoopSender struct{}

// SendVerificationEmail does nothing
func (NoopSender) SendVerificationEmail(ctx context.Context, to string, verifyURL string) error {
	return nil
}
