package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ride-sharing/chat-service/api"
	"github.com/labstack/echo/v4"
	
	"github.com/ride-sharing/chat-service/internal/config"
	"github.com/ride-sharing/chat-service/internal/domain"
	kafkainfra "github.com/ride-sharing/chat-service/internal/kafka"
	scyllarepo "github.com/ride-sharing/chat-service/internal/repository/scylladb"
	"github.com/ride-sharing/chat-service/internal/service"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

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

// Compile-time check that OpenAPIHandler implements api.ServerInterface.
var _ api.ServerInterface = (*OpenAPIHandler)(nil)

// OpenAPIHandler implements the generated api.ServerInterface and
// manages WebSocket connections via a sharded Hub, publishing chat
// events to Kafka for cross-instance fan-out.
type OpenAPIHandler struct {
	cfg      *config.Config
	svc      *service.Service
	hub      *ws.Hub
	producer *kafkainfra.Producer
	log      *slog.Logger
}

// NewOpenAPIHandler creates a new handler.
func NewOpenAPIHandler(cfg *config.Config, svc *service.Service, hub *ws.Hub, producer *kafkainfra.Producer, log *slog.Logger) *OpenAPIHandler {
	return &OpenAPIHandler{cfg: cfg, svc: svc, hub: hub, producer: producer, log: log}
}

// errResp returns a map suitable for JSON error responses.
func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}

// extractUserID extracts the authenticated user ID from the request.
// The API gateway sets the X-User-ID header after JWT validation.
// Falls back to query param for WebSocket connections.
func (h *OpenAPIHandler) extractUserID(c echo.Context) string {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		userID = c.QueryParam("user_id")
	}
	return userID
}

// GetConversations handles GET /v1/conversations.
func (h *OpenAPIHandler) GetConversations(ctx echo.Context) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	convs, err := h.svc.GetConversations(ctx.Request().Context(), userID)
	if err != nil {
		h.log.Error("failed to get conversations", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get conversations"))
	}

	result := make([]api.Conversation, 0, len(convs))
	for _, c := range convs {
		result = append(result, toAPIConversation(c))
	}
	return ctx.JSON(http.StatusOK, result)
}

// CreateConversation handles POST /v1/conversations.
func (h *OpenAPIHandler) CreateConversation(ctx echo.Context) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	var req api.CreateConversationRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResp("invalid request body: " + err.Error()))
	}

	if req.UserId == "" {
		return ctx.JSON(http.StatusBadRequest, errResp("user_id is required"))
	}

	// Validate that the other user is not the same as the current user
	if req.UserId == userID {
		return ctx.JSON(http.StatusBadRequest, errResp("cannot create conversation with yourself"))
	}

	matchID := ""
	if req.MatchId != nil {
		matchID = *req.MatchId
	}

	now := time.Now().UTC()
	conv := &domain.Conversation{
		ID:        scyllarepo.NewUUID(),
		User1ID:   userID,
		User2ID:   req.UserId,
		MatchID:   matchID,
		CreatedAt: now,
	}

	if err := h.svc.CreateConversation(ctx.Request().Context(), conv); err != nil {
		h.log.Error("failed to create conversation", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to create conversation"))
	}

	return ctx.JSON(http.StatusCreated, toAPIConversation(conv))
}

// GetMessages handles GET /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) GetMessages(ctx echo.Context, id string, params api.GetMessagesParams) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	// Verify the user is a participant in this conversation
	conv, err := h.svc.GetConversation(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, errResp("conversation not found"))
	}
	if conv.User1ID != userID && conv.User2ID != userID {
		return ctx.JSON(http.StatusForbidden, errResp("not a participant in this conversation"))
	}

	limit := 50
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
		if limit > 200 {
			limit = 200
		}
	}

	messages, err := h.svc.GetMessages(ctx.Request().Context(), id, limit, 0)
	if err != nil {
		h.log.Error("failed to get messages", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get messages"))
	}

	result := make([]api.Message, 0, len(messages))
	for _, m := range messages {
		result = append(result, toAPIMessage(m))
	}
	return ctx.JSON(http.StatusOK, result)
}

