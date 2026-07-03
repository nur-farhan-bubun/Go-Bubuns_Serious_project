# Dating App — Microservices Architecture

A production-ready dating app backend built with **Go (Echo)** and **Next.js**, designed as **standalone microservices** from day one. Each service is its own Go module with independent deploy, database, and scaling.

---

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│              Next.js 14+ (App Router)         │
│     Tailwind + TanStack Query + Zustand       │
│         WebSocket client for real-time chat   │
└──────────────────┬────────────────────────────┘
                   │ HTTPS / REST / JSON / WS
          ┌────────▼────────┐
          │   Nginx / Caddy │  ← SSL, rate limit, static assets
          │   (or Vercel)   │
          └────────┬────────┘
                   │
          ┌────────▼────────┐
          │  API Gateway    │  ← Echo, JWT validation, routing
          │  (Echo)         │
          └────────┬────────┘
                   │
    ┌──────────────┼──────────────┬──────────────┐
    │              │              │              │
┌───▼───┐   ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
│ User  │   │ Match   │   │  Chat   │   │Location │
│ Svc   │   │ Svc     │   │  Svc    │   │ Svc     │
│:8081  │   │ :8082   │   │ :8083   │   │ :8084   │
└───┬───┘   └────┬────┘   └────┬────┘   └────┬────┘
    │            │             │             │
┌───▼───┐   ┌────▼────┐   ┌──▼─────┐   ┌───▼────┐
│PostgreSQL│  │PostgreSQL│   │ScyllaDB│   │ Redis  │
│(users)  │   │(matches) │   │(chat)  │   │ (GEO)  │
└─────────┘   └─────────┘   └────────┘   └────────┘
    │            │             │             │
    └────────────┴─────────────┴─────────────┘
                   │
            ┌──────▼──────┐
            │   Kafka /   │  ← Event bus (match.created, message.sent)
            │  Redis Pub  │
            └─────────────┘
                   │
            ┌──────▼──────┐
            │  S3 / GCS   │  ← Profile photos (pre-signed URLs)
            └─────────────┘
```

---

## Services

| Service | Port | Database | Responsibility |
|---------|------|----------|----------------|
| **API Gateway** | `:8080` | None | JWT validation, rate limiting, routing to services |
| **User Service** | `:8081` | PostgreSQL | Profiles, auth webhooks, photo uploads |
| **Match Service** | `:8082` | PostgreSQL | Swipes, matches, discovery algorithm |
| **Chat Service** | `:8083` | ScyllaDB + Redis | Messages, conversations, WebSocket gateway |
| **Location Service** | `:8084` | Redis GEO | Live coordinates, nearby discovery |
| **Shared** | — | None | Common errors, logger, JWT middleware, domain structs |

---

## Monorepo Structure

```
dating-app/
├── docker-compose.yml              # Local dev: all services + infra
├── go.work                         # Go workspace (links all modules)
├── Makefile                        # Build / test / deploy all services
│
├── api-gateway/                    # API Gateway (single entry point)
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── handler/gateway.go      # Reverse proxy to services
│   │   └── middleware/
│   │       ├── auth.go             # Clerk JWT validation (once)
│   │       └── rate_limit.go
│   ├── go.mod
│   └── Dockerfile
│
├── user-service/                   # User profiles & auth webhooks
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── domain/user.go
│   │   ├── repository/postgres/
│   │   │   └── user_repo.go
│   │   ├── service/user_service.go
│   │   └── handler/user_handler.go
│   ├── migrations/001_users.up.sql
│   ├── go.mod
│   └── Dockerfile
│
├── match-service/                  # Swipes, matches, discovery
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── domain/match.go
│   │   ├── repository/postgres/
│   │   │   └── match_repo.go
│   │   ├── service/match_service.go
│   │   └── handler/match_handler.go
│   ├── migrations/001_matches.up.sql
│   ├── go.mod
│   └── Dockerfile
│
├── chat-service/                   # Real-time messaging
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── domain/chat.go
│   │   ├── repository/
│   │   │   ├── scylladb/chat_repo.go
│   │   │   └── redis/presence_repo.go
│   │   ├── service/chat_service.go
│   │   ├── handler/
│   │   │   ├── chat_handler.go
│   │   │   └── ws_handler.go
│   │   └── websocket/hub.go
│   ├── migrations/001_chat.up.sql
│   ├── go.mod
│   └── Dockerfile
│
├── location-service/               # Geo queries & presence
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── domain/location.go
│   │   ├── repository/redis/
│   │   │   ├── location_repo.go
│   │   │   └── presence_repo.go
│   │   ├── service/location_service.go
│   │   └── handler/location_handler.go
│   ├── go.mod
│   └── Dockerfile
│
└── shared/                         # Common library (no main.go)
    ├── pkg/
    │   ├── errors/errors.go        # Custom error types
    │   ├── logger/logger.go        # Structured slog setup
    │   ├── middleware/jwt.go       # JWT parsing helpers
    │   └── validators/validators.go # Custom go-playground validators
    ├── domain/                     # Shared domain structs (optional)
    │   └── common.go
    └── go.mod
