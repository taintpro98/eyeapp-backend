package middlewares

import (
	"net/http"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/pkg/ratelimit"
)

// RateLimit returns a middleware that limits requests using the provided RateLimiter.
// keyFn determines the rate limit key per request (e.g. ratelimit.KeyByIPAndPath).
func RateLimit(limiter ratelimit.RateLimiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "60")
				apierrors.JSON(w, http.StatusTooManyRequests, map[string]any{
					"error": map[string]string{
						"code":    "rate_limit_exceeded",
						"message": "Too many requests. Please try again later.",
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
