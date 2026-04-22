package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/internal/platform/crypto"
	_ "github.com/lib/pq"
)

type seedUser struct {
	email       string
	password    string
	displayName string
	role        string
}

var users = []seedUser{
	{
		email:       "alex@gmail.com",
		password:    "1234",
		displayName: "Alex",
		role:        "user",
	},
}

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Wait for DB to be ready (up to 30s)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		if ctx.Err() != nil {
			log.Fatal("timed out waiting for database")
		}
		time.Sleep(1 * time.Second)
	}

	for _, u := range users {
		if err := seedUser_(ctx, db, u); err != nil {
			log.Fatalf("failed to seed user %s: %v", u.email, err)
		}
		fmt.Printf("seeded: %s\n", u.email)
	}

	fmt.Println("seed complete")
	os.Exit(0)
}

func seedUser_(ctx context.Context, db *sql.DB, u seedUser) error {
	// Hash password
	hash, err := crypto.HashPassword(u.password, crypto.DefaultArgon2idParams())
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()

	// Insert user (idempotent)
	var userID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO users (email, display_name, status, role, created_at, updated_at)
		VALUES ($1, $2, 'active', $3, $4, $4)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id`,
		u.email, u.displayName, u.role, now,
	).Scan(&userID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	// Insert identity with verified_at set (bypasses email verification)
	_, err = db.ExecContext(ctx, `
		INSERT INTO user_identities (user_id, provider, provider_user_id, email, password_hash, verified_at, created_at, updated_at)
		VALUES ($1, 'password', $2, $2, $3, $4, $4, $4)
		ON CONFLICT (provider, email) DO UPDATE
		  SET password_hash = EXCLUDED.password_hash,
		      verified_at   = EXCLUDED.verified_at,
		      updated_at    = EXCLUDED.updated_at`,
		userID, u.email, hash, now,
	)
	if err != nil {
		return fmt.Errorf("insert identity: %w", err)
	}

	return nil
}
