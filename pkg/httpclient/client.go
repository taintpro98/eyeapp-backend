package httpclient

import (
	"net/http"
	"time"
)

// Client wraps the HTTP client with common configurations
type Client struct {
	*http.Client
}

// Config holds HTTP client configuration
type Config struct {
	Timeout             time.Duration
	MaxIdleConns        int
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:             30 * time.Second,
		MaxIdleConns:        100,
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 10,
	}
}

// New creates a new HTTP client with the given configuration
func New(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
	}

	return &Client{
		Client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

// NewDefault creates a new HTTP client with default configuration
func NewDefault() *Client {
	return New(nil)
}
