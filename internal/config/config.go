package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// appConfig holds app-level settings from config.yml
type appConfig struct {
	Env         string `yaml:"env"`
	Port        string `yaml:"port"`
	ServiceName string `yaml:"service_name"`
}

// serverConfig holds server/token settings from config.yml
type serverConfig struct {
	AccessTokenTTLMinutes int `yaml:"access_token_ttl_minutes"`
	RefreshTokenTTLHours  int `yaml:"refresh_token_ttl_hours"`
}

// loggingConfig holds logging settings from config.yml
type loggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// emailConfig holds email settings from config.yml
type emailConfig struct {
	VerificationTokenTTLHours int `yaml:"verification_token_ttl_hours"`
}

// databaseConfig holds database pool settings from config.yml
type databaseConfig struct {
	MaxOpenConns           int `yaml:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int `yaml:"conn_max_lifetime_minutes"`
}

// yamlConfig matches the structure of configs/config.yml
type yamlConfig struct {
	App      appConfig      `yaml:"app"`
	Server   serverConfig   `yaml:"server"`
	Logging  loggingConfig  `yaml:"logging"`
	Email    emailConfig    `yaml:"email"`
	Database databaseConfig `yaml:"database"`
}

type Config struct {
	AppEnv          string
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	LogLevel        string
	LogFormat       string
	ServiceName     string
	EmailVerificationTTL time.Duration
	ResendAPIKey        string
	EmailFrom           string
	AppVerifyURLBase    string
	DBMaxOpenConns      int
	DBMaxIdleConns      int
	DBConnMaxLifetime   time.Duration
}

func Load() *Config {
	_ = godotenv.Load() // load .env if present (no-op if missing)

	cfg := &Config{
		AppEnv:                 "development",
		Port:                   "8080",
		DatabaseURL:            "",
		JWTSecret:              "",
		AccessTokenTTL:         15 * time.Minute,
		RefreshTokenTTL:        720 * time.Hour,
		LogLevel:               "info",
		LogFormat:              "",
		ServiceName:            "alumieye-api",
		EmailVerificationTTL:   24 * time.Hour,
		ResendAPIKey:           "",
		EmailFrom:              "",
		AppVerifyURLBase:       "http://localhost:5173/verify-email",
		DBMaxOpenConns:         25,
		DBMaxIdleConns:         5,
		DBConnMaxLifetime:      5 * time.Minute,
	}

	// Load public config from configs/config.yml
	if data, err := os.ReadFile(configPath()); err == nil {
		var yc yamlConfig
		if err := yaml.Unmarshal(data, &yc); err == nil {
			if yc.App.Env != "" {
				cfg.AppEnv = yc.App.Env
			}
			if yc.App.Port != "" {
				cfg.Port = yc.App.Port
			}
			if yc.App.ServiceName != "" {
				cfg.ServiceName = yc.App.ServiceName
			}
			if yc.Server.AccessTokenTTLMinutes > 0 {
				cfg.AccessTokenTTL = time.Duration(yc.Server.AccessTokenTTLMinutes) * time.Minute
			}
			if yc.Server.RefreshTokenTTLHours > 0 {
				cfg.RefreshTokenTTL = time.Duration(yc.Server.RefreshTokenTTLHours) * time.Hour
			}
			if yc.Logging.Level != "" {
				cfg.LogLevel = yc.Logging.Level
			}
			cfg.LogFormat = yc.Logging.Format
		}
		if yc.Email.VerificationTokenTTLHours > 0 {
			cfg.EmailVerificationTTL = time.Duration(yc.Email.VerificationTokenTTLHours) * time.Hour
		}
		if yc.Database.MaxOpenConns > 0 {
			cfg.DBMaxOpenConns = yc.Database.MaxOpenConns
		}
		if yc.Database.MaxIdleConns > 0 {
			cfg.DBMaxIdleConns = yc.Database.MaxIdleConns
		}
		if yc.Database.ConnMaxLifetimeMinutes > 0 {
			cfg.DBConnMaxLifetime = time.Duration(yc.Database.ConnMaxLifetimeMinutes) * time.Minute
		}
	}

	// Secrets: from environment only (no defaults in config.yml)
	// DATABASE_URL takes precedence; otherwise build from DB_* parts
	if u := getEnv("DATABASE_URL", ""); u != "" {
		cfg.DatabaseURL = u
	} else {
		cfg.DatabaseURL = buildDatabaseURL(
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "postgres"),
			getEnv("DB_NAME", "alumieye"),
			getEnv("DB_SSL_MODE", "disable"),
		)
	}
	cfg.JWTSecret = getEnv("JWT_SECRET", "change_me_in_production")
	cfg.ResendAPIKey = getEnv("RESEND_API_KEY", "")
	cfg.EmailFrom = getEnv("EMAIL_FROM", "ALumiEye <onboarding@resend.dev>")
	cfg.AppVerifyURLBase = getEnv("APP_VERIFY_URL_BASE", "http://localhost:5173/verify-email")
	cfg.DBMaxOpenConns = getEnvAsInt("DB_MAX_OPEN_CONNS", cfg.DBMaxOpenConns)
	cfg.DBMaxIdleConns = getEnvAsInt("DB_MAX_IDLE_CONNS", cfg.DBMaxIdleConns)
	if mins := getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 5); mins > 0 {
		cfg.DBConnMaxLifetime = time.Duration(mins) * time.Minute
	}

	return cfg
}

func configPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return filepath.Join("configs", "config.yml")
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// buildDatabaseURL constructs a PostgreSQL connection string from parts
func buildDatabaseURL(host, port, user, password, dbName, sslMode string) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   "/" + dbName,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()
	return u.String()
}
