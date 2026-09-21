// Runs every pending migration in ./migrations against DATABASE_URL.
// Idempotent — safe to run on every deploy (see deploy.sh), including the
// very first one, which creates the schema.
package main

import (
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	// Self-loads .env so deploy.sh doesn't need to `source .env` into the
	// shell first — that pattern was what leaked stale env vars into pm2's
	// cached process environment (see internal/config/config.go).
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	path := os.Getenv("MIGRATIONS_PATH")
	if path == "" {
		path = "migrations"
	}

	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		log.Fatalf("migrate init failed: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No pending migrations.")
			return
		}
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("Migrations applied.")
}
