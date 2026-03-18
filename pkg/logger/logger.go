package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger
type Logger struct {
	zerolog.Logger
}

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Environment string // development, production
	LogFormat   string // json, console (empty = auto from Environment)
	ServiceName string
}

// New creates a new logger based on the environment
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = &Config{
			Level:       "info",
			Environment: "development",
			ServiceName: "api",
		}
	}

	// Set log level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	var output io.Writer
	useJSON := cfg.LogFormat == "json" || (cfg.LogFormat == "" && cfg.Environment == "production")

	if useJSON {
		// Production: JSON output to stdout (machine-readable for log aggregators)
		output = os.Stdout
	} else {
		// Development: Pretty console output
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(output).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Logger()

	return &Logger{logger}
}

// NewProduction creates a production JSON logger
func NewProduction(serviceName string) *Logger {
	return New(&Config{
		Level:       "info",
		Environment: "production",
		ServiceName: serviceName,
	})
}

// NewDevelopment creates a development console logger
func NewDevelopment(serviceName string) *Logger {
	return New(&Config{
		Level:       "debug",
		Environment: "development",
		ServiceName: serviceName,
	})
}

// NewNop returns a no-op logger that discards all output. Use in tests.
func NewNop() *Logger {
	return &Logger{Logger: zerolog.Nop()}
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithRequestID returns a logger with request ID context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{l.With().Str("request_id", requestID).Logger()}
}

// WithUserID returns a logger with user ID context
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{l.With().Str("user_id", userID).Logger()}
}

// WithError returns a logger with error context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{l.With().Err(err).Logger()}
}

// WithField returns a logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{l.With().Interface(key, value).Logger()}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{ctx.Logger()}
}
