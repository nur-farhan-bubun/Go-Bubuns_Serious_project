# Chat Message Flow

> Complete functional flow map for conversation messaging — from browser to backend and back.

## Architecture Overview

```
┌──────────┐     HTTP/WS      ┌──────────────┐    proxy    ┌───────────────┐
│  Browser  │ ──────────────▶ │ API Gateway   │ ──────────▶ │ Chat Service  │
│ (Next.js) │ ◀────────────── │ (port 8080)   │ ◀────────── │ (port 8083)   │
└──────────┘                 └──────────────┘              └───────┬───────┘
                                                                   │
                                            ┌──────────────────────┼──────────────────────┐
                                            │                      │                      │
                                       ┌────▼────┐          ┌─────▼─────┐          ┌─────▼─────┐
                                       │ ScyllaDB│          │   Redis   │          │   Kafka   │
                                       │  chat   │          │ presence  │          │  fan-out  │
                                       └─────────┘          └───────────┘          └───────────┘
```

## Authentication Flow (Pre-Chat)

```
Browser                          API Gateway                    User Service
   │                                  │                              │
   │  1. Login (Google OAuth)         │                              │
   │ ────────────────────────────────▶│                              │
   │                                  │  2. POST /v1/auth/google     │
   │                                  │ ────────────────────────────▶│
   │                                  │                              │
   │                                  │  3. JWT token                │
   │                                  │ ◀────────────────────────────│
   │  4. Store JWT (localStorage)     │                              │
   │ ◀────────────────────────────────│                              │
   │                                  │                              │
   │  5. REST call with Authorization: Bearer <JWT>                  │
   │ ────────────────────────────────▶│                              │
   │                                  │  6. Verify JWT (HS256)       │
   │                                  │  Set X-User-ID header        │
   │                                  │  Proxy to chat-service       │
   │                                  │ ────────────────────────────▶│
```

**Key files:**
- `web/src/lib/auth.ts` — JWT generation, storage, `loginSimulated()`
- `api-gateway/internal/middleware/auth.go` — JWT verification, sets `X-User-ID`
- `api-gateway/internal/handler/gateway.go` — Route proxies

---

## Opening the Chat Panel

```
Browser (ChatStore.openChat())
   │
   ├──▶ GET /v1/conversations (with Bearer token)
   │       │
   │       ├──▶ API Gateway proxies to chat-service:8083
   │       │       │
   │       │       ├──▶ openapi_handler.GetConversations()
   │       │       │       │
   │       │       │       ├──▶ service.GetConversations(userID)
   │       │       │       │       │
   │       │       │       │       └──▶ repo.GetConversationsByUserID(userID)
   │       │       │       │               │
   │       │       │       │               ├──▶ Query conversations WHERE user1_id = ?
   │       │       │       │               ├──▶ Query conversations WHERE user2_id = ?
   │       │       │       │               └──▶ Query conversation_members WHERE user_id = ?  ← group convs
   │       │       │       │
   │       │       │       └──▶ Returns []api.Conversation (JSON)
   │       │       │
   │       │       ◀── API response
   │       │
   │   ◀── Conversations list
   │
   ├── (If no convs exist) POST /v1/conversations { user_id: "user-alice" }
   │
   └──▶ connectToRoom(currentUser.id, conversationId)
           │
           └──▶ connectChatWS(userID, roomID, onMessage, onStatusChange)
                   │
                   └──▶ new WebSocket("ws://localhost:8080/ws?user_id=xxx&room_id=xxx&token=xxx")
                           │
                           ├──▶ Upgrade to WebSocket
                           ├──▶ hub.Register(client)          → presence: online
                           └──▶ hub.JoinRoom(client, roomID)  → added to shard
```

