package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/auth"
	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/internal/email"
	"github.com/alumieye/eyeapp-backend/routes"
	"github.com/alumieye/eyeapp-backend/internal/identity"
	"github.com/alumieye/eyeapp-backend/internal/session"
	"github.com/alumieye/eyeapp-backend/internal/user"
	"github.com/alumieye/eyeapp-backend/internal/verification"
	"github.com/alumieye/eyeapp-backend/middlewares"
	"github.com/alumieye/eyeapp-backend/pkg/db"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
	"github.com/go-chi/chi/v5/middleware"

	_ "github.com/alumieye/eyeapp-backend/docs" // Swagger docs
)

// @title ALumiEye API
// @version 1.0
// @description ALumiEye Backend Authentication API for the MVP
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@alumieye.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(&logger.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.AppEnv,
		LogFormat:   cfg.LogFormat,
		ServiceName: cfg.ServiceName,
	})

	log.Info(context.Background(), "Starting ALumiEye API server",
		logger.Str("env", cfg.AppEnv),
		logger.Str("log_level", cfg.LogLevel),
	)

	// Connect to database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(context.Background(), "Failed to connect to database", logger.Err(err))
	}
	defer database.Close()
	log.Info(context.Background(), "Connected to database")

	// Initialize repositories
	userRepo := user.NewRepository(database)
	identityRepo := identity.NewRepository(database)
	sessionRepo := session.NewRepository(database)
	verificationRepo := verification.NewRepository(database)

	// Initialize email sender (Resend or no-op if not configured)
	var emailSender email.Sender
	if cfg.ResendAPIKey != "" {
		emailSender = email.NewResendSender(log, cfg.ResendAPIKey, cfg.EmailFrom)
	} else {
		emailSender = &email.NoopSender{}
	}

	// Initialize verification service
	verificationService := verification.NewService(
		log,
		verificationRepo,
		identityRepo,
		emailSender,
		cfg.EmailVerificationTTL,
		cfg.AppVerifyURLBase,
	)

	// Initialize token service
	tokenService := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL)

	// Initialize auth service
	authService := auth.NewService(
		log,
		userRepo,
		identityRepo,
		sessionRepo,
		tokenService,
		verificationService,
		cfg.RefreshTokenTTL,
	)

	// Initialize handlers
	authHandler := auth.NewHandler(authService)

	// Setup router: middleware first (chi requires this), then routes
	router := routes.NewRouter(authHandler, tokenService)
	router.Use(middleware.RealIP)
	router.Use(middlewares.TraceID())
	router.Use(middlewares.CORS)
	router.Use(middlewares.Logging(log))
	router.Use(middlewares.Recovery(log))
	mux := router.Setup()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info(context.Background(), "Server listening",
			logger.Str("port", cfg.Port),
			logger.Str("swagger_url", "http://localhost:"+cfg.Port+"/docs/"),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(context.Background(), "Server error", logger.Err(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info(context.Background(), "Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error(ctx, "Server forced to shutdown", logger.Err(err))
	}

	log.Info(context.Background(), "Server stopped")
}
