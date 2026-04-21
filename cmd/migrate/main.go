package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	direction := flag.String("direction", "up", "Direction to migrate: up | down")
	steps := flag.Int("steps", 0, "Number of steps to migrate down (0 = all)")
	flag.Parse()

	// Load .env if present (safe to ignore error in production)
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	m, err := migrate.New("file://cmd/migrate/migrations", dsn)
	if err != nil {
		log.Fatalf("failed to initialize migrate: %v", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate up failed: %v", err)
		}
		log.Println("✓ migrations applied successfully")

	case "down":
		n := *steps
		if n == 0 {
			if err = m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("migrate down failed: %v", err)
			}
			log.Println("✓ all migrations rolled back")
		} else {
			if err = m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("migrate down %d steps failed: %v", n, err)
			}
			log.Printf("✓ rolled back %d step(s)", n)
		}

	default:
		log.Fatalf("unknown direction %q — use 'up' or 'down'", *direction)
	}
}
