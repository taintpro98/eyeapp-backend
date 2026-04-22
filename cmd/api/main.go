package main

import (
	"context"
	"net/http"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/auth"
	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/alumieye/eyeapp-backend/pkg/email"
	"github.com/alumieye/eyeapp-backend/internal/orders"
	"github.com/alumieye/eyeapp-backend/internal/repositories"
	"github.com/alumieye/eyeapp-backend/internal/verification"
	"github.com/alumieye/eyeapp-backend/middlewares"
	"github.com/alumieye/eyeapp-backend/pkg/db"
	"github.com/alumieye/eyeapp-backend/pkg/logger"
	"github.com/alumieye/eyeapp-backend/routes"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/fx"

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
// @description Type: Bearer {your_token}

func provideConfig() *config.Config {
	return config.Load()
}

func provideLogger(cfg *config.Config) logger.Logger {
	return logger.New(&logger.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.AppEnv,
		LogFormat:   cfg.LogFormat,
		ServiceName: cfg.ServiceName,
	})
}

func provideDatabase(lc fx.Lifecycle, cfg *config.Config, log logger.Logger) *db.DB {
	database, err := db.Connect(cfg.DatabaseURL, db.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	})
	if err != nil {
		log.Error(context.Background(), "Failed to connect to database", logger.Err(err))
		panic(err)
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return database.Close()
		},
	})
	log.Info(context.Background(), "Connected to database")
	return database
}

func provideEyebrokerDatabase(lc fx.Lifecycle, cfg *config.Config, log logger.Logger) *db.EyebrokerDB {
	pool := db.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	}
	raw, err := db.Connect(cfg.EyebrokerDatabaseURL, pool)
	if err != nil {
		log.Error(context.Background(), "Failed to connect to eyebroker database", logger.Err(err))
		panic(err)
	}
	eyebrokerDB := &db.EyebrokerDB{DB: raw.DB}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return eyebrokerDB.Close()
		},
	})
	log.Info(context.Background(), "Connected to eyebroker database")
	return eyebrokerDB
}

func provideEmailSender(log logger.Logger, cfg *config.Config) email.Sender {
	if cfg.ResendAPIKey != "" {
		return email.NewResendSender(log, cfg.ResendAPIKey, cfg.EmailFrom)
	}
	return &email.NoopSender{}
}

var ReposModule = fx.Module("repos",
	fx.Provide(
		repositories.NewUserRepository,
		repositories.NewIdentityRepository,
		repositories.NewSessionRepository,
		repositories.NewVerificationRepository,
		repositories.NewOrderRepository,
	),
)

var ServicesModule = fx.Module("services",
	fx.Provide(
		provideEmailSender,
		verification.NewService,
		auth.NewTokenService,
		auth.NewService,
	),
)

func provideAuthHandler(authService *auth.Service) *auth.Handler {
	return auth.NewHandler(authService)
}

func provideOrdersHandler(repo repositories.OrderRepository) *orders.Handler {
	return orders.NewHandler(repo)
}

func provideRouter(authHandler *auth.Handler, ordersHandler *orders.Handler, tokenService *auth.TokenService) *routes.Router {
	return routes.NewRouter(authHandler, ordersHandler, tokenService)
}

func provideMux(router *routes.Router, log logger.Logger) *chi.Mux {
	router.Use(middleware.RealIP)
	router.Use(middlewares.TraceID())
	router.Use(middlewares.CORS)
	router.Use(middlewares.Logging(log))
	router.Use(middlewares.Recovery(log))
	return router.Setup()
}

func startServer(lc fx.Lifecycle, mux *chi.Mux, cfg *config.Config, log logger.Logger) {
	log.Info(context.Background(), "Starting ALumiEye API server",
		logger.Str("env", cfg.AppEnv),
		logger.Str("log_level", cfg.LogLevel),
	)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Info(context.Background(), "Server listening",
					logger.Str("port", cfg.Port),
					logger.Str("swagger_url", "http://localhost:"+cfg.Port+"/docs/"),
				)
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal(context.Background(), "Server error", logger.Err(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info(context.Background(), "Shutting down server...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Error(ctx, "Server forced to shutdown", logger.Err(err))
				return err
			}
			log.Info(context.Background(), "Server stopped")
			return nil
		},
	})
}

func main() {
	fx.New(
		fx.Provide(
			provideConfig,
			provideLogger,
			provideDatabase,
			provideEyebrokerDatabase,
		),
		ReposModule,
		ServicesModule,
		fx.Provide(
			provideAuthHandler,
			provideOrdersHandler,
			provideRouter,
			provideMux,
		),
		fx.Invoke(
			startServer,
		),
	).Run()
}
