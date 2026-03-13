package http

import (
	"net/http"

	"github.com/alumieye/eyeapp-backend/internal/apierrors"
	"github.com/alumieye/eyeapp-backend/internal/auth"
	"github.com/alumieye/eyeapp-backend/pkg/logger"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Router sets up HTTP routes
type Router struct {
	mux            *http.ServeMux
	authHandler    *auth.Handler
	authMiddleware *auth.Middleware
	logger         *logger.Logger
}

// NewRouter creates a new router
func NewRouter(authHandler *auth.Handler, authMiddleware *auth.Middleware, log *logger.Logger) *Router {
	return &Router{
		mux:            http.NewServeMux(),
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
		logger:         log,
	}
}

// Setup configures all routes
func (r *Router) Setup() http.Handler {
	// Health check
	r.mux.HandleFunc("GET /health", r.handleHealth)

	// Swagger documentation
	r.mux.Handle("GET /docs/", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Auth endpoints (public)
	r.mux.HandleFunc("POST /auth/register", r.authHandler.Register)
	r.mux.HandleFunc("POST /auth/login", r.authHandler.Login)
	r.mux.HandleFunc("POST /auth/refresh", r.authHandler.Refresh)
	r.mux.HandleFunc("POST /auth/logout", r.authHandler.Logout)

	// Protected endpoints
	r.mux.Handle("GET /me", r.authMiddleware.Authenticate(http.HandlerFunc(r.authHandler.Me)))

	// Apply global middleware
	return r.applyMiddleware(r.mux)
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

// applyMiddleware applies global middleware to the router
func (r *Router) applyMiddleware(handler http.Handler) http.Handler {
	// Recovery middleware (outermost - catches panics from all other middleware)
	handler = RecoveryMiddleware(r.logger)(handler)
	// Logging middleware
	handler = LoggingMiddleware(r.logger)(handler)
	// CORS middleware
	handler = CORSMiddleware()(handler)
	return handler
}
