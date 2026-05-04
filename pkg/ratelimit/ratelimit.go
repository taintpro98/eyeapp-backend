package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// RateLimiter is the interface all backends implement.
type RateLimiter interface {
	Allow(key string) bool
}

// Config holds rate limit parameters.
type Config struct {
	Limit  int           // max requests allowed per window
	Window time.Duration // time window (e.g. time.Minute)
}

// KeyByIPAndPath returns a key combining the client IP and request path.
// Use this as the keyFn in the RateLimit middleware.
func KeyByIPAndPath(r *http.Request) string {
	return clientIP(r) + ":" + r.URL.Path
}

// clientIP extracts the real client IP, respecting X-Forwarded-For and X-Real-IP.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
