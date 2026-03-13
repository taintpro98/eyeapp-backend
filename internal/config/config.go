package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

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

// yamlConfig matches the structure of configs/config.yml
type yamlConfig struct {
	App     appConfig     `yaml:"app"`
	Server  serverConfig  `yaml:"server"`
	Logging loggingConfig `yaml:"logging"`
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
}

func Load() *Config {
	cfg := &Config{
		AppEnv:          "development",
		Port:            "8080",
		DatabaseURL:     "",
		JWTSecret:       "",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
		LogLevel:        "info",
		LogFormat:       "",
		ServiceName:     "alumieye-api",
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
	}

	// Secrets: from environment only (no defaults in config.yml)
	cfg.DatabaseURL = getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alumieye?sslmode=disable")
	cfg.JWTSecret = getEnv("JWT_SECRET", "change_me_in_production")

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
