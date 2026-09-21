// Runs every pending migration in ./migrations against DATABASE_URL.
// Idempotent — safe to run on every deploy (see deploy/start.sh), including
// the very first one, which creates the schema.
package main

import (
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
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