// SendMessage handles POST /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) SendMessage(ctx echo.Context, id string) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	var req api.SendMessageRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResp("invalid request body: " + err.Error()))
	}

	if req.Content == "" {
		return ctx.JSON(http.StatusBadRequest, errResp("content is required"))
	}

	// Verify the conversation exists and user is a participant
	conv, err := h.svc.GetConversation(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, errResp("conversation not found"))
	}
	if conv.User1ID != userID && conv.User2ID != userID {
		return ctx.JSON(http.StatusForbidden, errResp("not a participant in this conversation"))
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ConversationID: id,
		SenderID:       userID,
		Content:        req.Content,
		CreatedAt:      now,
	}

	if err := h.svc.SendMessage(ctx.Request().Context(), msg); err != nil {
		h.log.Error("failed to save message", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to send message"))
	}

	// Broadcast the message via WebSocket for real-time delivery
	h.broadcastNewMessage(id, msg)

	return ctx.JSON(http.StatusCreated, toAPIMessage(msg))
}

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
// It first attempts Kafka (cross-instance fan-out), then falls back to local
// broadcast when Kafka is unavailable or fails.
func (h *OpenAPIHandler) broadcastNewMessage(conversationID string, msg *domain.Message) {
	chatMsg := domain.WSChatMessage{
		MessageID:      msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		CreatedAt:      msg.CreatedAt,
	}

	rawData, err := json.Marshal(chatMsg)
	if err != nil {
		h.log.Error("failed to marshal chat broadcast", slog.String("error", err.Error()))
		return
	}

	if h.producer != nil {
		if err := h.producer.Publish(context.Background(), &kafkainfra.Message{
			RoomID: conversationID,
			Type:   domain.WSMsgTypeChat,
			Data:   rawData,
		}); err != nil {
			h.log.Warn("kafka publish failed, falling back to local broadcast",
				slog.String("error", err.Error()),
			)
			// Fallback: broadcast locally so the message still reaches
			// WebSocket clients connected to this instance.
			h.broadcastToRoom(conversationID, chatMsg)
		}
		// Kafka publish succeeded — the consumer on every instance will
		// fan out the message locally (handled by the consumer loop).
	} else {
		h.broadcastToRoom(conversationID, chatMsg)
	}
}

// GetPresence handles GET /v1/presence/{userID}.
func (h *OpenAPIHandler) GetPresence(ctx echo.Context, userID string) error {
	presence, err := h.svc.GetPresence(ctx.Request().Context(), userID)
	if err != nil {
		h.log.Error("failed to get presence", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get presence"))
	}

	status := api.PresenceStatus(presence.Status)
	return ctx.JSON(http.StatusOK, api.Presence{
		UserId:   &presence.UserID,
		Status:   &status,
		LastSeen: &presence.LastSeen,
	})
}

// upgrader is configured with tuned buffer sizes and compression off by default.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	EnableCompression: false,
	CheckOrigin:       func(r *http.Request) bool { return true },
}

// HandleWebSocket upgrades the HTTP connection to WebSocket, creates a Client,
// registers it with the Hub, and starts the read/write pump goroutines.
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
	}

	// Update presence to online
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

// pooledErrorEnvelope returns a pooled WSEnvelope set to an error message.
// The caller must return the envelope to the pool after SendJSON completes.
func pooledErrorEnvelope(msg string) *domain.WSEnvelope {
	env := envelopePool.Get().(*domain.WSEnvelope)
	env.Type = domain.WSMsgTypeError
	env.RoomID = ""
	env.Data = map[string]string{"error": msg}
	return env
}

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

	rawData, err := json.Marshal(chatMsg)
	if err != nil {
		h.log.Error("failed to marshal chat data", slog.String("error", err.Error()))
		return
	}

	// Publish to Kafka for cross-instance fan-out. If Kafka is unavailable,
	// fall back to local broadcast so connected clients still receive the message.
	if h.producer != nil {
		if err := h.producer.Publish(context.Background(), &kafkainfra.Message{
			RoomID: roomID,
			Type:   domain.WSMsgTypeChat,
			Data:   rawData,
		}); err != nil {
			h.log.Warn("kafka publish failed, falling back to local broadcast",
				slog.String("error", err.Error()),
			)
			h.broadcastToRoom(roomID, chatMsg)
		}
	} else {
		h.broadcastToRoom(roomID, chatMsg)
	}
}

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

// ─── Conversion Helpers ─────────────────────────────────────────────────────

func toAPIConversation(c *domain.Conversation) api.Conversation {
	id := c.ID
	user1ID := c.User1ID
	user2ID := c.User2ID
	createdAt := c.CreatedAt
	matchID := c.MatchID

	return api.Conversation{
		Id:        &id,
		User1Id:   &user1ID,
		User2Id:   &user2ID,
		MatchId:   &matchID,
		CreatedAt: &createdAt,
	}
}

func toAPIMessage(m *domain.Message) api.Message {
	id := m.ID
	convID := m.ConversationID
	senderID := m.SenderID
	content := m.Content
	createdAt := m.CreatedAt

	return api.Message{
		Id:             &id,
		ConversationId: &convID,
		SenderId:       &senderID,
		Content:        &content,
		CreatedAt:      &createdAt,
	}
}
