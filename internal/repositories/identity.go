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
	ErrIdentityNotFound = errors.New("identity not found")
)

type IdentityRepository interface {
	Create(ctx context.Context, identity *models.Identity) error
	GetByProviderAndEmail(ctx context.Context, provider models.IdentityProvider, email string) (*models.Identity, error)
	GetByUserID(ctx context.Context, userID string) ([]*models.Identity, error)
	UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error
	UpdateVerifiedAt(ctx context.Context, id string, verifiedAt *time.Time) error
}

type identityPostgres struct {
	db *db.DB
}

func NewIdentityRepository(database *db.DB) IdentityRepository {
	return &identityPostgres{db: database}
}

func (r *identityPostgres) Create(ctx context.Context, identity *models.Identity) error {
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

func (r *identityPostgres) GetByProviderAndEmail(ctx context.Context, provider models.IdentityProvider, email string) (*models.Identity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, password_hash, verified_at, created_at, updated_at
		FROM user_identities
		WHERE provider = $1 AND email = $2
	`
	identity := &models.Identity{}
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

func (r *identityPostgres) GetByUserID(ctx context.Context, userID string) ([]*models.Identity, error) {
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

	var identities []*models.Identity
	for rows.Next() {
		identity := &models.Identity{}
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

func (r *identityPostgres) UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error {
	query := `UPDATE user_identities SET password_hash = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, passwordHash, id)
	return err
}

func (r *identityPostgres) UpdateVerifiedAt(ctx context.Context, id string, verifiedAt *time.Time) error {
	query := `UPDATE user_identities SET verified_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, verifiedAt, id)
	return err
}
