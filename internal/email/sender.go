package email

// Sender sends transactional emails
type Sender interface {
	SendVerificationEmail(to string, verifyURL string) error
}
