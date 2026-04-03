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
	ErrSessionNotFound = errors.New("session not found")
)

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByRefreshTokenHash(ctx context.Context, tokenHash string) (*models.Session, error)
	GetByUserID(ctx context.Context, userID string) ([]*models.Session, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	RevokeActiveSessionsByPlatform(ctx context.Context, userID string, platform string) error
	UpdateLastUsed(ctx context.Context, id string) error
	UpdateRefreshToken(ctx context.Context, id string, newTokenHash string, newExpiresAt time.Time) error
	BeginTx(ctx context.Context) (*SessionTx, error)
}

// SessionTx wraps a database transaction for atomic session operations.
type SessionTx struct {
	tx *sql.Tx
}

func (t *SessionTx) RevokeActiveSessionsByPlatform(ctx context.Context, userID string, platform string) error {
	query := `
		UPDATE auth_sessions
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND platform = $2
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`
	_, err := t.tx.ExecContext(ctx, query, userID, platform)
	return err
}

func (t *SessionTx) Create(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO auth_sessions (user_id, refresh_token_hash, platform, user_agent, ip_address, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	now := time.Now()
	session.CreatedAt = now
	session.LastUsedAt = now

	return t.tx.QueryRowContext(ctx, query,
		session.UserID,
		session.RefreshTokenHash,
		session.Platform,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		session.CreatedAt,
		session.LastUsedAt,
	).Scan(&session.ID)
}

func (t *SessionTx) Commit() error {
	return t.tx.Commit()
}

func (t *SessionTx) Rollback() error {
	return t.tx.Rollback()
}

type sessionPostgres struct {
	db *db.DB
}

func NewSessionRepository(database *db.DB) SessionRepository {
	return &sessionPostgres{db: database}
}

func (r *sessionPostgres) Create(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO auth_sessions (user_id, refresh_token_hash, platform, user_agent, ip_address, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	now := time.Now()
	session.CreatedAt = now
	session.LastUsedAt = now

	err := r.db.QueryRowContext(ctx, query,
		session.UserID,
		session.RefreshTokenHash,
		session.Platform,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		session.CreatedAt,
		session.LastUsedAt,
	).Scan(&session.ID)

	return err
}

func (r *sessionPostgres) GetByRefreshTokenHash(ctx context.Context, tokenHash string) (*models.Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, platform, user_agent, ip_address, expires_at, revoked_at, created_at, last_used_at
		FROM auth_sessions
		WHERE refresh_token_hash = $1
	`
	session := &models.Session{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.Platform,
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

func (r *sessionPostgres) GetByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, platform, user_agent, ip_address, expires_at, revoked_at, created_at, last_used_at
		FROM auth_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&session.Platform,
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

func (r *sessionPostgres) Revoke(ctx context.Context, id string) error {
	query := `UPDATE auth_sessions SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *sessionPostgres) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `UPDATE auth_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// RevokeActiveSessionsByPlatform revokes all active sessions for the given user and platform.
// Used to enforce single-session-per-platform before creating a new session.
func (r *sessionPostgres) RevokeActiveSessionsByPlatform(ctx context.Context, userID string, platform string) error {
	query := `
		UPDATE auth_sessions
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND platform = $2
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`
	_, err := r.db.ExecContext(ctx, query, userID, platform)
	return err
}

func (r *sessionPostgres) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE auth_sessions SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *sessionPostgres) UpdateRefreshToken(ctx context.Context, id string, newTokenHash string, newExpiresAt time.Time) error {
	query := `UPDATE auth_sessions SET refresh_token_hash = $1, expires_at = $2, last_used_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, newTokenHash, newExpiresAt, id)
	return err
}

func (r *sessionPostgres) BeginTx(ctx context.Context) (*SessionTx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &SessionTx{tx: tx}, nil
}
