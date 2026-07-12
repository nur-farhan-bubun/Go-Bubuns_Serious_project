SERVICES := api-gateway user-service match-service chat-service location-service
SHARED := shared

# ─── Go Build ────────────────────────────────────────────────────────────────

.PHONY: build build-all build-$(SERVICES)

build-all: $(addprefix build-, $(SERVICES))

build-%:
	cd $* && go build -o bin/$* ./cmd/api

build-shared:
	@echo "shared is a library module — nothing to build"

# ─── Go Run (local development) ─────────────────────────────────────────────

.PHONY: run run-$(SERVICES)

run-%:
	cd $* && go run ./cmd/api

run-all:
	@echo "Starting all services with Docker Compose..."
	docker compose up -d

# ─── Docker Build ────────────────────────────────────────────────────────────

.PHONY: docker-build docker-build-$(SERVICES)

docker-build-all: $(addprefix docker-build-, $(SERVICES))

docker-build-%:
	docker build -t $* ./$*

# ─── Docker Compose ─────────────────────────────────────────────────────────

.PHONY: up down logs

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

# ─── Go Test ────────────────────────────────────────────────────────────────

.PHONY: test test-all test-$(SERVICES)

test-all: $(addprefix test-, $(SERVICES)) test-shared

test-%:
	cd $* && go test ./... -v -count=1

test-shared:
	cd $(SHARED) && go test ./pkg/... -v -count=1

# ─── Go Mod Tidy ────────────────────────────────────────────────────────────

.PHONY: tidy tidy-all tidy-$(SERVICES)

tidy-all: $(addprefix tidy-, $(SERVICES)) tidy-shared

tidy-%:
	cd $* && go mod tidy

tidy-shared:
	cd $(SHARED) && go mod tidy

# ─── Migration ──────────────────────────────────────────────────────────────

USER_DB_URL  := postgres://postgres:postgres@localhost:5432/users?sslmode=disable
MATCH_DB_URL := postgres://postgres:postgres@localhost:5433/matches?sslmode=disable
LOCATION_DB_URL := postgres://postgres:postgres@localhost:5434/locations?sslmode=disable

export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: migrate-up migrate-down migrate-status
.PHONY: migrate-user migrate-user-down
.PHONY: migrate-match migrate-match-down
.PHONY: migrate-location migrate-location-down
.PHONY: migrate-chat migrate-chat-down

migrate-up: migrate-user migrate-match migrate-location migrate-chat
	@echo "✓ All migrations applied"

migrate-down: migrate-chat-down migrate-location-down migrate-match-down migrate-user-down
	@echo "✓ All migrations rolled back"

migrate-status:
	@echo "── user-db ──"
	@migrate -path user-service/migrations -database "$(USER_DB_URL)" version 2>/dev/null || echo "(no migrations applied)"
	@echo "── match-db ──"
	@migrate -path match-service/migrations -database "$(MATCH_DB_URL)" version 2>/dev/null || echo "(no migrations applied)"
	@echo "── location-db ──"
	@migrate -path location-service/migrations -database "$(LOCATION_DB_URL)" version 2>/dev/null || echo "(no migrations applied)"
	@echo "── chat-db (ScyllaDB) ──"
	@echo "(check with: cqlsh localhost 9042 -e \"SELECT * FROM system_schema_migrations;\")"

# PostgreSQL services

migrate-user:
	@echo "▸ Migrating user-db..."
	@migrate -path user-service/migrations -database "$(USER_DB_URL)" up

migrate-user-down:
	@echo "▸ Rolling back user-db..."
	@migrate -path user-service/migrations -database "$(USER_DB_URL)" down 1

migrate-match:
	@echo "▸ Migrating match-db..."
	@migrate -path match-service/migrations -database "$(MATCH_DB_URL)" up

migrate-match-down:
	@echo "▸ Rolling back match-db..."
	@migrate -path match-service/migrations -database "$(MATCH_DB_URL)" down 1

migrate-location:
	@echo "▸ Migrating location-db..."
	@migrate -path location-service/migrations -database "$(LOCATION_DB_URL)" up

migrate-location-down:
	@echo "▸ Rolling back location-db..."
	@migrate -path location-service/migrations -database "$(LOCATION_DB_URL)" down 1

# ScyllaDB service

migrate-chat:
	@echo "▸ Migrating chat-db (ScyllaDB)..."
	@cat chat-service/migrations/001_chat.up.sql chat-service/migrations/002_groups.up.sql | docker compose exec -T chat-db cqlsh

migrate-chat-down:
	@echo "▸ chat-db: no down migration for ScyllaDB (drop keyspace manually if needed)"

# ─── Lint / Vet ─────────────────────────────────────────────────────────────

.PHONY: vet vet-all vet-$(SERVICES)

vet-all: $(addprefix vet-, $(SERVICES)) vet-shared

vet-%:
	cd $* && go vet ./...

vet-shared:
	cd $(SHARED) && go vet ./pkg/...

# ─── Clean ────────────────────────────────────────────────────────────────

.PHONY: clean

clean:
	rm -rf $(SERVICES:%=%/bin)

# ─── Help ────────────────────────────────────────────────────────────────────

.PHONY: help

help:
	@echo "Usage:"
	@echo "  make build-<service>     Build a service (api-gateway, user-service, etc.)"
	@echo "  make build-all           Build all services"
	@echo "  make run-<service>       Run a service locally"
	@echo "  make run-all             Start all services via Docker Compose"
	@echo "  make test-all            Run all tests"
	@echo "  make vet-all             Run go vet on all modules"
	@echo "  make tidy-all            Run go mod tidy on all modules"
	@echo "  make docker-build-all    Build all Docker images"
	@echo "  make up                  Start all services with Docker Compose"
	@echo "  make down                Stop all services"
	@echo "  make logs                Tail Docker Compose logs"
	@echo "  make clean               Remove build artifacts"
	@echo ""
	@echo "Services: $(SERVICES)"
# ─── gRPC Protobuf Compilation ──────────────────────────────────────────────

.PHONY: compile-proto
compile-proto:
	@echo "Compiling gRPC Protocol Buffers..."
	protoc --proto_path=shared \
	       --go_out=shared --go_opt=paths=source_relative \
	       --go-grpc_out=shared --go-grpc_opt=paths=source_relative \
	       shared/proto/user/user.proto

.PHONY: tidy-all
tidy-all:
	cd shared && go mod tidy
	cd user-service && go mod tidy
	cd chat-service && go mod tidy