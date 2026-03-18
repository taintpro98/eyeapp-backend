package routes

import (
	"net/http"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/internal/auth"
	"github.com/alumieye/eyeapp-backend/middlewares"
	"github.com/go-chi/chi/v5"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Router sets up HTTP routes
type Router struct {
	mux          *chi.Mux
	authHandler  *auth.Handler
	tokenService *auth.TokenService
}

// NewRouter creates a new router
func NewRouter(authHandler *auth.Handler, tokenService *auth.TokenService) *Router {
	return &Router{
		mux:          chi.NewRouter(),
		authHandler:  authHandler,
		tokenService: tokenService,
	}
}

// Use adds middleware to the router. Must be called before Setup().
func (r *Router) Use(middlewares ...func(http.Handler) http.Handler) {
	r.mux.Use(middlewares...)
}

// Setup configures all routes and returns the mux.
// Middleware must be applied via Use() before calling Setup().
func (r *Router) Setup() *chi.Mux {
	// Health check
	r.mux.Get("/health", r.handleHealth)

	// Swagger documentation
	r.mux.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Auth endpoints (public)
	r.mux.Post("/auth/register", r.authHandler.Register)
	r.mux.Post("/auth/login", r.authHandler.Login)
	r.mux.Post("/auth/verify-email", r.authHandler.VerifyEmail)
	r.mux.Post("/auth/resend-verification-email", r.authHandler.ResendVerificationEmail)
	r.mux.Post("/auth/refresh", r.authHandler.Refresh)
	r.mux.Post("/auth/logout", r.authHandler.Logout)

	// Protected endpoints
	r.mux.Group(func(protected chi.Router) {
		protected.Use(middlewares.Auth(r.tokenService))
		protected.Get("/me", r.authHandler.Me)
		protected.Get("/users/{id}", r.handleGetUser)
	})

	return r.mux
}

// handleHealth handles health check requests
// @Summary Health check
// @Description Check if the API is running
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string "API is healthy"
// @Router /health [get]
func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	apierrors.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGetUser handles getting a user by ID
// @Summary Get user by ID
// @Description Get a user's public information by their ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string "User info"
// @Failure 404 {object} apierrors.ErrorResponse "User not found"
// @Router /users/{id} [get]
func (r *Router) handleGetUser(w http.ResponseWriter, req *http.Request) {
	userID := chi.URLParam(req, "id")
	if userID == "" {
		apierrors.Error(w, http.StatusBadRequest, apierrors.CodeValidationError, "user id is required")
		return
	}
	// TODO: implement actual user lookup
	apierrors.JSON(w, http.StatusOK, map[string]string{"id": userID})
}