```

---

## Inter-Service Communication

### Synchronous (HTTP / gRPC)
Used when one service needs data from another **right now**.

| Caller | Calls | For |
|--------|-------|-----|
| Match Service | `GET user-service:8081/v1/internal/users/:id` | Fetch profile during swipe |
| Chat Service | `GET match-service:8082/v1/internal/matches/:id` | Verify match before creating conversation |
| API Gateway | All services | Route requests, aggregate responses |

### Asynchronous (Kafka / Redis PubSub)
Used for events that other services react to **eventually**.

| Event | Publisher | Consumers |
|-------|-----------|-----------|
| `user.created` | User Service | Match Service (index for discovery), Chat Service |
| `match.created` | Match Service | Chat Service (auto-create conversation), Notification Service |
| `message.sent` | Chat Service | Notification Service (push notification) |
| `location.updated` | Location Service | Match Service (update discovery index) |

---

## API Gateway Routing

```go
// api-gateway/internal/handler/gateway.go
func RegisterRoutes(e *echo.Echo, cfg *config.Config) {
    // Public
    e.GET("/health", healthCheck)

    // Protected
    api := e.Group("/v1")
    api.Use(middleware.AuthMiddleware(cfg.ClerkIssuer, cfg.ClerkKey))

    // Proxy to services
    api.Any("/users/*", proxyTo(cfg.UserServiceURL))
    api.Any("/me/*", proxyTo(cfg.UserServiceURL))
    api.Any("/swipe", proxyTo(cfg.MatchServiceURL))
    api.Any("/matches/*", proxyTo(cfg.MatchServiceURL))
    api.Any("/discover", proxyTo(cfg.MatchServiceURL)) // or Location Service
    api.Any("/messages/*", proxyTo(cfg.ChatServiceURL))
    api.Any("/ws", proxyTo(cfg.ChatServiceURL))        // WebSocket passthrough
    api.Any("/location/*", proxyTo(cfg.LocationServiceURL))
}
```

**Headers passed downstream:**
- `X-User-ID` — extracted from Clerk JWT
- `X-Request-ID` — for distributed tracing

---

## Database Per Service

| Service | Database | Schema | Reason |
|---------|----------|--------|--------|
| User Service | PostgreSQL | `app_users` | Relational user data, profiles |
| Match Service | PostgreSQL | `app_matches` | Swipes, matches (transactional) |
| Chat Service | ScyllaDB | `app_chat` | Write-heavy messages, time-series |
| Location Service | Redis | GEO keys | Sub-millisecond geo queries |

**Rule:** Services do not share tables. If Match Service needs a user's name, it calls User Service via HTTP — it does not join the `users` table.

---

## WebSocket Architecture (Chat Service)

```
Client ──WS──► API Gateway ──WS──► Chat Service Instance A
                                      │
                                      ├──► Redis Pub/Sub ("chat:room:{id}")
                                      │
Client ──WS──► API Gateway ──WS──► Chat Service Instance B
```

- **Stateless:** Any Chat Service instance can handle any connection.
- **Redis Pub/Sub:** Broadcasts messages across instances.
- **Auth at connection time:** JWT validated in API Gateway, passed as `X-User-ID` header.

---

## Tech Stack

### Backend (Go)
| Layer | Library |
|-------|---------|
| **Framework** | `github.com/labstack/echo/v4` |
| **Database** | `github.com/jackc/pgx/v5` (PostgreSQL), `github.com/gocql/gocql` (ScyllaDB) |
| **Redis** | `github.com/redis/go-redis/v9` |
| **WebSocket** | `github.com/gorilla/websocket` |
| **Validation** | `github.com/go-playground/validator/v10` |
| **Auth** | **Clerk** (external) — JWT validated in API Gateway |
| **Config** | `github.com/caarlos0/env/v10` |
| **Logging** | `log/slog` (stdlib) + OpenTelemetry hooks |
| **Migrations** | `github.com/golang-migrate/migrate` |
| **HTTP Client** | `github.com/go-resty/resty/v2` (service-to-service calls) |
| **Events** | `github.com/segmentio/kafka-go` or `github.com/redis/go-redis/v9` (PubSub) |

### Frontend (Next.js)
| Layer | Library |
|-------|---------|
| **Framework** | Next.js 14+ App Router |
| **Styling** | Tailwind CSS |
| **Server State** | TanStack Query |
| **Client State** | Zustand |
| **Forms** | React Hook Form + Zod |
| **Real-time** | Native WebSocket |
| **HTTP Client** | `ky` |

---

## Design Decisions

### 1. Standalone Microservices (Separate Go Modules)
Each service is a **separate Go module** with its own `go.mod`, `main.go`, and `Dockerfile`. They share nothing except the `shared/` library.

### 2. API Gateway Pattern
- **Single entry point** for all clients (Web + Mobile).
- **JWT validation happens once** here, not in every service.
- **Rate limiting, SSL, CORS** handled at the edge.
- Services trust internal requests (mTLS or VPC in production).

### 3. Interfaces for Cross-Service Calls
Each service defines repository interfaces. When extracting or calling another service, you implement the same interface over HTTP:

```go
// Match Service needs user data
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*domain.User, error)
}

