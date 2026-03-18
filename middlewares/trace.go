package middlewares

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"

	"github.com/alumieye/eyeapp-backend/pkg/trace"
)

const (
	// Header names for request/response propagation
	HeaderRequestID = "X-Request-ID"
	HeaderTraceID   = "X-Trace-ID"
)

// TraceID returns middleware that sets trace_id in context and response header.
// Uses X-Request-ID from request if present, otherwise generates a UUID.
func TraceID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := strings.TrimSpace(r.Header.Get(HeaderRequestID))
			if traceID == "" {
				traceID = generateUUID()
			}

			ctx := context.WithValue(r.Context(), trace.TraceIDContextKey, traceID)
			w.Header().Set(HeaderTraceID, traceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", b[0])
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
