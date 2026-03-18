package auth

import "context"

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
)

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}
