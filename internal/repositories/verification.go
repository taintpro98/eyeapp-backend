package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/models"
	"github.com/alumieye/eyeapp-backend/pkg/db"
)

var (
	ErrVerificationTokenNotFound = errors.New("verification token not found")
)

type VerificationRepository interface {
	Create(ctx context.Context, token *models.VerificationToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.VerificationToken, error)
	MarkConsumed(ctx context.Context, id string) error
}

type verificationPostgres struct {
	db *db.DB
}

func NewVerificationRepository(database *db.DB) VerificationRepository {
	return &verificationPostgres{db: database}
}

func (r *verificationPostgres) Create(ctx context.Context, token *models.VerificationToken) error {
	query := `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	now := time.Now()
	token.CreatedAt = now

	return r.db.QueryRowContext(ctx, query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
	).Scan(&token.ID)
}

func (r *verificationPostgres) GetByTokenHash(ctx context.Context, tokenHash string) (*models.VerificationToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
	`
	token := &models.VerificationToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.ConsumedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVerificationTokenNotFound
		}
		return nil, err
	}
	return token, nil
}

func (r *verificationPostgres) MarkConsumed(ctx context.Context, id string) error {
	query := `
		UPDATE email_verification_tokens
		SET consumed_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
