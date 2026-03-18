package verification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alumieye/eyeapp-backend/pkg/db"
)

var (
	ErrTokenNotFound = errors.New("verification token not found")
)

// Repository manages email verification tokens
type Repository interface {
	Create(ctx context.Context, token *Token) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*Token, error)
	MarkConsumed(ctx context.Context, id string) error
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
	db *db.DB
}

// NewRepository creates a new verification repository
func NewRepository(database *db.DB) Repository {
	return &PostgresRepository{db: database}
}

// Create stores a new verification token
func (r *PostgresRepository) Create(ctx context.Context, token *Token) error {
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

// GetByTokenHash finds a token by its hash
func (r *PostgresRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*Token, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
	`
	token := &Token{}
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
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return token, nil
}

// MarkConsumed marks a token as consumed
func (r *PostgresRepository) MarkConsumed(ctx context.Context, id string) error {
	query := `
		UPDATE email_verification_tokens
		SET consumed_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
