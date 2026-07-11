package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/ride-sharing/chat-service/internal/domain"
	kafkainfra "github.com/ride-sharing/chat-service/internal/kafka"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

// ─── Pools ──────────────────────────────────────────────────────────────

// envelopePool reduces GC pressure by recycling WSEnvelope structs that are
// allocated on every incoming chat/typing message, used once for json.Marshal,
// then discarded.
var envelopePool = sync.Pool{
	New: func() any { return &domain.WSEnvelope{} },
}

// incomingMsgPool recycles WSIncomingMessage structs that back the hot path
// for parsing every WebSocket frame.
var incomingMsgPool = sync.Pool{
	New: func() any { return &domain.WSIncomingMessage{} },
}

// upgrader is configured with tuned buffer sizes and compression off by default.
var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: false,
	CheckOrigin:       func(r *http.Request) bool { return true },
}

// ─── Global Connection Network Map ──────────────────────────────────────
//
// Tracks users who connect to the WebSocket without a room_id (global
// connection state). This is used to deliver global events (e.g. presence
// updates, room_ready notifications) even when the user hasn't joined any
// specific conversation room.
//
// Map: userID → connectedAt (timestamp)
var (
	globalConnectionsMu sync.RWMutex
	globalConnections   = make(map[string]time.Time)
)

// trackGlobalConnection registers a user in the global connection network map.
// Called when a WebSocket upgrade succeeds without a room_id parameter.
func trackGlobalConnection(userID string) {
	globalConnectionsMu.Lock()
	globalConnections[userID] = time.Now()
	globalConnectionsMu.Unlock()
}

// removeGlobalConnection removes a user from the global connection network map.
// Should be called when a WebSocket client disconnects.
// Returns true if the user was tracked in the map.
func removeGlobalConnection(userID string) bool {
	globalConnectionsMu.Lock()
	_, ok := globalConnections[userID]
	delete(globalConnections, userID)
	globalConnectionsMu.Unlock()
	return ok
}

// ─── Presence Broadcast ─────────────────────────────────────────────────

// PresenceBroadcast sends a presence event to all local clients AND publishes
// to Kafka so other service instances receive it too.
// Also cleans up the global connection network map when a user goes offline.
func (h *OpenAPIHandler) PresenceBroadcast(userID string, status string) {
	// Clean up global connection map on disconnect
	if status == "offline" {
		if removed := removeGlobalConnection(userID); removed {
			h.log.Info("removed user from global connection map",
				slog.String("user_id", userID),
			)
		}
	}

	// 1. Locally broadcast to all connected clients on this instance
	h.hub.BroadcastPresence(userID, status)

	// 2. Publish to Kafka for cross-instance fan-out
	if h.producer != nil {
		rawData, err := json.Marshal(domain.WSPresenceUpdate{
			UserID: userID,
			Status: status,
		})
		if err != nil {
			h.log.Error("failed to marshal presence data", slog.String("error", err.Error()))
			return
		}
		if err := h.producer.Publish(context.Background(), &kafkainfra.Message{
			RoomID: "", // presence is global, not room-scoped
			Type:   domain.WSMsgTypePresence,
			Data:   rawData,
		}); err != nil {
			h.log.Warn("kafka presence publish failed", slog.String("error", err.Error()))
			// Local broadcast already happened, so this instance's clients are updated
		}
	}
}

// ─── HandleWebSocket ────────────────────────────────────────────────────

// HandleWebSocket upgrades the HTTP connection to WebSocket, creates a Client,
// registers it with the Hub, and starts the read/write pump goroutines.
//
// On upgrade success it:
//  1. Updates the user's presence to "online" in Redis (HSET presence:online).
//  2. Broadcasts a presence_update event via Kafka/memory pipeline.
//  3. If no room_id is provided, tracks the user in the global connection
//     network map so they can receive global events (room_ready, etc.).
func (h *OpenAPIHandler) HandleWebSocket(ctx echo.Context) error {
	userID := ctx.QueryParam("user_id")
	if userID == "" {
		return ctx.String(http.StatusBadRequest, "user_id query parameter is required")
	}

	roomID := ctx.QueryParam("room_id") // optional — client may join later

	conn, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		h.log.Error("websocket upgrade failed", slog.String("error", err.Error()))
		return err
	}

	client := ws.NewClient(h.hub, conn, userID, h.log)

	h.hub.Register(client)

	if roomID != "" {
		h.hub.JoinRoom(client, roomID)
	} else {
		// No room_id — track in global connection network map for global events
		trackGlobalConnection(userID)
		h.log.Info("tracked global connection (no room_id)",
			slog.String("user_id", userID),
		)
	}

	// Update presence to online in Redis (HSET presence:online userID timestamp)
	_ = h.svc.SetPresence(ctx.Request().Context(), &domain.Presence{
		UserID:   userID,
		Status:   "online",
		LastSeen: time.Now(),
	})

	// Spawn the single-writer goroutine.
	go client.WritePump()

	// Spawn the reader goroutine with a handler that broadcasts to the room.
	go client.ReadPump(func(c *ws.Client, message []byte) {
		h.handleIncomingMessage(c, message)
	})

	return nil
}

