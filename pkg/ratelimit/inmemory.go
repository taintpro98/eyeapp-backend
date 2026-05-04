package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// InMemoryRateLimiter is a per-key token bucket rate limiter backed by memory.
// Safe for a single instance. Not shared across multiple pods.
type InMemoryRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	r       rate.Limit
	burst   int
	window  time.Duration
}

// NewInMemory creates an in-memory rate limiter.
// Stale keys are cleaned up every 2× the window duration.
func NewInMemory(cfg Config) *InMemoryRateLimiter {
	l := &InMemoryRateLimiter{
		entries: make(map[string]*entry),
		r:       rate.Every(cfg.Window / time.Duration(cfg.Limit)),
		burst:   cfg.Limit,
		window:  cfg.Window,
	}
	go l.cleanup()
	return l
}

// Allow returns true if the key is within the rate limit.
func (l *InMemoryRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	e, ok := l.entries[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.r, l.burst)}
		l.entries[key] = e
	}
	e.lastSeen = time.Now()
	l.mu.Unlock()
	return e.limiter.Allow()
}

// cleanup removes stale entries to prevent unbounded memory growth.
func (l *InMemoryRateLimiter) cleanup() {
	for {
		time.Sleep(l.window * 2)
		l.mu.Lock()
		for k, e := range l.entries {
			if time.Since(e.lastSeen) > l.window*2 {
				delete(l.entries, k)
			}
		}
		l.mu.Unlock()
	}
}