**Key files:**
- `web/src/components/chat/ChatStore.ts` — `openChat()`, `setConversation()`, `connectToRoom()`
- `web/src/lib/chat.ts` — `fetchConversations()`, `connectChatWS()`
- `chat-service/internal/handler/openapi_handler.go` — `GetConversations()`, `HandleWebSocket()`
- `chat-service/internal/websocket/hub.go` — `Register()`, `JoinRoom()`
- `chat-service/internal/websocket/client.go` — `ReadPump()`, `WritePump()`

---

## Sending a Message (REST path)

```
Browser (ChatFeed: user clicks Send)
   │
   ├──▶ ChatStore.sendMessage(content)
   │       │
   │       ├──▶ Optimistically add to local messages[]  (instant UI)
   │       │
   │       ├──▶ sendMessageAPI(conversationId, content)  (REST, fire-and-forget)
   │       │       │
   │       │       └──▶ POST /v1/conversations/{id}/messages  { content: "..." }
   │       │               │
   │       │               └──▶ API Gateway → chat-service
   │       │                       │
   │       │                       ├──▶ openapi_handler.SendMessage()
   │       │                       │       │
   │       │                       │       ├──▶ service.GetConversation()     → verify participant
   │       │                       │       ├──▶ service.SendMessage(msg)       → ScyllaDB persist
   │       │                       │       └──▶ broadcastNewMessage(id, msg)   → WebSocket + Kafka
   │       │                       │               │
   │       │                       │               ├──▶ Local: broadcastToRoom(roomID, chatMsg)
   │       │                       │               │       │
   │       │                       │               │       └──▶ hub.SendToRoom(roomID, payload)
   │       │                       │               │               │
   │       │                       │               │               └──▶ client.SendBytes(data)
   │       │                       │               │                       for each client in shard
   │       │                       │               │
   │       │                       │               └──▶ Kafka: producer.Publish(chatMsg)
   │       │                       │                       │
   │       │                       │                       └──▶ kafka.WriteMessages (async)
   │       │                       │
   │       │                       └──▶ Response: 201 Created { id, ... }
   │       │
   │       └──▶ wsConnection.send(...)  (WebSocket, real-time)
   │               │
   │               └──▶ WS { type: "chat_message", room_id, data: { content } }
   │                       │
   │                       └──▶ chat-service WebSocket handler
   │                               │
   │                               └──▶ handleIncomingChat()
   │                                       │
   │                                       ├──▶ service.SendMessage(msg)    → ScyllaDB persist
   │                                       ├──▶ broadcastToRoom(roomID, msg) → local clients
   │                                       └──▶ Kafka publish               → cross-instance
   │
   └──▶ Update conversations[] lastMessage, lastTime
```

**Key files:**
- `web/src/components/chat/ChatFeed.tsx` — input form, `handleSend()`
- `web/src/components/chat/ChatStore.ts` — `sendMessage()`
- `web/src/lib/chat.ts` — `sendMessageAPI()`
- `chat-service/internal/handler/openapi_handler.go` — `SendMessage()`, `handleIncomingChat()`, `broadcastNewMessage()`

---

## Receiving a Message (WebSocket delivery)

```
chat-service broadcastToRoom(roomID, chatMsg)
   │
   ├──▶ WSEnvelope { type: "chat_message", room_id, data: { message_id, ... } }
   │
   ├──▶ json.Marshal → payload ([]byte, marshalled once)
   │
   └──▶ hub.SendToRoom(roomID, payload)
           │
           ├──▶ hashRoomID(roomID) → shard index
           ├──▶ shard.mu.RLock → get clients set
           └──▶ for each client in shard.rooms[roomID]:
                    │
                    └──▶ client.SendBytes(data)
                            │
                            └──▶ select { case client.Send <- data: default: drop oldest }
                                    │
                                    │ (WritePump goroutine picks up)
                                    │
                                    └──▶ conn.WriteMessage(TextMessage, data)
                                            │
                                            └──▶ WebSocket frame → Browser
                                                    │
                                                    └──▶ ws.onmessage(event)
                                                            │
                                                            └──▶ JSON.parse → ChatWSEnvelope
                                                                    │
                                                                    ├──▶ If type === "chat_message":
                                                                    │       │
                                                                    │       ├──▶ Deduplicate (wsMessageIds)
                                                                    │       ├──▶ Skip if from self (already optimistically added)
                                                                    │       ├──▶ Skip if stale room
                                                                    │       └──▶ Add to messages[]
                                                                    │
                                                                    └──▶ If type === "presence":
                                                                            │
                                                                            └──▶ applyPresenceToStore(userId, status)
```

