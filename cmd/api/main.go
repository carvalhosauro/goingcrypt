package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/carvalhosauro/goingcrypt/adapters"
	adapthttp "github.com/carvalhosauro/goingcrypt/adapters/http"
	"github.com/carvalhosauro/goingcrypt/adapters/postgres"
	"github.com/carvalhosauro/goingcrypt/internal/core"
	"github.com/carvalhosauro/goingcrypt/internal/env"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "go.uber.org/automaxprocs" // auto-sets GOMAXPROCS to the cgroup CPU quota
)

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type jwtConfig struct {
	secret string
	issuer string
}

type config struct {
	port            string
	shutdownTimeout time.Duration
	db              dbConfig
	jwt             jwtConfig
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load .env if present; in production, variables are injected externally.
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, relying on environment variables")
	}

	cfg := config{
		port:            env.GetString("PORT", "8080"),
		shutdownTimeout: 30 * time.Second,
		db: dbConfig{
			addr:         env.GetString("DATABASE_URL", ""),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 25),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 5),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "5m"),
		},
		jwt: jwtConfig{
			secret: env.GetString("JWT_SECRET", ""),
			issuer: env.GetString("JWT_ISSUER", "goingcrypt"),
		},
	}

	// Validate required fields eagerly, before any connection attempt.
	if cfg.db.addr == "" {
		log.Fatal(`required environment variable "DATABASE_URL" is not set`)
	}
	if cfg.jwt.secret == "" {
		log.Fatal(`required environment variable "JWT_SECRET" is not set`)
	}

	// Database
	db, err := postgres.NewDB(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()
	slog.Info("database connection pool established")

	// Adapters
	generator := adapters.NewGenerator()
	hasher := adapters.NewArgon2idHasher()
	tokenManager := adapters.NewJWTTokenManager([]byte(cfg.jwt.secret), cfg.jwt.issuer)

	// Repositories
	transactor := postgres.NewTransactor(db)
	userRepo := postgres.NewUserRepository(db)
	tokenRepo := postgres.NewRefreshTokenRepository(db)
	linkRepo := postgres.NewLinkRepository(db)

	// Services
	authSvc := core.NewAuthService(userRepo, tokenRepo, transactor, generator, hasher, tokenManager)
	linkSvc := core.NewLinkService(linkRepo, generator)
	adminUserSvc := core.NewAdminUserService(userRepo)

	// HTTP handlers
	healthHandler := adapthttp.NewHealthHandler()
	authHandler := adapthttp.NewAuthHandler(authSvc)
	linkHandler := adapthttp.NewLinkHandler(linkSvc)
	adminHandler := adapthttp.NewAdminHandler(linkSvc, adminUserSvc)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(adapthttp.AuthMiddleware(tokenManager))

	r.Route("/health", healthHandler.RegisterRoutes)
	r.Route("/api/v1/auth", authHandler.RegisterRoutes)
	r.Route("/api/v1/links", linkHandler.RegisterRoutes)
	r.Route("/api/v1/admin", adminHandler.RegisterRoutes)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Block until we receive SIGTERM (sent by Docker/k8s) or SIGINT (Ctrl+C).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// done is closed on shutdown to broadcast stop to all background workers.
	// Closing a channel is the idiomatic Go way to fan-out a signal to N goroutines.
	done := make(chan struct{})

	// Background worker: marks expired links every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := linkRepo.InvalidateExpiredLinks(ctx); err != nil {
					slog.Error("failed to invalidate expired links", "err", err)
				}
				cancel()
			case <-done:
				return
			}
		}
	}()

	// Background worker: purges revoked/expired tokens older than 7 days, every hour.
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				cutoff := time.Now().Add(-7 * 24 * time.Hour)
				n, err := tokenRepo.DeleteExpiredAndRevoked(ctx, cutoff)
				if err != nil {
					slog.Error("failed to cleanup expired tokens", "err", err)
				} else if n > 0 {
					slog.Info("cleaned up expired tokens", "count", n)
				}
				cancel()
			case <-done:
				return
			}
		}
	}()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)

	case sig := <-quit:
		slog.Info("received signal — initiating graceful shutdown", "signal", sig)
	}

	// Signal all background workers to stop.
	close(done)

	// Give in-flight requests up to shutdownTimeout to finish.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown after timeout", "err", err)
	} else {
		slog.Info("server shut down cleanly")
	}
}
