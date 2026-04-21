package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/carvalhosauro/goingcrypt/adapters"
	adapthttp "github.com/carvalhosauro/goingcrypt/adapters/http"
	"github.com/carvalhosauro/goingcrypt/adapters/postgres"
	"github.com/carvalhosauro/goingcrypt/internal/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	jwtSecret := mustEnv("JWT_SECRET")
	jwtIssuer := getEnv("JWT_ISSUER", "goingcrypt")
	port := getEnv("PORT", "8080")

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	generator := adapters.NewGenerator()
	hasher := adapters.NewArgon2idHasher()
	tokenManager := adapters.NewJWTTokenManager([]byte(jwtSecret), jwtIssuer)
	totpAdapter := adapters.NewTOTPAdapter()
	transactor := postgres.NewTransactor(db)
	userRepo := postgres.NewUserRepository(db)
	tokenRepo := postgres.NewRefreshTokenRepository(db)
	linkRepo := postgres.NewLinkRepository(db)

	authSvc := core.NewAuthService(userRepo, tokenRepo, transactor, generator, hasher, tokenManager, totpAdapter, jwtIssuer)
	linkSvc := core.NewLinkService(linkRepo, transactor, generator)

	authHandler := adapthttp.NewAuthHandler(authSvc)
	linkHandler := adapthttp.NewLinkHandler(linkSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(adapthttp.AuthMiddleware(tokenManager))

	r.Route("/api/v1/auth", authHandler.RegisterRoutes)
	r.Route("/api/v1/links", linkHandler.RegisterRoutes)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
