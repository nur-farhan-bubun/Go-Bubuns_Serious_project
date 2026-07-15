<div align="center">

  <h1>Bubun · Social Discovery Platform</h1>

  <p>
    <strong>Real-time dating & social discovery platform</strong><br>
    Go microservices · Next.js frontend · WebSocket real-time · Kafka event bus
  </p>

  <p>
    <a href="#-architecture"><strong>Architecture</strong></a> ·
    <a href="#-services"><strong>Services</strong></a> ·
    <a href="#-tech-stack"><strong>Tech Stack</strong></a> ·
    <a href="#-getting-started"><strong>Getting Started</strong></a> ·
    <a href="#-deployment"><strong>Deployment</strong></a>
  </p>

  <br>

  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go">
  <img alt="Next.js" src="https://img.shields.io/badge/Next.js-15-000000?logo=next.js">
  <img alt="License" src="https://img.shields.io/badge/license-MIT-green">

  <br>
  <br>

</div>

---

> **Status:** Active development. Each service is an independent Go module with its own database, Dockerfile, and scaling profile — built to be truly standalone from day one.

---

## 📋 Table of Contents

- [Architecture Overview](#-architecture-overview)
- [Services Breakdown](#-services-breakdown)
- [Tech Stack](#-tech-stack)
- [Data Flow](#-data-flow)
- [Getting Started](#-getting-started)
- [Makefile Commands](#-makefile-commands)
- [Deployment](#-deployment)
- [Project Structure](#-project-structure)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🏛 Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                Next.js 15 (App Router)                       │
│     Tailwind CSS · Zustand · TanStack Query · maplibre-gl    │
│     WebSocket Client (Chat + Location) · Framer Motion       │
└──────────────────────────┬───────────────────────────────────┘
                           │
                    HTTP / WS (port 3000)
                    ┌──────▼──────┐
                    │ Next.js     │  ← Rewrites /v1/* & /ws
                    │ Rewrites    │     to API Gateway (no CORS)
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  API Gateway  │  ← Echo v4, JWT validation
                    │  (Port 8080)  │     Rate limiting, reverse proxy
                    └──────┬──────┘
                           │
     ┌─────────────────────┼────────────────────┬────────────────────┐
     │                     │                    │                    │
┌────▼────┐          ┌────▼────┐          ┌────▼────┐          ┌────▼────┐
│  User   │          │  Match  │          │  Chat   │          │Location │
│ Service │          │ Service │          │ Service │          │ Service │
│ :8081   │          │ :8082   │          │ :8083   │          │ :8084   │
└────┬────┘          └────┬────┘          └────┬────┘          └────┬────┘
     │                    │                    │                    │
┌────▼────────┐    ┌────▼────────┐    ┌───────▼────────┐    ┌─────▼─────────┐
│ PostgreSQL  │    │ PostgreSQL  │    │   PostgreSQL   │    │   Redis       │
│  (users)    │    │  (matches)  │    │   (messages)   │    │   (GEO)       │
│  :5432      │    │  :5433      │    │   :5433        │    │   :6379       │
└─────────────┘    └─────────────┘    └───────┬────────┘    └───────┬────────┘
                                              │                    │
                                        ┌─────▼────────────────────▼─────┐
                                        │        Redis Pub/Sub          │
                                        │  (cross-instance chat fanout) │
                                        └───────────┬───────────────────┘
                                                    │
                                        ┌───────────▼───────────┐
                                        │   Kafka / Redpanda    │
                                        │  Event Bus (optional) │
                                        └───────────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Standalone Go modules** | Each service has its own `go.mod`, `main.go`, and `Dockerfile` — they share only a `shared/` library for common types, errors, and middleware. |
| **Database per service** | No shared databases. Services communicate only via HTTP/gRPC APIs or Kafka events. Strict isolation enforces bounded contexts. |
| **API Gateway pattern** | Single entry point validates JWT once, then proxies to internal services with `X-User-ID` header. Rate limiting, CORS, and SSL terminate at the edge. |
| **WebSocket sharded hub** | Chat service uses 32 FNV-hash shards for the client hub, reducing lock contention under high concurrency. |
| **Choreography-based SAGA** | Multi-service transactions (e.g., match creation → conversation setup) use Kafka events with compensation handlers for rollback. |

---

## 🧩 Services Breakdown

### 1. API Gateway `(:8080)`

| Layer | Detail |
|-------|--------|
| **Framework** | Echo v4 (stateless) |
| **Auth** | JWT validation (Clerk RS256 / dev HS256) |
| **Purpose** | Central entry point — validates tokens, rate-limits, routes requests |
| **Key files** | `cmd/api/main.go`, `internal/middleware/auth.go`, `internal/handler/gateway.go` |

Routes are registered as reverse proxies to internal services:
```go
api.Any("/users/*", proxyTo(cfg.UserServiceURL))
api.Any("/conversations/*", proxyTo(cfg.ChatServiceURL))
api.Any("/ws", proxyTo(cfg.ChatServiceURL))
api.Any("/location/*", proxyTo(cfg.LocationServiceURL))
```

### 2. User Service `(:8081)`

| Layer | Detail |
|-------|--------|
| **Database** | PostgreSQL 16 |
| **Auth** | Google OAuth, email/password registration |
| **APIs** | User CRUD, dating profiles, profile photos, blocking |
| **Key files** | `internal/service/user_service.go`, `internal/handler/auth_handler.go` |

### 3. Match Service `(:8082)`

| Layer | Detail |
|-------|--------|
| **Database** | PostgreSQL 16 |
| **Purpose** | Swipe matching, discovery algorithm, match state management |
| **Key files** | `internal/service/match_service.go`, `internal/repository/postgres/match_repo.go` |

### 4. Chat Service `(:8083)`

| Layer | Detail |
|-------|--------|
| **Database** | PostgreSQL + Redis |
| **Real-time** | WebSocket (gorilla/websocket) |
| **Streaming** | Kafka (optional event bus) |
| **Purpose** | 1-on-1 & group messaging, presence tracking, WebSocket gateway |
| **Key files** | `internal/websocket/hub.go` (sharded room hub), `internal/websocket/client.go` |

### 5. Location Service `(:8084)`

| Layer | Detail |
|-------|--------|
| **Database** | Redis GEO + PostgreSQL/PostGIS |
| **Real-time** | WebSocket |
| **Purpose** | Live coordinate tracking, map posts with pins, nearby discovery |
| **Key files** | `internal/service/location_service.go`, `internal/websocket/hub.go` |

### 6. Web Frontend `(:3000)`

| Layer | Detail |
|-------|--------|
| **Framework** | Next.js 15 (App Router) |
| **Styling** | Tailwind CSS 3.4 + shadcn/ui |
| **State** | Zustand (client) + TanStack Query (server) |
| **Map** | maplibre-gl 5 + react-map-gl 8 (OpenFreeMap tiles) |
| **Real-time** | Native WebSocket |
| **Animation** | Framer Motion 12 |

Key components: `MapView` (interactive map with user markers & post pins), `ChatOverlay` (collapsible real-time chat), `CreatePostDrawer`, `ThreadDrawer`, `UsersSidebar`.

### 7. Shared Library `(shared/)`

| Package | Purpose |
|---------|---------|
| `pkg/errors/` | Structured `AppError` types (NotFound, BadRequest, etc.) |
| `pkg/logger/` | JSON structured logging via `log/slog` |
| `pkg/middleware/jwt.go` | JWT parsing helpers |
| `pkg/validators/` | Custom go-playground validators |
| `domain/` | Common domain structs |
| `proto/user/` | gRPC protobuf definitions |

---

## 💻 Tech Stack

### Backend (Go)

| Layer | Library |
|-------|---------|
| **Framework** | [`github.com/labstack/echo/v4`](https://echo.labstack.com) |
| **PostgreSQL** | [`github.com/jackc/pgx/v5`](https://github.com/jackc/pgx) |
| **Redis** | [`github.com/redis/go-redis/v9`](https://github.com/redis/go-redis) |
| **WebSocket** | [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket) |
| **Kafka** | [`github.com/segmentio/kafka-go`](https://github.com/segmentio/kafka-go) |
| **gRPC** | [`google.golang.org/grpc`](https://grpc.io) |
| **JWT** | [`github.com/golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) |
| **Validation** | [`github.com/go-playground/validator/v10`](https://github.com/go-playground/validator) |
| **Migrations** | [`github.com/golang-migrate/migrate/v4`](https://github.com/golang-migrate/migrate) |
| **Config** | [`github.com/caarlos0/env/v10`](https://github.com/caarlos0/env) |
| **Logging** | `log/slog` (stdlib) |

### Frontend (Next.js)

| Layer | Library |
|-------|---------|
| **Framework** | Next.js 15 (App Router) |
| **Styling** | Tailwind CSS 3.4 + shadcn/ui |
| **Client State** | [Zustand 5](https://github.com/pmndrs/zustand) |
| **Server State** | [TanStack Query](https://tanstack.com/query) |
| **Map** | [maplibre-gl 5](https://maplibre.org) + [react-map-gl 8](https://visgl.github.io/react-map-gl) |
| **Animation** | [Framer Motion 12](https://motion.dev) |
| **Drawer** | [Vaul](https://github.com/emilkowalski/vaul) |

### Infrastructure

| Component | Technology |
|-----------|------------|
| **Orchestration** | Docker Compose (dev) / Kubernetes (prod) |
| **Databases** | PostgreSQL 16, PostGIS 16-3.4, Redis 7 |
| **Event Bus** | Kafka 7.7 + ZooKeeper |
| **CI/CD** | GitHub Actions |
| **Load Balancing** | Ingress (Kubernetes) / Next.js rewrites (dev) |

---

## 🔄 Data Flow

### Message Send (Real-time WebSocket)

```
User types message
       │
       ▼
WebSocket ──► API Gateway ──► Chat Service
                                    │
                              ┌─────▼─────┐
                              │  Validate  │
                              │  & Save    │──► PostgreSQL (persist)
                              └─────┬─────┘
                                    │
                         ┌──────────▼──────────┐
                         │  Fan-out via Hub     │
                         │  (single marshal)    │
                         │  + Redis PubSub for  │
                         │  cross-instance      │
                         └──────────┬──────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
              ┌─────▼─────┐   ┌─────▼─────┐   ┌─────▼─────┐
              │ Local     │   │ Redis      │   │ Kafka     │
              │ Room      │   │ PubSub     │   │ (optional)│
              │ Clients   │   │ (other     │   │           │
              │           │   │  instances)│   │           │
              └───────────┘   └───────────┘   └───────────┘
```

### Auth Flow

```
1. User visits /login
       │
       ▼
2a. Google OAuth → user-service → Google → callback → JWT
    OR
2b. Simulated login → frontend generates HS256 JWT
       │
       ▼
3. Token stored in localStorage
4. WebSocket connects with ?token= query param
5. API Gateway validates token → sets X-User-ID header → proxies request
```

### Map Post Flow

```
User creates post via UI
       │
       ▼
POST /v1/posts ──► API Gateway ──► Location Service
                                          │
                                    ┌─────▼─────┐
                                    │ Save to   │
                                    │ PostGIS   │
                                    └───────────┘
       │
       ▼
Other users see pin on map (GET /v1/posts?lat=...&lng=...&radius=...)
```

---

## 🚀 Getting Started

### Prerequisites

- **Go** 1.22+
- **Docker** & **Docker Compose**
- **Node.js** 20+ (for the frontend)
- **kubectl** (for Kubernetes deployment)
- **Tilt** (optional, for local K8s dev)

### Quick Start (Docker Compose)

```bash
# Clone the repository
git clone git@github.com:nur-farhan-bubun/Go-Bubuns_Serious_project.git
cd microservices-go-starter

# Initialize Go workspace (one time)
go work init
go work use ./api-gateway ./user-service ./match-service ./chat-service ./location-service ./shared

# Start all services + infrastructure
make up

# Run database migrations
make migrate-up

# Check all services are healthy
make logs
```

### Start Frontend

```bash
cd web
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) — the Next.js rewrites proxy API calls to the backend automatically.

### Run Individual Services (for debugging)

```bash
# Run a specific service locally
make run-user-service

# Or directly
cd user-service && go run ./cmd/api
```

---

## 📜 Makefile Commands

| Command | Description |
|---------|-------------|
| `make build-all` | Build all Go services |
| `make build-<service>` | Build a specific service |
| `make run-all` | Start all services via Docker Compose |
| `make run-<service>` | Run a service locally |
| `make up` | `docker compose up -d` |
| `make down` | `docker compose down` |
| `make logs` | Tail Docker Compose logs |
| `make test-all` | Run all tests across services |
| `make test-<service>` | Run tests for a specific service |
| `make vet-all` | Run `go vet` on all modules |
| `make tidy-all` | Run `go mod tidy` on all modules |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Roll back the last migration |
| `make docker-build-all` | Build all Docker images |
| `make clean` | Remove build artifacts |
| `make compile-proto` | Compile gRPC protobuf definitions |

---

## ☸️ Deployment

### Local Kubernetes (Minikube / Docker Desktop)

```bash
# Using Tilt (hot-reload)
tilt up

# Monitor pods
kubectl get pods

# Or with Minikube
minikube dashboard
```

### Production (Google Cloud Example)

```bash
# 1. Build and push Docker images to Artifact Registry
docker build -t {REGION}-docker.pkg.dev/{PROJECT_ID}/ride-sharing/api-gateway:latest \
  --platform linux/amd64 -f api-gateway/Dockerfile .

# 2. Create a GKE cluster
# 3. Apply Kubernetes manifests
kubectl apply -f k8s/Db/       # Databases (PostgreSQL, Redis)
kubectl apply -f k8s/           # Services (API Gateway, etc.)
```

The [`.github/workflows/prod-cicd.yml`](.github/workflows/prod-cicd.yml) contains the CI/CD pipeline configuration.

---

## 📁 Project Structure

```
microservices-go-starter/
├── docker-compose.yml        # Local dev: all services + infrastructure
├── go.work                   # Go workspace (links all modules)
├── Makefile                  # Build, test, migration, deployment commands
├── Tiltfile                  # Local K8s development (hot-reload)
│
├── api-gateway/              # Edge: JWT validation, routing, rate limiting
├── user-service/             # Users, profiles, auth, blocking
├── match-service/            # Swipes, matches, discovery
├── chat-service/             # Real-time messaging, presence, WebSocket hub
├── location-service/         # Geo tracking, map posts, nearby queries
├── shared/                   # Common library (errors, logger, middleware)
│
├── web/                      # Next.js 15 frontend
├── docs/                     # Architecture documentation
├── infra/                    # Dockerfiles & K8s manifests
├── k8s/                      # Kubernetes deployment configs
├── proto/                    # Protobuf definitions
└── scripts/                  # Utility scripts
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Each service is a standalone Go module — add dependencies via `go mod tidy` in that service's directory
- Run `make vet-all` and `make test-all` before pushing
- Follow the database-per-service rule: no cross-service table joins
- Use structured `log/slog` for all logging
- Keep WebSocket read/write pumps as goroutines with backpressure channels

---

## 📄 License

This project is licensed under the MIT License.

---

<div align="center">
  <sub>Built with Go, Next.js, and ☕</sub>
</div>
