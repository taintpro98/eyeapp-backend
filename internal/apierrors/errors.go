package apierrors

import (
	"encoding/json"
	"net/http"
)

// ErrorCode represents a stable error code
type ErrorCode string

const (
	CodeValidationError          ErrorCode = "validation_error"
	CodeEmailAlreadyExists      ErrorCode = "email_already_exists"
	CodeInvalidCredentials      ErrorCode = "invalid_credentials"
	CodeEmailNotVerified        ErrorCode = "email_not_verified"
	CodeUnauthorized            ErrorCode = "unauthorized"
	CodeInvalidRefreshToken     ErrorCode = "invalid_refresh_token"
	CodeSessionExpired          ErrorCode = "session_expired"
	CodeUserBlocked             ErrorCode = "user_blocked"
	CodeInvalidVerificationToken ErrorCode = "invalid_verification_token"
	CodeVerificationTokenExpired ErrorCode = "verification_token_expired"
	CodeInternalError            ErrorCode = "internal_error"
	CodeNotFound                 ErrorCode = "not_found"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains error details
type ErrorBody struct {
	Code    ErrorCode `json:"code" example:"validation_error"`
	Message string    `json:"message" example:"Invalid email format"`
}

// JSON writes a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Error writes an error JSON response
func Error(w http.ResponseWriter, status int, code ErrorCode, message string) {
	JSON(w, status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

// ValidationError writes a validation error response
func ValidationError(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, CodeValidationError, message)
}

// Unauthorized writes an unauthorized error response
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, CodeUnauthorized, message)
}

// InternalError writes an internal server error response
func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, CodeInternalError, "An internal error occurred")
}
