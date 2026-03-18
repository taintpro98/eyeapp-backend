package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Define flags
	var (
		migrationsPath string
		steps          int
	)

	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations directory")
	flag.IntVar(&steps, "steps", 0, "Number of migrations to run (for up/down with steps)")
	flag.Parse()

	// Get command
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}
	command := args[0]

	// Load config
	cfg := config.Load()

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.AppEnv,
		LogFormat:   cfg.LogFormat,
		ServiceName: "migrate",
	})

	// Create migrate instance
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatal(context.Background(), "Failed to create migrate instance", logger.Err(err))
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		if steps > 0 {
			err = m.Steps(steps)
		} else {
			err = m.Up()
		}
		if err != nil && err != migrate.ErrNoChange {
			log.Fatal(context.Background(), "Migration up failed", logger.Err(err))
		}
		log.Info(context.Background(), "Migrations applied successfully")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatal(context.Background(), "Failed to get version", logger.Err(err))
		}
		log.Info(context.Background(), "Current migration version",
			logger.Uint("version", version),
			logger.Bool("dirty", dirty),
		)

	case "force":
		if len(args) < 2 {
			log.Fatal(context.Background(), "Force requires a version number")
		}
		var version int
		fmt.Sscanf(args[1], "%d", &version)
		err = m.Force(version)
		if err != nil {
			log.Fatal(context.Background(), "Force failed", logger.Err(err))
		}
		log.Info(context.Background(), "Forced to version", logger.Int("version", version))

	case "drop":
		err = m.Drop()
		if err != nil {
			log.Fatal(context.Background(), "Drop failed", logger.Err(err))
		}
		log.Info(context.Background(), "Database dropped")

	case "create":
		if len(args) < 2 {
			log.Fatal(context.Background(), "Create requires a migration name")
		}
		name := args[1]
		createMigration(migrationsPath, name, log)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ALumiEye Database Migration Tool

Usage:
  migrate [flags] <command> [args]

Commands:
  up              Apply all pending migrations
  version         Show current migration version
  force <version> Force set migration version (use with caution)
  drop            Drop everything in the database
  create <name>   Create a new migration file

Flags:
  -path string    Path to migrations directory (default "migrations")
  -steps int      Number of migrations to run (for up/down)

Examples:
  migrate up                    # Apply all migrations
  migrate -steps 1 up           # Apply 1 migration
  migrate version               # Show current version
  migrate create add_users      # Create new migration files
  migrate force 1               # Force version to 1`)
}

func createMigration(path, name string, log logger.Logger) {
	ctx := context.Background()
	// Get next version number
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(ctx, "Failed to read migrations directory", logger.Err(err))
	}

	version := 1
	for _, entry := range entries {
		if !entry.IsDir() {
			var v int
			fmt.Sscanf(entry.Name(), "%d_", &v)
			if v >= version {
				version = v + 1
			}
		}
	}

	// Create migration file (up only, no down migrations)
	upFile := fmt.Sprintf("%s/%03d_%s.up.sql", path, version, name)
	upContent := fmt.Sprintf("-- Migration: %s\n-- Created at: %s\n\n-- Add your migration SQL here\n", name, "now")

	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		log.Fatal(ctx, "Failed to create migration file", logger.Err(err))
	}

	log.Info(ctx, "Migration file created", logger.Str("file", upFile))
}
