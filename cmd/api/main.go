package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/carvalhosauro/goingcrypt/adapters"
	adapthttp "github.com/carvalhosauro/goingcrypt/adapters/http"
	"github.com/carvalhosauro/goingcrypt/adapters/postgres"
	"github.com/carvalhosauro/goingcrypt/internal/core"
	"github.com/carvalhosauro/goingcrypt/internal/env"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
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
	port string
	db   dbConfig
	jwt  jwtConfig
}

func main() {
	// Load .env if present; in production, variables are injected externally.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	cfg := config{
		port: env.GetString("PORT", "8080"),
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
	log.Println("database connection pool established")

	// Adapters
	generator := adapters.NewGenerator()
	hasher := adapters.NewArgon2idHasher()
	tokenManager := adapters.NewJWTTokenManager([]byte(cfg.jwt.secret), cfg.jwt.issuer)
	totpAdapter := adapters.NewTOTPAdapter()

	// Repositories
	transactor := postgres.NewTransactor(db)
	userRepo := postgres.NewUserRepository(db)
	tokenRepo := postgres.NewRefreshTokenRepository(db)
	linkRepo := postgres.NewLinkRepository(db)

	// Services
	authSvc := core.NewAuthService(userRepo, tokenRepo, transactor, generator, hasher, tokenManager, totpAdapter, cfg.jwt.issuer)
	linkSvc := core.NewLinkService(linkRepo, transactor, generator)

	// HTTP handlers
	healthHandler := adapthttp.NewHealthHandler()
	authHandler := adapthttp.NewAuthHandler(authSvc)
	linkHandler := adapthttp.NewLinkHandler(linkSvc)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(adapthttp.AuthMiddleware(tokenManager))

	// Health endpoint — no auth required so probes and load-balancers can reach it freely
	r.Route("/health", healthHandler.RegisterRoutes)

	r.Route("/api/v1/auth", authHandler.RegisterRoutes)
	r.Route("/api/v1/links", linkHandler.RegisterRoutes)

	// Server
	addr := fmt.Sprintf(":%s", cfg.port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
