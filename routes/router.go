package routes

import (
	"net/http"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/internal/auth"
	"github.com/alumieye/eyeapp-backend/internal/orders"
	"github.com/alumieye/eyeapp-backend/middlewares"
	"github.com/go-chi/chi/v5"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Router sets up HTTP routes
type Router struct {
	mux           *chi.Mux
	authHandler   *auth.Handler
	ordersHandler *orders.Handler
	tokenService  *auth.TokenService
}

// NewRouter creates a new router
func NewRouter(authHandler *auth.Handler, ordersHandler *orders.Handler, tokenService *auth.TokenService) *Router {
	return &Router{
		mux:           chi.NewRouter(),
		authHandler:   authHandler,
		ordersHandler: ordersHandler,
		tokenService:  tokenService,
	}
}

// Use adds middleware to the router. Must be called before Setup().
func (r *Router) Use(middlewares ...func(http.Handler) http.Handler) {
	r.mux.Use(middlewares...)
}

// Setup configures all routes and returns the mux.
// Middleware must be applied via Use() before calling Setup().
func (r *Router) Setup() *chi.Mux {
	// Health check (unversioned — used by load balancers / uptime monitors)
	r.mux.Get("/health", r.handleHealth)

	// Swagger documentation
	r.mux.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
		httpSwagger.UIConfig(map[string]string{
			// Auto-prepend "Bearer " if the user pastes a raw token
			"requestInterceptor": `(req) => {
				const auth = req.headers['Authorization'];
				if (auth && !auth.startsWith('Bearer ')) {
					req.headers['Authorization'] = 'Bearer ' + auth;
				}
				return req;
			}`,
		}),
	))

	// v1 API
	r.mux.Route("/api/v1", func(v1 chi.Router) {
		// Auth endpoints (public)
		v1.Post("/auth/register", r.authHandler.Register)
		v1.Post("/auth/login", r.authHandler.Login)
		v1.Post("/auth/verify-email", r.authHandler.VerifyEmail)
		v1.Post("/auth/resend-verification-email", r.authHandler.ResendVerificationEmail)
		v1.Post("/auth/refresh", r.authHandler.Refresh)
		v1.Post("/auth/logout", r.authHandler.Logout)

		// Protected endpoints
		v1.Group(func(protected chi.Router) {
			protected.Use(middlewares.Auth(r.tokenService))
			protected.Get("/me", r.authHandler.Me)
			protected.Get("/users/{id}", r.handleGetUser)
			protected.Get("/orders", r.ordersHandler.List)
		})
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
// @Router /api/v1/users/{id} [get]
func (r *Router) handleGetUser(w http.ResponseWriter, req *http.Request) {
	userID := chi.URLParam(req, "id")
	if userID == "" {
		apierrors.Error(w, http.StatusBadRequest, apierrors.CodeValidationError, "user id is required")
		return
	}
	// TODO: implement actual user lookup
	apierrors.JSON(w, http.StatusOK, map[string]string{"id": userID})
}