---

## Kafka Cross-Instance Fan-Out

```
chat-service Instance A                        Kafka                      chat-service Instance B
      │                                         │                               │
      ├── producer.Publish(msg) ───────────────▶│                               │
      │   (async, non-blocking)                  │                               │
      │                                         │                               │
      │                                         ├──▶ consumer.ReadMessage() ────▶│
      │                                         │                               │
      │                                         │                               ├──▶ hub.SendToRoom(roomID, payload)
      │                                         │                               │       │
      │                                         │                               │       └──▶ local clients in room
```

**Details:**
- Producer: `Async: true`, `RequiredAcks: RequireNone` — fires and forgets
- Consumer: Unique consumer group per instance (UUID) → every instance receives every message (fan-out)
- Presence events: broadcast globally (`hub.Broadcast`)
- Chat messages: routed to specific room (`hub.SendToRoom`)

**Key files:**
- `chat-service/internal/kafka/producer.go` — async Kafka writer
- `chat-service/internal/kafka/consumer.go` — reads from Kafka, fans out via Hub
- `chat-service/internal/kafka/message.go` — Kafka message structure

---

## WebSocket Reconnection & Message Sync

```
Browser disconnects (network blip, deploy, etc.)
   │
   ├──▶ ws.onclose → status: "disconnected"
   ├──▶ Reconnect with exponential backoff (1s → 1.5s → 2.25s → ... → 15s max)
   │
   └──▶ ws.onopen (reconnected)
           │
           ├──▶ status: "connected"
           ├──▶ hasConnectedOnce was true → trigger syncMissedOnReconnect()
           │       │
           │       └──▶ GET /v1/conversations/{id}/messages?limit=50
           │               │
           │               └──▶ Fetch latest messages from REST API
           │                       │
           │                       ├──▶ Merge with existing messages[]
           │                       ├──▶ Replace optimistic messages (prefix "msg-") with server IDs
           │                       └──▶ Add new message IDs to dedup set
           │
           └──▶ Real-time delivery resumes via WebSocket
```

**Key files:**
- `web/src/lib/chat.ts` — `connectChatWS()`, reconnection logic
- `web/src/components/chat/ChatStore.ts` — `connectToRoom()`, `syncMissedOnReconnect()`

---

## User Presence Flow

```
User A connects (WebSocket open)
   │
   ├──▶ hub.Register(client)
   │       │
   │       └──▶ hub.OnPresenceChange("user-A", "online")
   │               │
   │               ├──▶ hub.BroadcastPresence("user-A", "online")
   │               │       │
   │               │       └──▶ WSEnvelope { type: "presence", data: { user_id, status } }
   │               │               │
   │               │               └──▶ hub.Broadcast(payload) → ALL connected clients
   │               │
   │               └──▶ Kafka publish → cross-instance fan-out
   │
   ├──▶ redis: SetPresence(userID, "online")
   │
   └──▶ User A is now "online" → visible to all other users

User A disconnects (closes tab, navigates away)
   │
   ├──▶ hub.Unregister(client)
   │       │
   │       └──▶ hub.OnPresenceChange("user-A", "offline")
   │               │
   │               ├──▶ hub.BroadcastPresence("user-A", "offline")
   │               └──▶ Kafka publish
   │
   └──▶ User A is now "offline"
```

**Frontend handling:**
```
Frontend receives presence WS envelope
   │
   └──▶ applyPresenceToStore(userId, status)
           │
           ├──▶ Update registeredUsers[] (user directory)
           ├──▶ Update users[] (conversation members)
           └──▶ Update conversations[].members[].status + onlineCount
```

