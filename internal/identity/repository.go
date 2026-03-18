package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alumieye/eyeapp-backend/pkg/db"
)

var (
	ErrIdentityNotFound = errors.New("identity not found")
)

type Repository interface {
	Create(ctx context.Context, identity *Identity) error
	GetByProviderAndEmail(ctx context.Context, provider Provider, email string) (*Identity, error)
	GetByUserID(ctx context.Context, userID string) ([]*Identity, error)
	UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error
	UpdateVerifiedAt(ctx context.Context, id string, verifiedAt *time.Time) error
}

type PostgresRepository struct {
	db *db.DB
}

func NewRepository(database *db.DB) Repository {
	return &PostgresRepository{db: database}
}

func (r *PostgresRepository) Create(ctx context.Context, identity *Identity) error {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_user_id, email, password_hash, verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	now := time.Now()
	identity.CreatedAt = now
	identity.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		identity.UserID,
		identity.Provider,
		identity.ProviderUserID,
		identity.Email,
		identity.PasswordHash,
		identity.VerifiedAt,
		identity.CreatedAt,
		identity.UpdatedAt,
	).Scan(&identity.ID)

	return err
}

func (r *PostgresRepository) GetByProviderAndEmail(ctx context.Context, provider Provider, email string) (*Identity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, password_hash, verified_at, created_at, updated_at
		FROM user_identities
		WHERE provider = $1 AND email = $2
	`
	identity := &Identity{}
	err := r.db.QueryRowContext(ctx, query, provider, email).Scan(
		&identity.ID,
		&identity.UserID,
		&identity.Provider,
		&identity.ProviderUserID,
		&identity.Email,
		&identity.PasswordHash,
		&identity.VerifiedAt,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIdentityNotFound
		}
		return nil, err
	}
	return identity, nil
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID string) ([]*Identity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, password_hash, verified_at, created_at, updated_at
		FROM user_identities
		WHERE user_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []*Identity
	for rows.Next() {
		identity := &Identity{}
		err := rows.Scan(
			&identity.ID,
			&identity.UserID,
			&identity.Provider,
			&identity.ProviderUserID,
			&identity.Email,
			&identity.PasswordHash,
			&identity.VerifiedAt,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func (r *PostgresRepository) UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error {
	query := `UPDATE user_identities SET password_hash = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, passwordHash, id)
	return err
}

func (r *PostgresRepository) UpdateVerifiedAt(ctx context.Context, id string, verifiedAt *time.Time) error {
	query := `UPDATE user_identities SET verified_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, verifiedAt, id)
	return err
}
