package middlewares

import (
	"net/http"
	"time"

	"github.com/alumieye/eyeapp-backend/pkg/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	size        int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Logging returns HTTP request logging middleware
func Logging(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := wrapResponseWriter(w)
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			ctx := r.Context()

			fields := []logger.LogField{
				logger.Str("method", r.Method),
				logger.Str("path", r.URL.Path),
				logger.Str("query", r.URL.RawQuery),
				logger.Int("status", wrapped.status),
				logger.Int("size", wrapped.size),
				logger.Dur("duration_ms", duration),
				logger.Str("remote_addr", r.RemoteAddr),
				logger.Str("user_agent", r.UserAgent()),
			}

			switch {
			case wrapped.status >= 500:
				log.Error(ctx, "Request completed", fields...)
			case wrapped.status >= 400:
				log.Warn(ctx, "Request completed", fields...)
			default:
				log.Info(ctx, "Request completed", fields...)
			}
		})
	}
}
