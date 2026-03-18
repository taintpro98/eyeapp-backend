package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

const resendAPIURL = "https://api.resend.com/emails"

// ResendSender sends emails via Resend API
type ResendSender struct {
	apiKey string
	from   string
	log    logger.Logger
}

// NewResendSender creates a new Resend email sender
func NewResendSender(log logger.Logger, apiKey, from string) *ResendSender {
	return &ResendSender{
		apiKey: apiKey,
		from:   strings.TrimSpace(from),
		log:    log,
	}
}

// SendVerificationEmail sends an email verification link
func (s *ResendSender) SendVerificationEmail(ctx context.Context, to string, verifyURL string) error {
	s.log.Info(ctx, "Sending verification email", logger.Str("to", to))
	if s.apiKey == "" {
		return fmt.Errorf("resend: API key not configured")
	}

	subject := "Verify your ALumiEye email"
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2>Verify your email</h2>
  <p>Thanks for signing up for ALumiEye. Please click the link below to verify your email address:</p>
  <p><a href="%s" style="display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px;">Verify email</a></p>
  <p>Or copy and paste this URL into your browser:</p>
  <p style="word-break: break-all; font-size: 12px;">%s</p>
  <p style="font-size: 14px; color: #666;">This link expires in 24 hours. If you did not sign up for ALumiEye, you can safely ignore this email.</p>
</body>
</html>`, verifyURL, verifyURL)

	payload := map[string]interface{}{
		"from":    s.from,
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		s.log.Error(ctx, "Failed to marshal email payload", logger.Err(err))
		return fmt.Errorf("resend: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendAPIURL, bytes.NewReader(body))
	if err != nil {
		s.log.Error(ctx, "Failed to create Resend request", logger.Err(err))
		return fmt.Errorf("resend: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Error(ctx, "Failed to send verification email", logger.Err(err), logger.Str("to", to))
		return fmt.Errorf("resend: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Error(ctx, "Resend API returned error status", logger.Int("status", resp.StatusCode), logger.Str("to", to))
		return fmt.Errorf("resend: API returned status %d", resp.StatusCode)
	}

	s.log.Info(ctx, "Verification email sent successfully", logger.Str("to", to))
	return nil
}
