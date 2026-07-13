# 🏗️ Project Architecture & Deployment Plan

> **Microservices Dating App** — Go (Echo) Backend + Next.js Frontend
> Real-time chat, live location tracking, interactive map with posts, and matchmaking.

---

## 📋 Table of Contents

1. [Project Overview](#-project-overview)
2. [Architecture Diagram](#-architecture-diagram)
3. [Services Breakdown](#-services-breakdown)
4. [Tech Stack](#-tech-stack)
5. [Data Flow](#-data-flow)
6. [Database Schemas](#-database-schemas)
7. [API Routes](#-api-routes)
8. [Environment Variables](#-environment-variables)
9. [Local Development Setup](#-local-development-setup)
10. [Deployment Options](#-deployment-options)
11. [CI/CD Pipeline](#-cicd-pipeline)
12. [Monitoring & Observability](#-monitoring--observability)
13. [Security Checklist](#-security-checklist)
14. [Scaling Strategy](#-scaling-strategy)

---

## 🚀 Project Overview

A **real-time dating/social discovery platform** built as **standalone microservices** from day one. Each service is an independent Go module with its own database, Dockerfile, and scaling characteristics.

### Core Features

| Feature | Service(s) | Real-Time? |
|---------|-----------|-----------|
| User registration & auth | User Service + API Gateway | ❌ |
| Google OAuth login | User Service | ❌ |
| User blocking | User Service | ❌ |
| 1-on-1 & group messaging | Chat Service | ✅ WebSocket |
| User presence (online/offline) | Chat Service | ✅ WebSocket |
| Interactive map with posts | Location Service | ✅ WebSocket |
| Post threads with comments | Frontend (Zustand) | ❌ |
| User matching / swipe | Match Service | ❌ |
| Live location tracking | Location Service | ✅ WebSocket |

---

## 🏛 Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                    Next.js 15 (App Router)                       │
│         Tailwind CSS · Framer Motion · Zustand · TanStack       │
│       maplibre-gl (Map) · WebSocket Client (Chat + Location)    │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                    HTTP / WS (port 3000)
                           │
                    ┌──────▼──────┐
                    │ Next.js     │  ← Rewrites /v1/* → localhost:8080
                    │ Rewrites    │     Rewrites /ws   → localhost:8080
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  API Gateway  │  ← Echo, JWT validation, reverse proxy
                    │  (Port 8080)  │     Rate limiting, route aggregation
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
│ PostgreSQL  │    │ PostgreSQL  │    │   ScyllaDB     │    │   Redis       │
│  (users)    │    │  (matches)  │    │   (messages)   │    │   (GEO)       │
│  Port 5432  │    │  Port 5433  │    │   Port 9042    │    │   Port 6379   │
└─────────────┘    └─────────────┘    └───────┬────────┘    └───────┬────────┘
                                              │                    │
                                        ┌─────▼────────────────────▼─────┐
                                        │        Redis Pub/Sub          │
                                        │  ("chat:room:{id}" fan-out)   │
                                        └───────────┬───────────────────┘
                                                    │
                                        ┌───────────▼───────────┐
                                        │   Kafka / Redpanda    │
                                        │  Event Bus (optional) │
                                        └───────────────────────┘
```

---

## 🧩 Services Breakdown

### 1. **API Gateway** (`api-gateway/`)

| Property | Value |
|----------|-------|
| **Port** | `:8080` |
| **Framework** | Echo v4 |
| **Database** | None (stateless) |
| **Auth** | JWT validation (Clerk RS256 / dev HS256) |
| **Purpose** | Single entry point — JWT validation, rate limiting, reverse proxy |

**Key Files:**

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point, server start |
| `internal/config/config.go` | Env-based config |
| `internal/handler/gateway.go` | Route registration + reverse proxy |
| `internal/middleware/auth.go` | JWT validation (RS256/HS256) + DevAuthMiddleware |
| `internal/middleware/rate_limit.go` | Rate limiting |

**Routing Logic:**
```go
// Dev mode: no signature verification (CLERK_JWT_KEY empty)
// Production: RS256/HS256 validation via Clerk public key
api.Any("/users/*", proxyTo(cfg.UserServiceURL))
api.Any("/conversations/*", proxyTo(cfg.ChatServiceURL))
api.Any("/ws", proxyTo(cfg.ChatServiceURL))            // WebSocket passthrough
api.Any("/location/*", proxyTo(cfg.LocationServiceURL))
api.Any("/posts/*", proxyTo(cfg.LocationServiceURL))
```

### 2. **User Service** (`user-service/`)

| Property | Value |
|----------|-------|
| **Port** | `:8081`, gRPC `:50051` |
| **Database** | PostgreSQL 16 |
| **Auth** | Google OAuth, email/password |
| **Purpose** | User CRUD, profiles, blocking, Google OAuth |

**Key Files:**

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point |
| `internal/domain/user.go` | Domain models |
| `internal/handler/auth_handler.go` | Google OAuth flow |
| `internal/handler/user_handler.go` | User CRUD + blocking |
| `internal/service/user_service.go` | Business logic |
| `internal/repository/postgres/` | User, profile, block repos |
| `internal/kafka/producer.go` | Kafka user events |

### 3. **Chat Service** (`chat-service/`)

| Property | Value |
|----------|-------|
| **Port** | `:8083` |
| **Database** | ScyllaDB + Redis |
| **Real-time** | WebSocket (Gorilla) |
| **Streaming** | Kafka (optional event bus) |
| **Purpose** | Messaging, presence, group chats |

**Key Files:**

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point |
| `internal/websocket/hub.go` | **Sharded room hub** — core WebSocket manager |
| `internal/websocket/client.go` | Read/Write pumps, backpressure, heartbeat |
| `internal/handler/ws_handler.go` | WebSocket upgrade + message routing |
| `internal/handler/openapi_handler.go` | REST API for messages + conversations |
| `internal/service/chat_service.go` | Business logic |
| `internal/repository/scylladb/chat_repo.go` | ScyllaDB message persistence |
| `internal/repository/redis/presence_repo.go` | Redis presence tracking |
| `internal/kafka/consumer.go` | Kafka consumer for user events |

**WebSocket Architecture:**
```
Client ──WS──► API Gateway ──WS──► Chat Service
                                       │
                                  ┌────▼────┐
                                  │  Hub    │  ← 32 shards (FNV hash)
                                  │  Shards │     Reduces lock contention
                                  └────┬────┘
                                       │
                              ┌────────▼────────┐
                              │  Room Registry   │
                              │  map[roomID] →   │
                              │  set[*Client]    │
                              └────────┬────────┘
                                       │
                              ┌────────▼────────┐
                              │   Client Pumps   │
                              │  ReadPump (1)    │
                              │  WritePump (1)   │ ← Single writer rule
                              │  send chan[256]  │
                              └──────────────────┘
```

### 4. **Location Service** (`location-service/`)

| Property | Value |
|----------|-------|
| **Port** | `:8084` |
| **Database** | Redis (GEO) + PostgreSQL/PostGIS |
| **Real-time** | WebSocket |
| **Purpose** | Real-time location, map posts, nearby queries |

**Key Files:**

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point |
| `internal/service/location_service.go` | Location business logic |
| `internal/repository/redis/location_repo.go` | Redis GEO commands |
| `internal/repository/postgres/map_post_repo.go` | PostGIS map posts |
| `internal/websocket/hub.go` | Location broadcast hub |

### 5. **Match Service** (`match-service/`)

| Property | Value |
|----------|-------|
| **Port** | `:8082` |
| **Database** | PostgreSQL |
| **Purpose** | Swipe matching |

**Key Files:**

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Entry point |
| `internal/service/match_service.go` | Matching algorithm |
| `internal/repository/postgres/match_repo.go` | Match persistence |

### 6. **Web Frontend** (`web/`)

| Property | Value |
|----------|-------|
| **Port** | `:3000` |
| **Framework** | Next.js 15 (App Router) |
| **Styling** | Tailwind CSS + shadcn/ui |
| **State** | Zustand (client) + TanStack Query (server) |
| **Map** | maplibre-gl + react-map-gl |
| **Real-time** | Native WebSocket |
| **Animation** | Framer Motion |

**Key Components:**

| Component | Purpose |
|-----------|---------|
| `MapView.tsx` | Interactive map with markers, pins, user avatars |
| `ChatOverlay.tsx` | Collapsible chat panel with conversation list |
| `ChatFeed.tsx` | Message thread with real-time updates |
| `CreatePostDrawer.tsx` | Create map posts with categories |
| `ThreadDrawer.tsx` | Post details with comments, voting, sharing |
| `WorkspaceBar.tsx` | Side navigation bar |
| `UsersSidebar.tsx` | User directory + search |
| `UserProfileView.tsx` | User profile + block/unblock |
| `CreateGroupDialog.tsx` | Group conversation creation |

**Auth Flow:**
1. User lands on login page → registers/logs in
2. Frontend generates HS256 JWT (dev) or gets OAuth token from backend
3. Token stored in `localStorage`
4. Every API call includes `Authorization: Bearer <token>`
5. WebSocket connections pass token via `?token=` query param
6. Next.js rewrites proxy `/v1/*` → `localhost:8080` to avoid CORS

### 7. **Shared Library** (`shared/`)

| Package | Purpose |
|---------|---------|
| `pkg/errors/` | Structured AppError types (NotFound, BadRequest, etc.) |
| `pkg/logger/` | JSON structured logger via `log/slog` |
| `pkg/middleware/jwt.go` | JWT parsing helpers |
| `pkg/validators/` | Custom go-playground validators |
| `domain/` | Common domain structs (Pagination, etc.) |
| `proto/user/` | gRPC protobuf definitions |

---

## 💻 Tech Stack

### Backend (Go)

| Layer | Library | Purpose |
|-------|---------|---------|
| **Framework** | `github.com/labstack/echo/v4` | HTTP router & middleware |
| **JWT** | `github.com/golang-jwt/jwt/v5` | Token validation (RS256/HS256) |
| **PostgreSQL** | `github.com/jackc/pgx/v5` | High-performance PG driver |
| **ScyllaDB** | `github.com/gocql/gocql` | Cassandra-compatible driver |
| **Redis** | `github.com/redis/go-redis/v9` | Client + PubSub + GEO |
| **WebSocket** | `github.com/gorilla/websocket` | Real-time connections |
| **Kafka** | `github.com/segmentio/kafka-go` | Event bus producer/consumer |
| **gRPC** | `google.golang.org/grpc` | Inter-service RPC |
| **Validation** | `github.com/go-playground/validator/v10` | Request validation |
| **Config** | `github.com/caarlos0/env/v10` | Env variable parsing |
| **Logging** | `log/slog` (stdlib) | Structured JSON logging |
| **Migrations** | `github.com/golang-migrate/migrate/v4` | DB schema management |

### Frontend (Next.js)

| Layer | Library | Purpose |
|-------|---------|---------|
| **Framework** | Next.js 15 (App Router) | Full-stack React framework |
| **Styling** | Tailwind CSS 3.4 | Utility-first CSS |
| **UI Components** | shadcn/ui (Radix) | Accessible primitives |
| **Client State** | Zustand 5 | Lightweight global state |
| **Server State** | TanStack Query | Caching & data fetching |
| **Map** | maplibre-gl 5 + react-map-gl 8 | Open-source map tiles |
| **Animation** | Framer Motion 12 | Smooth transitions |
| **Drawer** | Vaul | Bottom sheet drawer |
| **Icons** | Emoji/Unicode | Inline emoji icons |

### Infrastructure

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Orchestration** | Docker Compose (dev) / K8s (prod) | Container management |
| **PostgreSQL** | postgres:16-alpine | User, match, location data |
| **ScyllaDB** | scylladb/scylla:5.4 | Chat messages (write-heavy) |
| **Redis** | redis:7-alpine | Presence, GEO, PubSub |
| **PostGIS** | postgis/postgis:16-3.4 | Geo-spatial map posts |
| **Kafka** | confluentinc/cp-kafka:7.7 | Event bus |
| **ZooKeeper** | confluentinc/cp-zookeeper:7.7 | Kafka coordination |
| **Tilt** | Tiltfile | Local dev hot-reload (K8s) |

---

## 🔄 Data Flow

### Message Send Flow (Real-time)
```
Alice types message
       │
       ▼
Client WebSocket ──► API Gateway ──► Chat Service
                                          │
                                    ┌─────▼─────┐
                                    │ ws_handler │
                                    │ Parse msg  │
                                    └─────┬─────┘
                                          │
                               ┌──────────▼──────────┐
                               │  Chat Service        │
                               │  1. Save to ScyllaDB │
                               │  2. Publish to Redis │
                               │     PubSub channel   │
                               │  3. Fan-out via Hub  │
                               │     (single marshal) │
                               └──────────┬──────────┘
                                          │
                    ┌─────────────────────┼──────────────────┐
                    │                     │                  │
              ┌─────▼─────┐         ┌─────▼─────┐    ┌──────▼──────┐
              │Hub.SendTo │         │ Redis      │    │ Kafka       │
              │ Room()    │         │ PubSub     │    │ Producer    │
              │ (local)   │         │ (cross-    │    │ (optional)  │
              │           │         │  instance) │    │             │
              └───────────┘         └───────────┘    └─────────────┘
```

### Presence Flow
```
User connects WebSocket
       │
       ▼
Hub.Register(client)
       │
       ▼
OnPresenceChange("user_123", "online")
       │
       ▼
Hub.BroadcastPresence() ──► All connected clients
                              (presence_update event)

User disconnects
       │
       ▼
Hub.Unregister(client) ──► OnPresenceChange("user_123", "offline")
                              ──► Broadcast to all clients
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
Other users see pin on map
       │
       ▼
GET /v1/posts?lat=...&lng=...&radius=...
       │
       ▼
Frontend renders seed posts + API posts
```

### Auth Flow
```
1. User visits /login
       │
       ▼
2a. Google OAuth: redirect → user-service → Google → callback → JWT
    OR
2b. Simulated login: frontend creates HS256 JWT
       │
       ▼
3. Token stored in localStorage
4. WebSocket connects with ?token= query param
5. API Gateway validates token → sets X-User-ID header → proxy to service
```

---

## 🗄 Database Schemas

### User Service — PostgreSQL (`users`)

**`app_users`**
```sql
CREATE TABLE app_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**`dating_profiles`**
```sql
CREATE TABLE dating_profiles (
    user_id      UUID PRIMARY KEY REFERENCES app_users(id),
    display_name VARCHAR(100),
    bio          TEXT,
    age          INT,
    gender       VARCHAR(50),
    location     VARCHAR(255),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);
```

**`profile_photos`**
```sql
CREATE TABLE profile_photos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES app_users(id),
    url        TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**`blocked_users`**
```sql
CREATE TABLE blocked_users (
    blocker_id UUID NOT NULL REFERENCES app_users(id),
    blocked_id UUID NOT NULL REFERENCES app_users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id)
);
```

### Chat Service — ScyllaDB

```cql
CREATE KEYSPACE app_chat WITH replication = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
};

CREATE TABLE app_chat.messages (
    conversation_id UUID,
    created_at      TIMESTAMP,
    message_id      UUID,
    sender_id       UUID,
    content         TEXT,
    PRIMARY KEY ((conversation_id), created_at, message_id)
) WITH CLUSTERING ORDER BY (created_at DESC);

CREATE TABLE app_chat.conversations (
    conversation_id UUID PRIMARY KEY,
    type            TEXT,        -- 'direct' or 'group'
    name            TEXT,
    user1_id        UUID,
    user2_id        UUID,
    match_id        UUID,
    member_ids      SET<UUID>,
    created_at      TIMESTAMP
);

CREATE TABLE app_chat.groups (
    group_id   UUID PRIMARY KEY,
    name       TEXT,
    owner_id   UUID,
    member_ids SET<UUID>,
    created_at TIMESTAMP
);
```

### Match Service — PostgreSQL (`matches`)

```sql
CREATE TABLE matches (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id   UUID NOT NULL,
    user2_id   UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user1_id, user2_id)
);
```

### Location Service — Redis (GEO)

```
GEOADD locations <longitude> <latitude> <user_id>
              
Key: user:<user_id>:location
Value: {latitude, longitude, updated_at}

PubSub channel: "location:updates"
```

### Location Service — PostGIS (`locations`)

```sql
CREATE TABLE map_posts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    title       VARCHAR(255) NOT NULL,
    content     TEXT,
    category    VARCHAR(50) NOT NULL, -- 'RESTAURANT', 'SOCIAL_LIFE', 'EVENT'
    image_urls  TEXT[],
    location    GEOGRAPHY(Point, 4326) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_map_posts_location ON map_posts USING GIST(location);
```

---

## 🌐 API Routes

### API Gateway (`:8080`)

| Method | Path | Service | Auth | Description |
|--------|------|---------|------|-------------|
| GET | `/health` | — | ❌ | Health check |
| GET | `/docs` | — | ❌ | Swagger UI |
| GET/POST | `/v1/auth/*` | User | ❌ | Google OAuth routes |
| Any | `/v1/users` | User | ✅ | User CRUD |
| Any | `/v1/users/*` | User | ✅ | User details, search, block |
| Any | `/v1/me/*` | User | ✅ | Current user |
| Any | `/v1/conversations` | Chat | ✅ | Conversation CRUD |
| Any | `/v1/conversations/*` | Chat | ✅ | Messages, details |
| Any | `/v1/groups` | Chat | ✅ | Group CRUD |
| Any | `/v1/groups/*` | Chat | ✅ | Group details |
| Any | `/v1/presence/*` | Chat | ✅ | User presence |
| Any | `/v1/chat/users` | Chat | ✅ | User cache |
| Any | `/v1/messages/*` | Chat | ✅ | Messages |
| Any | `/v1/swipe` | Match | ✅ | Swipe action |
| Any | `/v1/matches` | Match | ✅ | Match list |
| Any | `/v1/matches/*` | Match | ✅ | Match details |
| Any | `/v1/discover` | Match | ✅ | Discovery feed |
| Any | `/v1/location` | Location | ✅ | Location CRUD |
| Any | `/v1/location/*` | Location | ✅ | Nearby, WS |
| Any | `/v1/posts` | Location | ✅ | Map posts CRUD |
| Any | `/v1/posts/*` | Location | ✅ | Post details |
| WS | `/ws` | Chat | ✅ | WebSocket connection |

### Next.js Rewrites

| Source | Destination | Purpose |
|--------|-------------|---------|
| `/v1/:path*` | `http://localhost:8080/v1/:path*` | REST API proxy |
| `/ws` | `http://localhost:8080/ws` | WebSocket proxy |

> **Note:** Next.js rewrites only proxy HTTP — WebSocket upgrades go directly to `ws://localhost:8080/ws` via `NEXT_PUBLIC_WS_URL`.

---

## 🔐 Environment Variables

### API Gateway

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8080` | ❌ | HTTP port |
| `USER_SERVICE_URL` | `http://user-service:8081` | ❌ | User service address |
| `MATCH_SERVICE_URL` | `http://match-service:8082` | ❌ | Match service address |
| `CHAT_SERVICE_URL` | `http://chat-service:8083` | ❌ | Chat service address |
| `LOCATION_SERVICE_URL` | `http://location-service:8084` | ❌ | Location service address |
| `CLERK_JWT_ISSUER` | `""` | ❌ (prod) | Clerk JWT issuer URL |
| `CLERK_JWT_KEY` | `""` | ❌ (prod) | Clerk public key (RS256) or shared secret (HS256) |

> **Dev mode:** When `CLERK_JWT_KEY` is empty, the gateway uses `DevAuthMiddleware` which decodes JWTs without verification. Set it to `dev-jwt-secret-do-not-use-in-production` to validate HS256 dev tokens.

### User Service

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8081` | ❌ | HTTP port |
| `GRPC_PORT` | `50051` | ❌ | gRPC port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/users?sslmode=disable` | ❌ | PostgreSQL connection |
| `GOOGLE_CLIENT_ID` | `""` | ❌ (OAuth) | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | `""` | ❌ (OAuth) | Google OAuth secret |
| `GOOGLE_REDIRECT_URL` | `http://localhost:8080/v1/auth/google/callback` | ❌ | OAuth callback URL |
| `JWT_SECRET` | `change-me-in-production` | ❌ | JWT signing secret |
| `KAFKA_BROKERS` | `kafka:9092` | ❌ | Kafka broker addresses |
| `S3_BUCKET` | `""` | ❌ | S3 bucket for photos |
| `S3_REGION` | `""` | ❌ | S3 region |

### Chat Service

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8083` | ❌ | HTTP port |
| `SCYLLA_URL` | `scylladb://chat-db:9042/app_chat` | ❌ | ScyllaDB connection |
| `REDIS_URL` | `redis:6379` | ❌ | Redis address |
| `KAFKA_BROKERS` | `kafka:9092` | ❌ | Kafka broker addresses |
| `USER_SERVICE_URL` | `http://user-service:8081` | ❌ | User service address |
| `USER_SERVICE_GRPC` | `user-service:50051` | ❌ | gRPC address |

### Match Service

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8082` | ❌ | HTTP port |
| `DATABASE_URL` | `postgres://postgres:postgres@match-db:5432/matches?sslmode=disable` | ❌ | PostgreSQL connection |
| `USER_SERVICE_URL` | `http://user-service:8081` | ❌ | User service address |

### Location Service

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8084` | ❌ | HTTP port |
| `REDIS_URL` | `redis:6379` | ❌ | Redis address |
| `DATABASE_URL` | `postgres://postgres:postgres@location-db:5432/locations?sslmode=disable` | ❌ | PostGIS connection |

### Web Frontend

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `NEXT_PUBLIC_API_URL` | `""` (same-origin) | ❌ | API base URL |
| `NEXT_PUBLIC_WS_URL` | `ws://localhost:8080` | ❌ | WebSocket URL |

---

## 🛠 Local Development Setup

### Prerequisites

- **Go 1.25+** — for backend services
- **Node.js 20+** — for Next.js frontend
- **Docker & Docker Compose** — for infrastructure (PostgreSQL, ScyllaDB, Redis, Kafka)
- **Make** — for build/task automation

### Quick Start

```bash
# 1. Clone and enter project
cd microservices-go-starter

# 2. Initialize Go workspace (one time)
go work init
go work use ./api-gateway ./user-service ./match-service ./chat-service ./location-service ./shared

# 3. Start all infrastructure + services
docker compose up -d

# 4. Run database migrations
make migrate-up

# 5. Start the frontend (separate terminal)
cd web && npm install && npm run dev

# 6. Open browser
open http://localhost:3000
```

> **First-time setup:** After running `docker compose up -d`, wait 30-60 seconds for ScyllaDB to initialize. The chat-service has `restart: on-failure:5` to handle this.

### Available Make Commands

```bash
make up                  # Start all services via Docker Compose
make down                # Stop all services
make logs                # Tail all logs
make build-all           # Build all Go services
make run-user            # Run user-service locally (outside docker)
make run-chat            # Run chat-service locally
make test-all            # Run all tests
make migrate-up          # Apply all migrations
make migrate-user        # Apply user-service migrations only
make migrate-chat        # Apply ScyllaDB chat migrations
make tidy-all            # Run go mod tidy on all modules
make vet-all             # Run go vet on all modules
make compile-proto       # Compile gRPC protobuf
```

### Running Individual Services for Debugging

```bash
# Run infrastructure only
docker compose up -d user-db match-db chat-db redis

# Run a single service locally with hot-reload
cd user-service && go run ./cmd/api
cd chat-service && go run ./cmd/api
cd api-gateway && go run ./cmd/api
```

### Testing

```bash
# Run all Go tests
make test-all

# Run the chat flow integration test
chmod +x scripts/test-chat-flow.sh
./scripts/test-chat-flow.sh

# Run with verbose output
VERBOSE=true ./scripts/test-chat-flow.sh
```

### Test Script (`scripts/test-chat-flow.sh`)

The test script automates:
1. Health check against API Gateway
2. Creates two test users (Alice & Bob)
3. Creates a conversation between them
4. Sends messages back and forth
5. Verifies message persistence in ScyllaDB
6. Prints `wscat` commands for manual WebSocket testing

---

## 🚢 Deployment Options

### Option 1: Docker Compose (Single Server / MVP)

**Best for:** Demo, small-scale, 0–1k users

```bash
# Build and start all services
docker compose up -d --build

# Check status
docker compose ps

# View logs
docker compose logs -f
```

**Dockerfile patterns:**
- **api-gateway**: Standard `go build` inside container
- **user-service**: Multi-stage build copying `shared/` from parent
- **chat-service**: Multi-stage build copying `shared/` from parent
- **match-service**: Standard `go build`
- **location-service**: Standard `go build`

> **Note:** `user-service` and `chat-service` use `context: .` (project root) and `dockerfile: user-service/Dockerfile` because they depend on `shared/` module.

### Option 2: Kubernetes (Production)

**Best for:** 1k–500k users, auto-scaling, self-healing

#### Development K8s (Tilt)

```bash
# Install Tilt: https://tilt.dev/
tilt up
```

Tilt provides hot-reload development on K8s:
- Auto-compiles Go services on file changes
- Streams logs to terminal
- Port-forwards services to localhost

#### Production K8s Manifests

Located in `infra/production/k8s/`:

```bash
# Apply deployment
kubectl apply -f infra/production/k8s/app-config.yaml
kubectl apply -f infra/production/k8s/api-gateway-deployment.yaml
# ... add other services

# Create secrets
kubectl create secret generic user-service-secrets \
  --from-literal=DATABASE_URL=postgres://... \
  --from-literal=GOOGLE_CLIENT_ID=...
```

**Resources per service (production):**

| Service | CPU Request | Memory Request | CPU Limit | Memory Limit | Replicas |
|---------|-------------|----------------|-----------|--------------|----------|
| API Gateway | 125m | 128Mi | 125m | 128Mi | 2-3 |
| User Service | 250m | 256Mi | 500m | 512Mi | 2-3 |
| Chat Service | 500m | 512Mi | 1000m | 1Gi | 3-5 |
| Location Service | 250m | 256Mi | 500m | 512Mi | 2-3 |
| Match Service | 250m | 256Mi | 500m | 512Mi | 2-3 |
| Web Frontend | 100m | 256Mi | 500m | 512Mi | 2-3 |

### Option 3: Managed Cloud Services

**Best for:** Production scale, managed infrastructure

| Component | Recommended Service | Cost |
|-----------|-------------------|------|
| **API Gateway** | Cloud Run / AWS ECS Fargate | Pay-per-request |
| **User Service** | Cloud Run / ECS Fargate | Pay-per-request |
| **Chat Service** | Cloud Run (+ CPU always on for WS) | Pay-per-request |
| **Location Service** | Cloud Run (+ CPU always on for WS) | Pay-per-request |
| **Match Service** | Cloud Run / ECS Fargate | Pay-per-request |
| **Frontend** | Vercel (Next.js) | Free tier available |
| **PostgreSQL** | Cloud SQL / RDS | ~$15-50/mo |
| **ScyllaDB** | ScyllaDB Cloud / Astra DB | ~$50-200/mo |
| **Redis** | Memorystore / ElastiCache | ~$15-30/mo |
| **PostGIS** | RDS with PostGIS extension | ~$15-50/mo |
| **Kafka** | Confluent Cloud / MSK | ~$30-100/mo |
| **Object Storage** | S3 / GCS | ~$0.023/GB/mo |
| **Load Balancer** | Cloud LB / ALB | ~$20/mo |
| **Container Registry** | GCR / ECR / Docker Hub | Free tier |

#### Cloud Run Deployment (serverless)

```bash
# Build and push
gcloud builds submit --tag gcr.io/$PROJECT_ID/api-gateway ./api-gateway
gcloud builds submit --tag gcr.io/$PROJECT_ID/chat-service --context=. --dockerfile=chat-service/Dockerfile .

# Deploy
gcloud run deploy api-gateway \
  --image gcr.io/$PROJECT_ID/api-gateway \
  --port 8080 \
  --env-vars-file .env.prod.yaml

gcloud run deploy chat-service \
  --image gcr.io/$PROJECT_ID/chat-service \
  --port 8083 \
  --cpu-always \
  --env-vars-file .env.chat.yaml

gcloud run deploy web \
  --image gcr.io/$PROJECT_ID/web \
  --port 3000
```

---

## 🔄 CI/CD Pipeline

### Recommended Pipeline (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Run Go tests
        run: make test-all
      - name: Run Go vet
        run: make vet-all
      - name: Lint frontend
        run: cd web && npm ci && npm run lint

  build-and-push:
    needs: test
    strategy:
      matrix:
        service: [api-gateway, user-service, match-service, chat-service, location-service]
    steps:
      - uses: actions/checkout@v4
      - name: Build & push Docker image
        run: |
          docker build -t gcr.io/${{ secrets.PROJECT_ID }}/${{ matrix.service }}:latest \
            -f ${{ matrix.service }}/Dockerfile .
          docker push gcr.io/${{ secrets.PROJECT_ID }}/${{ matrix.service }}:latest

  deploy:
    needs: build-and-push
    steps:
      - name: Deploy to Cloud Run
        run: |
          for service in api-gateway user-service match-service chat-service location-service web; do
            gcloud run deploy $service \
              --image gcr.io/${{ secrets.PROJECT_ID }}/$service:latest \
              --region us-central1
          done
```

### Build Order (Dependency-Aware)

```
1. shared/          ← No dependencies (library only)
2. api-gateway/     ← No service dependencies
3. user-service/    ← Depends on shared/
4. match-service/   ← Depends on user-service (HTTP)
5. chat-service/    ← Depends on shared/, user-service (gRPC)
6. location-service/← No service dependencies
7. web/             ← Depends on API Gateway
```

---

## 📊 Monitoring & Observability

### Logging (Already Implemented)

All services use structured JSON logging via `log/slog`:

```json
{
  "time": "2026-07-13T10:30:00Z",
  "level": "INFO",
  "msg": "client registered",
  "component": "ws_hub",
  "user_id": "abc-123"
}
```

### Recommended Stack

| Layer | Tool | Purpose |
|-------|------|---------|
| **Metrics** | Prometheus + Grafana | Request latency, DB connections, WS clients |
| **Logging** | Loki / CloudWatch | Centralized log aggregation |
| **Tracing** | OpenTelemetry + Jaeger | Distributed tracing across services |
| **Alerts** | PagerDuty / Slack | Error rate > 1%, DB CPU > 80%, Kafka lag |
| **Uptime** | UptimeRobot / Pingdom | External health monitoring |

### Key Metrics to Track

```
# Per service
http_requests_total{service="chat-service", status="2xx"}
http_request_duration_seconds{service="chat-service", quantile="0.99"}
ws_connections_active{service="chat-service"}
ws_messages_fanned_out_total{service="chat-service"}

# Infrastructure
pg_connections_active{db="users"}
redis_memory_usage_bytes
kafka_consumer_lag{consumer_group="chat-service"}
scylla_latency_percentile{operation="write"}
```

### Health Endpoints

Each service should expose:
```
GET /health → 200 OK
GET /ready → 200 OK (dependencies ready)
GET /metrics → Prometheus metrics
```

---

## 🔒 Security Checklist

### Authentication

- [x] JWT-based auth (HS256 dev / RS256 production)
- [x] Google OAuth integration (user-service)
- [x] DevAuthMiddleware for local development
- [x] WebSocket auth via `?token=` query parameter
- [ ] mTLS between services (Linkerd / Istio in K8s)

### Authorization

- [x] `X-User-ID` header injected by API Gateway
- [x] Block user feature prevents cross-user messaging
- [ ] Role-based access control (admin endpoints)

### Data Protection

- [ ] PII Encryption at rest (S3 SSE-KMS for photos)
- [ ] TLS 1.3 for all in-transit data
- [ ] Database credentials via secrets (not hardcoded)
- [ ] ScyllaDB encryption at rest

### Rate Limiting

- [x] Rate limiter middleware in API Gateway (`internal/middleware/rate_limit.go`)
- [ ] 100 req/min per IP
- [ ] 1000 req/min per authenticated user
- [ ] WebSocket connection limits per user

### Input Validation

- [x] `go-playground/validator/v10` in all services
- [x] WebSocket message size limit: 4096 bytes
- [x] Request body size limits
- [ ] SQL injection prevention (parameterized queries — already used)

### WebSocket Security

- [x] Write deadlines prevent slow-client attacks
- [x] Read deadlines detect dead connections
- [x] Ping/pong heartbeats every 30s
- [x] Backpressure: drop oldest message when send buffer full
- [x] Max message size: 4096 bytes

### GDPR / Data Deletion

- [ ] Right to be forgotten — cascade delete across all services
- [ ] Data export endpoint
- [ ] Session invalidation on password change

---

## 📈 Scaling Strategy

### Stage 1: MVP (0–1k users)

```
Architecture: Single machine with Docker Compose
Databases:   All containers on same host
Frontend:    Vercel (serverless)
Cost:        ~$20-50/mo (VPS + databases)

Limitations:
- All services compete for same resources
- No horizontal scaling
- Single point of failure
```

### Stage 2: Growth (1k–50k users)

```
Architecture: Docker Compose → Managed services
Databases:    Cloud SQL / RDS (separate instances)
Frontend:     Vercel Pro
Chat:         Chat Service with Redis PubSub cross-instance
Cost:         ~$100-300/mo

Changes:
- Split databases to managed services
- Add read replicas for PostgreSQL
- Increase Chat Service replicas (Redis PubSub handles cross-instance)
- Add CDN for static assets
```

### Stage 3: Scale (50k–500k users)

```
Architecture: Kubernetes (GKE / EKS)
Databases:    Dedicated instances per service
Chat:         Multiple replicas + Redis PubSub + Kafka
Cost:         ~$500-2000/mo

Changes:
- K8s with auto-scaling (HPA based on CPU/memory/custom metrics)
- Service mesh (Linkerd or Istio) for mTLS + traffic management
- Database read replicas + connection pooling (PgBouncer)
- ScyllaDB multi-datacenter replication
- Redis Cluster (not standalone)
- Kafka partitioning by conversation_id
```

### Stage 4: Global (500k+ users)

```
Architecture: Multi-region K8s
Databases:    Multi-region with active-active
Chat:         Regional WebSocket gateways
Cost:         $2000+/mo

Changes:
- Multi-region deployment (us-east1, europe-west1, asia-east1)
- Global load balancer (Cloud CDN / CloudFront)
- ScyllaDB multi-datacenter with per-region replication
- Kafka MirrorMaker for cross-region events
- CDN for image assets + map tiles
- Read-write splitting at application layer
```

### Bottleneck Analysis

| Component | Bottleneck | Mitigation |
|-----------|-----------|------------|
| **Chat Service** | WebSocket connections per instance | Horizontal scaling + Redis PubSub |
| **ScyllaDB** | Write throughput — already optimized | Add nodes, increase RF |
| **PostgreSQL (User)** | Read queries | Read replicas + caching |
| **PostgreSQL (Match)** | Write contention | Shard by user_id hash |
| **Redis** | Memory (GEO data) | Redis Cluster |
| **API Gateway** | Request throughput | Stateless — scale horizontally |
| **Kafka** | Consumer lag | Increase partitions + consumer groups |

### Chat Service Scaling Notes

The Chat Service is already optimized for scale:

1. **Sharded room hub** — 32 shards reduce mutex contention
2. **Single-marshal fan-out** — marshal once before broadcast
3. **Backpressure** — drop oldest message when buffer full
4. **Redis PubSub** — broadcast across instances
5. **Kafka** — optional durable event log
6. **Graceful shutdown** — drain connections on SIGTERM

---

## 🧪 Go Workspace Structure

```
go.work
├── ./api-gateway       → go.mod: module github.com/ride-sharing/api-gateway
├── ./user-service      → go.mod: module github.com/ride-sharing/user-service
├── ./match-service     → go.mod: module github.com/ride-sharing/match-service
├── ./chat-service      → go.mod: module github.com/ride-sharing/chat-service
├── ./location-service  → go.mod: module github.com/ride-sharing/location-service
└── ./shared            → go.mod: module github.com/ride-sharing/shared
```

### Dependency Graph

```
shared (errors, logger, middleware, validators)
    ↑
    ├── api-gateway     (shared + echo + jwt)
    ├── user-service    (shared + pgx + kafka + grpc + oauth)
    ├── match-service   (shared + pgx)
    ├── chat-service    (shared + gocql + gorilla/ws + redis + kafka + grpc)
    └── location-service(shared + redis + pgx)
```

---

## 📝 Migrations

### Applying Migrations

```bash
# PostgreSQL migrations (user-service, match-service, location-service)
make migrate-user     # Users + profiles + blocks
make migrate-match    # Matches
make migrate-location # Map posts (PostGIS)

# ScyllaDB migration (chat-service)
make migrate-chat     # Messages + conversations + groups

# Rollback (PostgreSQL only)
make migrate-user-down
make migrate-match-down
make migrate-location-down
```

### Migration Files

| File | Service | Description |
|------|---------|-------------|
| `user-service/migrations/001_users.up.sql` | User | Creates app_users table |
| `user-service/migrations/002_blocked_users.up.sql` | User | Creates blocked_users table |
| `match-service/migrations/001_matches.up.sql` | Match | Creates matches table |
| `chat-service/migrations/001_chat.up.sql` | Chat | ScyllaDB keyspace + messages + conversations |
| `chat-service/migrations/002_groups.up.sql` | Chat | ScyllaDB groups table |
| `location-service/migrations/001_map_posts.up.sql` | Location | PostGIS map_posts table |

---

## ⚡ Performance Optimizations (Chat Service)

### Already Implemented

| Optimization | File | Description |
|-------------|------|-------------|
| **Sharded room hub** | `hub.go` | 32 shards via FNV hash, reduces lock contention |
| **Single-marshal fan-out** | `hub.go:SendToRoom` | Marshal once, broadcast pre-encoded bytes |
| **Pre-allocated maps** | `hub.go:JoinRoom` | `make(map[*Client]struct{}, 64)` |
| **sync.Pool-ready** | — | Use `sync.Pool` for frequently allocated structs |
| **Value types for small structs** | `domain/chat.go` | Prefer value types for small DTOs |
| **Single writer rule** | `client.go` | Exactly one goroutine calls WriteMessage |
| **Channel backpressure** | `client.go` | Drop oldest when buffer full (256 cap) |
| **Heartbeat** | `client.go` | Ping every 30s, pong timeout 35s |
| **Strict write deadlines** | `client.go` | 10s `SetWriteDeadline` before every write |
| **Graceful shutdown** | `hub.go:Shutdown` | Drain connections on SIGTERM |
| **Context-aware Kafka** | `kafka/consumer.go` | Context-aware graceful shutdown |

### Room for Improvement

| Optimization | Effort | Impact |
|-------------|--------|--------|
| Add `sync.Pool` for `WSEnvelope` marshaling | Low | Medium |
| Connection read/write timeouts configurable | Low | Medium |
| Add compression for large messages (>4KB) | Low | Low |
| Implement rate limiting per user on WS | Medium | High |
| Add prometheus metrics for WS connections | Low | High |
| Implement client-side message batching | Medium | Medium |

---

## 🧭 Frontend Architecture Notes

### Key Libraries

| Library | Version | Purpose |
|---------|---------|---------|
| Next.js | 15.1.5 | App Router, SSR/SSG |
| React | 19.0.0 | UI library |
| maplibre-gl | 5.24.0 | Open-source map rendering |
| react-map-gl | 8.1.1 | React wrapper for maplibre |
| Zustand | 5.0.14 | Client-side state management |
| Framer Motion | 12.42.2 | Animations |
| Vaul | 1.1.2 | Bottom sheet drawer |
| shadcn/ui | latest | Radix-based components |
| Tailwind CSS | 3.4.1 | Utility CSS |
| clsx + tailwind-merge | — | Class merging utility |

### Map Configuration

```typescript
// OpenFreeMap tiles (free, no API key needed)
export const MAP_STYLE = 'https://tiles.openfreemap.org/styles/liberty';

// Seed posts for demo — San Francisco downtown area
export const SEED_POSTS: MapPost[] = [
  // 7 restaurants, 5 social spots, 6 events
  // Spread across SF downtown with realistic coordinates
];
```

### WebSocket Connections

The frontend establishes two WebSocket connections:

1. **Chat WebSocket** — via `useChatHandshake` hook
   - Persistent connection for presence + room readiness
   - Auto-reconnect with exponential backoff (1s → 15s)
   
2. **Location WebSocket** — via `useLocationWebSocket` hook
   - Receives real-time location updates
   - Same auto-reconnect pattern

### Auth Token Flow

```
1. Login → token stored in localStorage
2. API calls → Authorization: Bearer <token>
3. WebSocket → ws://host/ws?user_id=xxx&token=xxx
4. API Gateway → validates token → sets X-User-ID header → proxies to service
```

### Seed Data Strategy

When the backend has no data (e.g., first run), the frontend provides:
- **Seed map posts** — 18 posts across SF (restaurants, social spots, events)
- **Seed comments** — rich thread discussions per post
- **Simulated users** — Alice, Bob, Carol moving around the map
- **Simulated movement** — user walking route around SF streets

---

## 🐳 Docker Compose Services Map

| Service | Image | Port | Depends On | Health Check |
|---------|-------|------|------------|--------------|
| `api-gateway` | Built from `api-gateway/Dockerfile` | 8080 | user-service, match-service, chat-service, location-service | — |
| `user-service` | Built from `user-service/Dockerfile` (root context) | 8081 | user-db (healthy) | — |
| `match-service` | Built from `match-service/Dockerfile` | 8082 | match-db (healthy) | — |
| `chat-service` | Built from `chat-service/Dockerfile` (root context) | 8083 | chat-db (healthy), redis | `restart: on-failure:5` |
| `location-service` | Built from `location-service/Dockerfile` | 8084 | redis, location-db | — |
| `user-db` | postgres:16-alpine | 5432 | — | `pg_isready` |
| `match-db` | postgres:16-alpine | 5433 | — | `pg_isready` |
| `chat-db` | scylladb/scylla:5.4 | 9042 | — | `cqlsh -e 'DESCRIBE KEYSPACES'` |
| `location-db` | postgis/postgis:16-3.4 | 5434 | — | `pg_isready` |
| `redis` | redis:7-alpine | 6379 | — | — |
| `zookeeper` | confluentinc/cp-zookeeper:7.7.1 | 2181 | — | `echo srvr \| nc.localhost 2181` |
| `kafka` | confluentinc/cp-kafka:7.7.1 | 9092 | zookeeper (healthy) | `kafka-topics --list` |

---

## 🏁 Getting Started Checklist

- [ ] Install Go 1.25+, Docker, Node.js 20+
- [ ] Clone repo: `git clone <repo-url>`
- [ ] Start infra: `docker compose up -d`
- [ ] Apply migrations: `make migrate-up`
- [ ] Install frontend deps: `cd web && npm install`
- [ ] Start frontend: `cd web && npm run dev`
- [ ] Open browser: `http://localhost:3000`
- [ ] Register a user → auto-login with dev JWT
- [ ] Open second browser → register another user
- [ ] Start a conversation → real-time chat works
- [ ] Verify map shows SF with seed posts
- [ ] Test blocking feature in user profile

### Demo Accounts (Dev Mode)

Users are created on-the-fly via simulated login. No pre-seeded accounts needed.

---

## 🧠 Key Architectural Decisions

### Why ScyllaDB for Chat?
- **Write-heavy workload** — messages are append-only, time-series data
- **Low latency** — consistent sub-5ms writes at high throughput
- **Linear scalability** — add nodes, no sharding logic needed
- **Cassandra-compatible** — familiar CQL, wide ecosystem support

### Why Separate PostgreSQL Instances?
- **Database-per-service** — no shared tables between services
- **Independent scaling** — user DB can be larger than match DB
- **Independent schema migrations** — each service manages its own
- **Fault isolation** — one DB crash doesn't cascade

### Why Redis for Location?
- **GEO commands** — native `GEOADD`, `GEORADIUS` for nearby queries
- **Sub-millisecond** — in-memory speed for real-time location
- **PubSub** — broadcast location updates across Chat Service instances

### Why Sharded Hub for WebSocket?
The Chat Service's hub uses 32 shards (FNV hash) instead of a global mutex:
- **Before**: One mutex for all rooms → contention at scale
- **After**: 32 independent mutexes → ~32x throughput before contention

### Why Next.js Rewrites Instead of CORS?
- **Simpler** — no CORS headers, preflight requests, or origin configuration
- **Same-origin** — cookies and auth tokens flow naturally
- **Production-ready** — reverse proxy (Nginx/Caddy/Cloud LB) at deploy time

---

## 📦 Repository Structure

```
microservices-go-starter/
├── deploy__project__archi__plan.md    ← This file
├── ReadMePlanv1.md                    ← Previous architecture doc
├── ChatRoomInstruction.md             ← Chat service optimization guide
├── go.work                            ← Go workspace
├── go.mod / go.sum                    ← Root module
├── Makefile                           ← Build automation
├── Tiltfile                           ← Tilt dev orchestration
├── docker-compose.yml                 ← Local development
│
├── api-gateway/                       ← API Gateway (Echo)
├── user-service/                      ← User management (PostgreSQL)
├── match-service/                     ← Matching logic (PostgreSQL)
├── chat-service/                      ← Real-time chat (ScyllaDB + Redis)
├── location-service/                  ← Location + posts (Redis + PostGIS)
├── shared/                            ← Shared Go libraries
├── web/                               ← Next.js frontend
│
├── proto/                             ← Protobuf definitions
├── scripts/                           ← Test & utility scripts
├── infra/                             ← Infrastructure configs
│   ├── development/                   ← Dev K8s + Docker configs
│   └── production/                    ← Production K8s configs
├── docs/                              ← Architecture docs & specs
└── api-gateway/docs/specs/            ← OpenAPI specs per service
```

---

> **Last Updated:** July 2026
> **Go Version:** 1.25
> **Node Version:** 20+
>
> *For questions or issues, refer to the individual service README files or open an issue in the repository.*
