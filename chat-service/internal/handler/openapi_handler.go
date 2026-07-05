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
	"github.com/ride-sharing/chat-service/api"
	"github.com/ride-sharing/chat-service/internal/config"
	"github.com/ride-sharing/chat-service/internal/domain"
	kafkainfra "github.com/ride-sharing/chat-service/internal/kafka"
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
	hub      *ws.Hub
	producer *kafkainfra.Producer
	log      *slog.Logger
}

// NewOpenAPIHandler creates a new handler.
func NewOpenAPIHandler(cfg *config.Config, hub *ws.Hub, producer *kafkainfra.Producer, log *slog.Logger) *OpenAPIHandler {
	return &OpenAPIHandler{cfg: cfg, hub: hub, producer: producer, log: log}
}

// GetConversations handles GET /v1/conversations.
func (h *OpenAPIHandler) GetConversations(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// CreateConversation handles POST /v1/conversations.
func (h *OpenAPIHandler) CreateConversation(ctx echo.Context) error {
	return ctx.JSON(http.StatusCreated, nil)
}

// GetMessages handles GET /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) GetMessages(ctx echo.Context, id string, params api.GetMessagesParams) error {
	return ctx.JSON(http.StatusOK, nil)
}

// SendMessage handles POST /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) SendMessage(ctx echo.Context, id string) error {
	return ctx.JSON(http.StatusCreated, nil)
}

// GetPresence handles GET /v1/presence/{userID}.
func (h *OpenAPIHandler) GetPresence(ctx echo.Context, userID string) error {
	return ctx.JSON(http.StatusOK, nil)
}

// upgrader is configured with tuned buffer sizes and compression off by default.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Compression is off by default to minimise CPU and latency.
	// Enable only when average payload exceeds several KB and bandwidth
	// is the proven bottleneck.
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
// Kafka for cross-instance fan-out, and returns immediately. The Kafka
// consumer on every instance will reconstruct the envelope and broadcast it
// to locally-connected clients via Hub.SendToRoom.
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

	chatMsg := domain.WSChatMessage{
		ConversationID: roomID,
		SenderID:       client.UserID,
		Content:        chatData.Content,
		CreatedAt:      time.Now(),
	}

	rawData, err := json.Marshal(chatMsg)
	if err != nil {
		h.log.Error("failed to marshal chat data", slog.String("error", err.Error()))
		return
	}

	// Publish to Kafka — the consumer on every instance will marshal the
	// WSEnvelope once and fan it out via Hub.SendToRoom.
	if h.producer != nil {
		if err := h.producer.Publish(context.Background(), &kafkainfra.Message{
			RoomID: roomID,
			Type:   domain.WSMsgTypeChat,
			Data:   rawData,
		}); err != nil {
			h.log.Error("failed to publish chat message to kafka",
				slog.String("error", err.Error()),
			)
		}
	} else {
		// Fallback: broadcast locally when Kafka is not configured.
		envelope := envelopePool.Get().(*domain.WSEnvelope)
		envelope.Type = domain.WSMsgTypeChat
		envelope.RoomID = roomID
		envelope.Data = chatMsg

		payload, err := json.Marshal(envelope)
		envelope.Type = ""
		envelope.RoomID = ""
		envelope.Data = nil
		envelopePool.Put(envelope)

		if err != nil {
			h.log.Error("failed to marshal chat envelope",
				slog.String("error", err.Error()),
			)
			return
		}
		h.hub.SendToRoom(roomID, payload)
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
