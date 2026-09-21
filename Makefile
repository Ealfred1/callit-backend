APP_NAME=callit-api

.PHONY: run build test migrate dev db-up db-down db-reset docker-logs fmt vet smoke

run:
	go run cmd/api/main.go

build:
	go build -o bin/$(APP_NAME) cmd/api/main.go

test:
	go test -v ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

# Local dev Postgres only — the app itself runs as a plain binary, not a
# container (see deploy/ for why: minimal footprint on a disk-constrained
# shared VPS). Migrations are never applied by hand: `make dev` always runs
# them first, same as deploy/start.sh does on the server.
db-up:
	docker compose up -d db

db-down:
	docker compose down -v

db-reset: db-down db-up
	@sleep 3
	@$(MAKE) migrate

docker-logs:
	docker compose logs -f

migrate:
	DATABASE_URL=$${DATABASE_URL:-postgres://callit:secret@localhost:5434/callit_db?sslmode=disable} go run cmd/migrate/main.go

# Quick dev cycle: local Postgres up, migrate, run the API — all against
# mock Panta/AI/email (see .env.example) unless a .env overrides it.
dev: db-up
	@sleep 2
	@$(MAKE) migrate
	go run cmd/api/main.go

smoke:
	bash scripts/smoke.sh
