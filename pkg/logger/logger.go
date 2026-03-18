package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/alumieye/eyeapp-backend/pkg/trace"
	"github.com/rs/zerolog"
)

// LogField represents a key-value pair for structured logging
type LogField struct {
	Key   string
	Value interface{}
}

// Str returns a string log field
func Str(key, value string) LogField { return LogField{Key: key, Value: value} }

// Int returns an int log field
func Int(key string, value int) LogField { return LogField{Key: key, Value: value} }

// Int64 returns an int64 log field
func Int64(key string, value int64) LogField { return LogField{Key: key, Value: value} }

// Err returns an error log field (zerolog uses "error" key)
func Err(err error) LogField { return LogField{Key: "error", Value: err} }

// Any returns a generic log field
func Any(key string, value interface{}) LogField { return LogField{Key: key, Value: value} }

// Dur returns a duration log field
func Dur(key string, value time.Duration) LogField { return LogField{Key: key, Value: value} }

// Bool returns a bool log field
func Bool(key string, value bool) LogField { return LogField{Key: key, Value: value} }

// Uint returns a uint log field
func Uint(key string, value uint) LogField { return LogField{Key: key, Value: value} }

// Logger is the interface for structured logging with context support (trace_id)
type Logger interface {
	Info(ctx context.Context, msg string, fields ...LogField)
	Error(ctx context.Context, msg string, fields ...LogField)
	Warn(ctx context.Context, msg string, fields ...LogField)
	Debug(ctx context.Context, msg string, fields ...LogField)
	Fatal(ctx context.Context, msg string, fields ...LogField)
}

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Environment string // development, production
	LogFormat   string // json, console (empty = auto from Environment)
	ServiceName string
}

// zerologLogger implements Logger using zerolog
type zerologLogger struct {
	z zerolog.Logger
}

// New creates a new logger based on the environment
func New(cfg *Config) Logger {
	if cfg == nil {
		cfg = &Config{
			Level:       "info",
			Environment: "development",
			ServiceName: "api",
		}
	}

	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	var output io.Writer
	useJSON := cfg.LogFormat == "json" || (cfg.LogFormat == "" && cfg.Environment == "production")

	if useJSON {
		output = os.Stdout
	} else {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	z := zerolog.New(output).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Logger()

	return &zerologLogger{z: z}
}

// NewNop returns a no-op logger that discards all output. Use in tests.
func NewNop() Logger {
	return &nopLogger{}
}

// nopLogger implements Logger and discards all output
type nopLogger struct{}

func (nopLogger) Info(context.Context, string, ...LogField)  {}
func (nopLogger) Error(context.Context, string, ...LogField) {}
func (nopLogger) Warn(context.Context, string, ...LogField)  {}
func (nopLogger) Debug(context.Context, string, ...LogField) {}
func (nopLogger) Fatal(context.Context, string, ...LogField) {}

func (l *zerologLogger) Info(ctx context.Context, msg string, fields ...LogField) {
	l.logEvent(ctx, zerolog.InfoLevel, msg, fields)
}

func (l *zerologLogger) Error(ctx context.Context, msg string, fields ...LogField) {
	l.logEvent(ctx, zerolog.ErrorLevel, msg, fields)
}

func (l *zerologLogger) Warn(ctx context.Context, msg string, fields ...LogField) {
	l.logEvent(ctx, zerolog.WarnLevel, msg, fields)
}

func (l *zerologLogger) Debug(ctx context.Context, msg string, fields ...LogField) {
	l.logEvent(ctx, zerolog.DebugLevel, msg, fields)
}

func (l *zerologLogger) Fatal(ctx context.Context, msg string, fields ...LogField) {
	l.logEvent(ctx, zerolog.FatalLevel, msg, fields)
	os.Exit(1)
}

func (l *zerologLogger) logEvent(ctx context.Context, level zerolog.Level, msg string, fields []LogField) {
	ev := l.z.WithLevel(level)
	if ctx != nil {
		if traceID := trace.GetTraceID(ctx); traceID != "" {
			ev = ev.Str("trace_id", traceID)
		}
	}
	for _, f := range fields {
		ev = applyField(ev, f)
	}
	ev.Msg(msg)
}

func applyField(ev *zerolog.Event, f LogField) *zerolog.Event {
	switch v := f.Value.(type) {
	case error:
		return ev.Err(v)
	case string:
		return ev.Str(f.Key, v)
	case int:
		return ev.Int(f.Key, v)
	case int64:
		return ev.Int64(f.Key, v)
	case bool:
		return ev.Bool(f.Key, v)
	case time.Duration:
		return ev.Dur(f.Key, v)
	case uint:
		return ev.Uint(f.Key, v)
	default:
		return ev.Interface(f.Key, v)
	}
}

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