// ─── Incoming Message Router ────────────────────────────────────────────

// handleIncomingMessage processes a raw message from a WebSocket client.
func (h *OpenAPIHandler) handleIncomingMessage(client *ws.Client, raw []byte) {
	incoming := incomingMsgPool.Get().(*domain.WSIncomingMessage)
	if err := json.Unmarshal(raw, incoming); err != nil {
		h.log.Error("invalid message from client",
			slog.String("user_id", client.UserID),
			slog.String("error", err.Error()),
		)
		env := pooledErrorEnvelope("invalid message format")
		_ = client.SendJSON(env)
		env.Type = ""
		env.RoomID = ""
		env.Data = nil
		envelopePool.Put(env)
		*incoming = domain.WSIncomingMessage{}
		incomingMsgPool.Put(incoming)
		return
	}

	roomID := incoming.RoomID
	if roomID == "" {
		if rid, ok := client.RoomID.Load().(string); ok {
			roomID = rid
		}
	}
	if roomID == "" {
		*incoming = domain.WSIncomingMessage{}
		incomingMsgPool.Put(incoming)
		h.log.Warn("message dropped — no room assigned",
			slog.String("user_id", client.UserID),
		)
		return
	}

	switch incoming.Type {
	case domain.WSMsgTypeChat:
		h.handleIncomingChat(client, roomID, incoming.Data)
	case domain.WSMsgTypeTyping:
		h.handleIncomingTyping(client, roomID, incoming.Data)
	default:
		h.log.Warn("unknown message type",
			slog.String("user_id", client.UserID),
			slog.String("type", string(incoming.Type)),
		)
	}

	*incoming = domain.WSIncomingMessage{}
	incomingMsgPool.Put(incoming)
}

// ─── Chat Message Handler ───────────────────────────────────────────────

// handleIncomingChat processes an incoming chat message, publishes it to
// Kafka for cross-instance fan-out, and persists to ScyllaDB.
func (h *OpenAPIHandler) handleIncomingChat(client *ws.Client, roomID string, data json.RawMessage) {
	var chatData struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &chatData); err != nil {
		env := pooledErrorEnvelope("invalid chat message format")
		_ = client.SendJSON(env)
		env.Type = ""
		env.RoomID = ""
		env.Data = nil
		envelopePool.Put(env)
		return
	}

	// Persist message to ScyllaDB
	now := time.Now()
	msg := &domain.Message{
		ConversationID: roomID,
		SenderID:       client.UserID,
		Content:        chatData.Content,
		CreatedAt:      now,
	}

	if err := h.svc.SendMessage(context.Background(), msg); err != nil {
		h.log.Error("failed to persist chat message", slog.String("error", err.Error()))
		// Continue to broadcast even if persistence fails — the message
		// will be delivered via WebSocket but lost on reconnect.
	}

	chatMsg := domain.WSChatMessage{
		MessageID:      msg.ID,
		ConversationID: roomID,
		SenderID:       client.UserID,
		Content:        chatData.Content,
		CreatedAt:      now,
	}

	// 1. Always broadcast locally for immediate delivery.
	h.broadcastToRoom(roomID, chatMsg)

	// 2. Publish to Kafka for cross-instance fan-out (best-effort).
	if h.producer != nil {
		rawData, err := json.Marshal(chatMsg)
		if err != nil {
			h.log.Error("failed to marshal chat data", slog.String("error", err.Error()))
			return
		}
		if err := h.producer.Publish(context.Background(), &kafkainfra.Message{
			RoomID: roomID,
			Type:   domain.WSMsgTypeChat,
			Data:   rawData,
		}); err != nil {
			h.log.Warn("kafka publish failed",
				slog.String("error", err.Error()),
			)
		}
	}
}

// ─── Typing Indicator Handler ───────────────────────────────────────────

// handleIncomingTyping processes a typing indicator and fans it out.
func (h *OpenAPIHandler) handleIncomingTyping(client *ws.Client, roomID string, data json.RawMessage) {
	var typingData domain.WSTypingIndicator
	if err := json.Unmarshal(data, &typingData); err != nil {
		return
	}
	typingData.UserID = client.UserID

	envelope := envelopePool.Get().(*domain.WSEnvelope)
	envelope.Type = domain.WSMsgTypeTyping
	envelope.RoomID = roomID
	envelope.Data = typingData

	payload, err := json.Marshal(envelope)
	envelope.Type = ""
	envelope.RoomID = ""
	envelope.Data = nil
	envelopePool.Put(envelope)

	if err != nil {
		return
	}

	h.hub.SendToRoom(roomID, payload)
}

