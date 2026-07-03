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

.PHONY: migrate-user migrate-match migrate-chat

migrate-user:
	@echo "Run: migrate -path user-service/migrations -database \$$DATABASE_URL up"

migrate-match:
	@echo "Run: migrate -path match-service/migrations -database \$$DATABASE_URL up"

migrate-chat:
	@echo "Run: cqlsh -f chat-service/migrations/001_chat.up.sql"

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