**Key files:**
- `chat-service/internal/websocket/hub.go` — `Register()`, `Unregister()`, `BroadcastPresence()`
- `chat-service/internal/handler/openapi_handler.go` — `PresenceBroadcast()`, `HandleWebSocket()`
- `chat-service/internal/repository/redis/presence_repo.go` — Redis presence storage
- `web/src/components/chat/ChatStore.ts` — `applyPresenceToStore()`, `updateUserPresence()`

---

## Data Flow: REST API Endpoints

| Action | Method | Path | Handler | Description |
|--------|--------|------|---------|-------------|
| List conversations | `GET` | `/v1/conversations` | `GetConversations` | All convs for current user |
| Create conversation | `POST` | `/v1/conversations` | `CreateConversation` | New direct chat |
| Create group | `POST` | `/v1/groups` | `CreateGroupConversation` | New group chat |
| Get messages | `GET` | `/v1/conversations/{id}/messages?limit=N` | `GetMessages` | Paginated messages |
| Send message | `POST` | `/v1/conversations/{id}/messages` | `SendMessage` | + WebSocket broadcast |
| Get presence | `GET` | `/v1/presence/{userID}` | `GetPresence` | Online/offline status |
| List users | `GET` | `/v1/chat/users` | `GetUsers` | Cached user directory |
| Get user | `GET` | `/v1/chat/users/{id}` | `GetUserByID` | Single cached user |
| WebSocket | `GET` | `/ws?user_id=xxx&room_id=xxx` | `HandleWebSocket` | Real-time messaging |

All proxied through the API Gateway at `localhost:8080/v1/...` and `localhost:8080/ws`.

---

## ScyllaDB Schema

### conversations (direct chats)
```
id         UUID      PRIMARY KEY
user1_id   TEXT
user2_id   TEXT
match_id   TEXT      (nullable)
created_at TIMESTAMP
```
Indexed by `user1_id` and `user2_id` for lookup.

### group_conversations
```
id         UUID      PRIMARY KEY
name       TEXT
created_at TIMESTAMP
```

### conversation_members
```
conversation_id UUID      PRIMARY KEY (conversation_id, user_id)
user_id         TEXT
joined_at       TIMESTAMP
```
Secondary index on `user_id` for "which groups is user in?" queries.

### messages
```
conversation_id UUID      PRIMARY KEY (conversation_id, id)
id              TIMEUUID  (sorted by time)
sender_id       TEXT
content         TEXT
created_at      TIMESTAMP
```

---

## Debugging Checklist

### Common Issues

| Symptom | Likely Cause | Check |
|---------|-------------|-------|
| Conversations not loading | `conversation_members` table missing | `docker compose exec chat-db cqlsh -e "USE app_chat; DESCRIBE TABLES"` |
| Messages not appearing in real-time | WebSocket not connected | Browser console: WebSocket status |
| Messages only visible after refresh | No WebSocket delivery, REST fallback only | Check `wsStatus` in ChatStore |
| "Failed to get conversations" error | ScyllaDB query failure | Check chat-service logs |
| Chat overlay not opening | Auth token missing/expired | `localStorage.getItem("chat_auth_token")` |
| One user sees messages, other doesn't | Different rooms | Check `room_id` in WS connection URL for both users |
| "Not a participant" error | User not in conv members list | Check `conversation_members` table |

### Quick Verification

```bash
# 1. Check chat-service logs for errors
docker compose logs chat-service

# 2. Check ScyllaDB tables exist
docker compose exec chat-db cqlsh -e "USE app_chat; DESCRIBE TABLES"

# 3. Check WebSocket connections
docker compose logs chat-service | grep "ws_hub"

# 4. Check presence events
docker compose logs chat-service | grep "presence"

# 5. Check messages in ScyllaDB
docker compose exec chat-db cqlsh -e "USE app_chat; SELECT * FROM messages LIMIT 5"
```
