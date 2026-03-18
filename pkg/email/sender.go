package email

import "context"

type Sender interface {
	SendVerificationEmail(ctx context.Context, to string, verifyURL string) error
}
