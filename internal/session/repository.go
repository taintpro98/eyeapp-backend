package session

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alumieye/eyeapp-backend/pkg/db"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type Repository interface {
	Create(ctx context.Context, session *Session) error
	GetByRefreshTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	GetByUserID(ctx context.Context, userID string) ([]*Session, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	UpdateLastUsed(ctx context.Context, id string) error
	UpdateRefreshToken(ctx context.Context, id string, newTokenHash string, newExpiresAt time.Time) error
}

type PostgresRepository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &PostgresRepository{db: database}
}

func (r *PostgresRepository) Create(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO auth_sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	now := time.Now()
	session.CreatedAt = now
	session.LastUsedAt = now

	err := r.db.QueryRowContext(ctx, query,
		session.UserID,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		session.CreatedAt,
		session.LastUsedAt,
	).Scan(&session.ID)

	return err
}

func (r *PostgresRepository) GetByRefreshTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at, last_used_at
		FROM auth_sessions
		WHERE refresh_token_hash = $1
	`
	session := &Session{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return session, nil
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at, last_used_at
		FROM auth_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&session.UserAgent,
			&session.IPAddress,
			&session.ExpiresAt,
			&session.RevokedAt,
			&session.CreatedAt,
			&session.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *PostgresRepository) Revoke(ctx context.Context, id string) error {
	query := `UPDATE auth_sessions SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `UPDATE auth_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE auth_sessions SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresRepository) UpdateRefreshToken(ctx context.Context, id string, newTokenHash string, newExpiresAt time.Time) error {
	query := `UPDATE auth_sessions SET refresh_token_hash = $1, expires_at = $2, last_used_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, newTokenHash, newExpiresAt, id)
	return err
}