// ─── Broadcast Helpers ──────────────────────────────────────────────────

// broadcastToRoom marshals a chat event and delivers it to all locally-connected
// WebSocket clients in the room, using the pool for zero-allocation.
func (h *OpenAPIHandler) broadcastToRoom(conversationID string, chatMsg domain.WSChatMessage) {
	envelope := envelopePool.Get().(*domain.WSEnvelope)
	envelope.Type = domain.WSMsgTypeChat
	envelope.RoomID = conversationID
	envelope.Data = chatMsg

	payload, err := json.Marshal(envelope)
	envelope.Type = ""
	envelope.RoomID = ""
	envelope.Data = nil
	envelopePool.Put(envelope)

	if err != nil {
		h.log.Error("failed to marshal chat envelope", slog.String("error", err.Error()))
		return
	}
	h.hub.SendToRoom(conversationID, payload)
}

// broadcastNewMessage sends a new message to all WebSocket clients in the room.
// It always broadcasts locally for immediate delivery, and also publishes to
// Kafka for cross-instance fan-out (the consumer on each instance reads from
// Kafka and re-broadcasts locally).
//
// The local broadcast is critical because the Kafka producer is configured
// with Async: true and RequiredAcks: none, meaning Publish returns immediately
// without confirmation. Relying solely on the Kafka consumer loop for local
// delivery would introduce latency or risk message loss if the async write
// fails silently.
func (h *OpenAPIHandler) broadcastNewMessage(conversationID string, msg *domain.Message) {
	chatMsg := domain.WSChatMessage{
		MessageID:      msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		CreatedAt:      msg.CreatedAt,
	}

	// 1. Always broadcast locally for immediate delivery.
	h.broadcastToRoom(conversationID, chatMsg)

	// 2. Publish to Kafka for cross-instance fan-out (best-effort).
	if h.producer != nil {
		rawData, err := json.Marshal(chatMsg)
		if err != nil {
			h.log.Error("failed to marshal chat broadcast", slog.String("error", err.Error()))
			return
		}
		if err := h.producer.Publish(context.Background(), &kafkainfra.Message{
			RoomID: conversationID,
			Type:   domain.WSMsgTypeChat,
			Data:   rawData,
		}); err != nil {
			h.log.Warn("kafka publish failed",
				slog.String("error", err.Error()),
			)
		}
	}
}

// pooledErrorEnvelope returns a pooled WSEnvelope set to an error message.
// The caller must return the envelope to the pool after SendJSON completes.
func pooledErrorEnvelope(msg string) *domain.WSEnvelope {
	env := envelopePool.Get().(*domain.WSEnvelope)
	env.Type = domain.WSMsgTypeError
	env.RoomID = ""
	env.Data = map[string]string{"error": msg}
	return env
}

// broadcastRoomReady sends a "room_ready" event to a specific user's global
// connection, notifying them that a conversation has been created.
func (h *OpenAPIHandler) broadcastRoomReady(recipientID string, conversationID string) {
	envelope := envelopePool.Get().(*domain.WSEnvelope)
	envelope.Type = domain.WSMsgTypeRoomReady
	envelope.RoomID = conversationID
	envelope.Data = map[string]string{
		"conversation_id": conversationID,
	}

	payload, err := json.Marshal(envelope)
	envelope.Type = ""
	envelope.RoomID = ""
	envelope.Data = nil
	envelopePool.Put(envelope)

	if err != nil {
		h.log.Error("failed to marshal room_ready envelope", slog.String("error", err.Error()))
		return
	}

	// Send to the recipient if they are globally connected (no specific room)
	if client, ok := h.hub.GetClient(recipientID); ok {
		client.SendBytes(payload)
		h.log.Info("sent room_ready to globally connected user",
			slog.String("user_id", recipientID),
			slog.String("conversation_id", conversationID),
		)
	}

	// Also publish to Kafka for cross-instance delivery
	if h.producer != nil {
		rawData, err := json.Marshal(map[string]string{
			"conversation_id": conversationID,
		})
		if err != nil {
			h.log.Error("failed to marshal room_ready for kafka", slog.String("error", err.Error()))
			return
		}
		if err := h.producer.PublishToTopic(context.Background(), kafkainfra.LifecycleTopic, &kafkainfra.Message{
			RoomID: conversationID,
			Type:   domain.WSMsgTypeRoomReady,
			Data:   rawData,
		}); err != nil {
			h.log.Warn("kafka room_ready publish failed", slog.String("error", err.Error()))
		}
	}
}