// Implementation A: PostgreSQL (monolith phase)
type userRepo struct { db *pgxpool.Pool }

// Implementation B: HTTP Client (microservice phase)
type userClient struct { baseURL string }
// Both implement UserRepository — MatchService doesn't care which one.
```

### 4. Event-Driven for Loose Coupling
Services publish events to Kafka/Redis instead of calling each other directly when eventual consistency is acceptable:
- Match created → Chat Service creates conversation (async).
- User deleted → All services purge data (async).

### 5. Database Per Service
No shared database. Each service owns its data. This is non-negotiable for microservices.

---

## Local Development

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- Node.js 20+ (for Next.js frontend)

### Start All Services
```bash
# Clone and enter project
cd dating-app

# Initialize Go workspace (one time)
go work init
go work use ./api-gateway ./user-service ./match-service ./chat-service ./location-service ./shared

# Start all services + infrastructure
docker-compose up -d

# Run migrations for each service
make migrate-user
make migrate-match
make migrate-chat

# Or start individual services for debugging
cd user-service && go run ./cmd/api
cd match-service && go run ./cmd/api
```

### Environment Variables (`.env` per service)

**API Gateway:**
```env
PORT=8080
USER_SERVICE_URL=http://user-service:8081
MATCH_SERVICE_URL=http://match-service:8082
CHAT_SERVICE_URL=http://chat-service:8083
LOCATION_SERVICE_URL=http://location-service:8084
CLERK_JWT_ISSUER=https://your-clerk-domain.clerk.accounts.dev
CLERK_JWT_KEY=-----BEGIN PUBLIC KEY-----\n...
```

**User Service:**
```env
PORT=8081
DATABASE_URL=postgres://postgres:postgres@user-db:5432/users?sslmode=disable
S3_BUCKET=your-bucket
S3_REGION=us-east-1
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx
```

**Match Service:**
```env
PORT=8082
DATABASE_URL=postgres://postgres:postgres@match-db:5432/matches?sslmode=disable
USER_SERVICE_URL=http://user-service:8081
KAFKA_BROKERS=kafka:9092
```

**Chat Service:**
```env
PORT=8083
DATABASE_URL=scylladb://chat-db:9042/chat
REDIS_URL=redis:6379
KAFKA_BROKERS=kafka:9092
```

**Location Service:**
```env
PORT=8084
REDIS_URL=redis:6379
```

---

## Deployment

### Docker Compose (Local / Demo)
```bash
docker-compose -f docker-compose.yml up -d
```

### Kubernetes (Production)
Each service has its own deployment manifest:

```yaml
# k8s/user-service-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: user-service
        image: user-service:v1.2.0
        envFrom:
        - secretRef:
            name: user-service-secrets
```

### Managed Cloud (Recommended for MVP)
| Component | Service |
|-----------|---------|
| **API Gateway** | AWS API Gateway or self-hosted on ECS/Fargate |
| **User/Match Service** | AWS ECS Fargate + RDS PostgreSQL |
| **Chat Service** | AWS ECS Fargate + ScyllaDB Cloud / Keyspaces |
| **Location Service** | AWS ECS Fargate + ElastiCache Redis |
| **Events** | Confluent Cloud (Kafka) or AWS MSK |
| **Files** | AWS S3 + CloudFront |
| **Frontend** | Vercel |

---

## Scaling Path

| Stage | Architecture | Trigger |
|-------|-------------|---------|
| **MVP** | All services on one machine (`docker-compose`) | 0–1k users |
| **Growth** | Separate containers, shared managed DBs | 1k–50k users |
| **Scale** | K8s, DB per service, read replicas | 50k–500k users |
| **Global** | Multi-region, CDN, Kafka MirrorMaker | 500k+ users |

---

## Monitoring & Observability

| Layer | Tool | Purpose |
|-------|------|---------|
| **Logs** | Loki / CloudWatch | Centralized from all services |
| **Metrics** | Prometheus + Grafana | Request latency, DB connections, WS clients |
| **Traces** | Jaeger / Tempo | Distributed trace across service boundaries |
| **Alerts** | PagerDuty / Slack | 500 errors > 1%, DB CPU > 80%, Kafka lag |

**Trace propagation:** API Gateway generates `X-Trace-ID`, passes it to all services via headers.

---

## Security Checklist

- [ ] **mTLS** between services (Linkerd / Istio in K8s)
- [ ] **OAuth2 + PKCE** for mobile apps (handled by Clerk)
- [ ] **PII Encryption:** User photos at rest (S3 SSE-KMS), DMs in transit (TLS 1.3)
- [ ] **Rate Limiting:** 100 req/min per IP, 1000 req/min per user (API Gateway)
- [ ] **Input Validation:** Strict request validation in every service
- [ ] **Image Moderation:** AWS Rekognition / Google Vision API on upload
- [ ] **GDPR:** Data deletion SAGA (right to be forgotten across all services)

---

## License

MIT
