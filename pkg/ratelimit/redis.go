package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowScript implements a sliding window rate limit using a sorted set.
// Each request is stored as a member scored by its timestamp (ms).
// Members older than the window are removed on every call.
//
// KEYS[1] = rate limit key
// ARGV[1] = window size in milliseconds
// ARGV[2] = max allowed requests per window
// ARGV[3] = current timestamp in milliseconds
var slidingWindowScript = redis.NewScript(`
local key    = KEYS[1]
local window = tonumber(ARGV[1])
local limit  = tonumber(ARGV[2])
local now    = tonumber(ARGV[3])
local min    = now - window

redis.call("ZREMRANGEBYSCORE", key, "-inf", min)
local count = redis.call("ZCARD", key)
if count < limit then
    redis.call("ZADD", key, now, now)
    redis.call("PEXPIRE", key, window)
    return 1
end
return 0
`)

// RedisRateLimiter is a sliding window rate limiter backed by Redis.
// Accurate across multiple instances/pods. Requires a running Redis server.
type RedisRateLimiter struct {
	client *redis.Client
	cfg    Config
	prefix string
}

// NewRedis creates a Redis-backed rate limiter.
// prefix is prepended to all keys (e.g. "rl:auth") to avoid collisions.
func NewRedis(client *redis.Client, cfg Config, prefix string) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		cfg:    cfg,
		prefix: prefix,
	}
}

// Allow returns true if the key is within the rate limit.
// Fails open — if Redis is unavailable, the request is allowed.
func (r *RedisRateLimiter) Allow(key string) bool {
	ctx := context.Background()
	now := time.Now().UnixMilli()
	windowMs := r.cfg.Window.Milliseconds()

	result, err := slidingWindowScript.Run(
		ctx,
		r.client,
		[]string{r.prefix + ":" + key},
		windowMs,
		r.cfg.Limit,
		now,
	).Int()
	if err != nil {
		// Fail open: don't block users if Redis is down
		return true
	}
	return result == 1
}
