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
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateLastLogin(ctx context.Context, id string, loginTime time.Time) error
	EmailExists(ctx context.Context, email string) (bool, error)
}

type userPostgres struct {
	db *db.DB
}

func NewUserRepository(database *db.DB) UserRepository {
	return &userPostgres{db: database}
}

func (r *userPostgres) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, display_name, status, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		user.Email,
		user.DisplayName,
		user.Status,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return err
	}
	return nil
}

func (r *userPostgres) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, status, role, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.Status,
		&user.Role,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *userPostgres) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, status, role, last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.Status,
		&user.Role,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *userPostgres) UpdateLastLogin(ctx context.Context, id string, loginTime time.Time) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, loginTime, id)
	return err
}

func (r *userPostgres) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	return exists, err
}
