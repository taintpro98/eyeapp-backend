package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
)

// Handler handles HTTP requests for authentication
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} AuthResponse "User registered successfully"
// @Failure 400 {object} apierrors.ErrorResponse "Validation error"
// @Failure 409 {object} apierrors.ErrorResponse "Email already exists"
// @Failure 500 {object} apierrors.ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.ValidationError(w, "Invalid request body")
		return
	}

	reqCtx := extractRequestContext(r)

	resp, err := h.service.Register(r.Context(), &req, reqCtx)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	apierrors.JSON(w, http.StatusCreated, resp)
}

// Login handles user login
// @Summary Login user
// @Description Authenticate a user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse "Login successful"
// @Failure 400 {object} apierrors.ErrorResponse "Validation error"
// @Failure 401 {object} apierrors.ErrorResponse "Invalid credentials"
// @Failure 403 {object} apierrors.ErrorResponse "Account blocked"
// @Failure 500 {object} apierrors.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.ValidationError(w, "Invalid request body")
		return
	}

	reqCtx := extractRequestContext(r)

	resp, err := h.service.Login(r.Context(), &req, reqCtx)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	apierrors.JSON(w, http.StatusOK, resp)
}

// Refresh handles token refresh
// @Summary Refresh access token
// @Description Exchange a valid refresh token for a new access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} AuthResponse "Tokens refreshed successfully"
// @Failure 401 {object} apierrors.ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} apierrors.ErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.ValidationError(w, "Invalid request body")
		return
	}

	reqCtx := extractRequestContext(r)

	resp, err := h.service.Refresh(r.Context(), &req, reqCtx)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	apierrors.JSON(w, http.StatusOK, resp)
}

// Logout handles user logout
// @Summary Logout user
// @Description Revoke the current session/refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LogoutRequest false "Refresh token to revoke"
// @Success 200 {object} map[string]string "Logout successful"
// @Failure 500 {object} apierrors.ErrorResponse "Internal server error"
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for simple logout
		req = LogoutRequest{}
	}

	if err := h.service.Logout(r.Context(), &req); err != nil {
		apierrors.InternalError(w)
		return
	}

	apierrors.JSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Me returns the current authenticated user
// @Summary Get current user
// @Description Get the profile of the currently authenticated user
// @Tags user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MeResponse "User profile"
// @Failure 401 {object} apierrors.ErrorResponse "Unauthorized"
// @Failure 500 {object} apierrors.ErrorResponse "Internal server error"
// @Router /me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		apierrors.Unauthorized(w, "User not authenticated")
		return
	}

	user, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		apierrors.InternalError(w)
		return
	}

	apierrors.JSON(w, http.StatusOK, MeResponse{User: user.ToResponse()})
}

// handleAuthError converts auth errors to HTTP responses
func (h *Handler) handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmailAlreadyExists):
		apierrors.Error(w, http.StatusConflict, apierrors.CodeEmailAlreadyExists, "Email already registered")
	case errors.Is(err, ErrInvalidCredentials):
		apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeInvalidCredentials, "Invalid email or password")
	case errors.Is(err, ErrUserBlocked):
		apierrors.Error(w, http.StatusForbidden, apierrors.CodeUserBlocked, "Account is blocked")
	case errors.Is(err, ErrInvalidRefreshToken):
		apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeInvalidRefreshToken, "Invalid refresh token")
	case errors.Is(err, ErrSessionExpired):
		apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeSessionExpired, "Session has expired")
	case errors.Is(err, ErrSessionRevoked):
		apierrors.Error(w, http.StatusUnauthorized, apierrors.CodeInvalidRefreshToken, "Session has been revoked")
	default:
		// Check for validation errors
		if err.Error() != "" && (
			err.Error() == "email is required" ||
			err.Error() == "invalid email format" ||
			err.Error() == "password is required" ||
			err.Error() == "password must be at least 8 characters") {
			apierrors.ValidationError(w, err.Error())
			return
		}
		apierrors.InternalError(w)
	}
}

// extractRequestContext extracts context information from the HTTP request
func extractRequestContext(r *http.Request) *RequestContext {
	return &RequestContext{
		UserAgent: r.UserAgent(),
		IPAddress: getClientIP(r),
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := len(xff); idx > 0 {
			for i, c := range xff {
				if c == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
