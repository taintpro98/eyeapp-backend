package email

import "context"

type NoopSender struct{}

func (NoopSender) SendVerificationEmail(ctx context.Context, to string, verifyURL string) error {
	return nil
}
