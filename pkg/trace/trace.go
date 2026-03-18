package trace

import "context"

type contextKey string

const TraceIDContextKey contextKey = "trace_id"

// GetTraceID extracts trace_id from context. Returns empty string if not set.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDContextKey).(string); ok {
		return v
	}
	return ""
}
