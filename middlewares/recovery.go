package middlewares

import (
	"net/http"

	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

// Recovery returns panic recovery middleware with logging
func Recovery(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error().
						Interface("panic", err).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("Panic recovered")

					w.Header().Set("Content-Type", "application/json")
					http.Error(w, `{"error":{"code":"internal_error","message":"Internal server error"}}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
