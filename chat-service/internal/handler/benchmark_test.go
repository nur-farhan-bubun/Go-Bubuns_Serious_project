package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ride-sharing/chat-service/internal/domain"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

// fixedTime is a constant timestamp so json.Marshal output is deterministic
// across benchmark iterations.
var fixedTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// benchClient creates a minimal ws.Client via NewClient for benchmarks.
// A real TCP connection is not established — Conn is set to nil, which is safe
// because the hot-path handler methods only access Hub, Send, UserID, RoomID,
// and logger.
func benchClient(hub *ws.Hub, userID, roomID string) *ws.Client {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := ws.NewClient(hub, nil, userID, logger)
	c.RoomID.Store(roomID)
	hub.Register(c)
	hub.JoinRoom(c, roomID)
	return c
}

// drainSend consumes from the client's Send channel in a background
// goroutine so benchmarks don't block when the buffer fills.
func drainSend(c *ws.Client) (cleanup func()) {
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-c.Send:
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

// ---------------------------------------------------------------------------
// Hot-path benchmarks — compare the full handleIncomingMessage pipeline with
// and without sync.Pool.
// ---------------------------------------------------------------------------

// BenchmarkHotPath_Pooled benchmarks the real handleIncomingChat fallback
// path with sync.Pool enabled (current implementation, producer=nil).
func BenchmarkHotPath_Pooled(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := ws.NewHub(logger)
	h := NewOpenAPIHandler(nil, hub, nil, logger)
	client := benchClient(hub, "user-1", "room-1")
	stop := drainSend(client)
	defer stop()

	chatPayload := json.RawMessage(`{"content":"Hello, World!"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.handleIncomingChat(client, "room-1", chatPayload)
	}
}

// BenchmarkHotPath_NoPool benchmarks handleIncomingChat with the same steps
// but without any sync.Pool, for a fair apples-to-apples A/B comparison.
func BenchmarkHotPath_NoPool(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := ws.NewHub(logger)
	client := benchClient(hub, "user-1", "room-1")
	stop := drainSend(client)
	defer stop()

	noop := &noPoolHandler{hub: hub}
	chatPayload := json.RawMessage(`{"content":"Hello, World!"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		noop.handleIncomingChat(client, "room-1", chatPayload)
	}
}

// noPoolHandler reproduces the exact logic of the real handleIncomingChat
// fallback path but without any sync.Pool usage.
type noPoolHandler struct {
	hub *ws.Hub
}

func (h *noPoolHandler) handleIncomingChat(client *ws.Client, roomID string, data json.RawMessage) {
	var chatData struct {
		Content string `json:"content"`
	}
	json.Unmarshal(data, &chatData) //nolint:errcheck — same as real path

	chatMsg := domain.WSChatMessage{
		ConversationID: roomID,
		SenderID:       client.UserID,
		Content:        chatData.Content,
		CreatedAt:      fixedTime,
	}
	json.Marshal(chatMsg) //nolint:errcheck — same extra marshal as real handler

	// Direct allocation — no pool.
	envelope := &domain.WSEnvelope{
		Type:   domain.WSMsgTypeChat,
		RoomID: roomID,
		Data:   chatMsg,
	}

	payload, _ := json.Marshal(envelope)
	h.hub.SendToRoom(roomID, payload)
}

// ---------------------------------------------------------------------------
// Micro-benchmarks — measure pool impact on individual types in isolation.
// ---------------------------------------------------------------------------

// BenchmarkIncomingMsgPooling compares direct allocation vs pooling for
// WSIncomingMessage (the struct allocated per incoming WebSocket frame).
func BenchmarkIncomingMsgPooling(b *testing.B) {
	raw := []byte(`{"type":"chat_message","room_id":"room-1","data":{"content":"Hello"}}`)
	pool := sync.Pool{New: func() any { return &domain.WSIncomingMessage{} }}

	b.Run("direct-alloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var msg domain.WSIncomingMessage
			json.Unmarshal(raw, &msg)
		}
	})

	b.Run("pooled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			msg := pool.Get().(*domain.WSIncomingMessage)
			json.Unmarshal(raw, msg)
			*msg = domain.WSIncomingMessage{}
			pool.Put(msg)
		}
	})
}

// BenchmarkEnvelopePooling compares direct allocation vs pooling for
// WSEnvelope (the struct allocated per outgoing broadcast).
func BenchmarkEnvelopePooling(b *testing.B) {
	pool := sync.Pool{New: func() any { return &domain.WSEnvelope{} }}

	b.Run("direct-alloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			env := &domain.WSEnvelope{
				Type:   domain.WSMsgTypeChat,
				RoomID: "room-1",
				Data:   domain.WSChatMessage{Content: "Hello"},
			}
			json.Marshal(env)
		}
	})

	b.Run("pooled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			env := pool.Get().(*domain.WSEnvelope)
			env.Type = domain.WSMsgTypeChat
			env.RoomID = "room-1"
			env.Data = domain.WSChatMessage{Content: "Hello"}
			json.Marshal(env)
			env.Type = ""
			env.RoomID = ""
			env.Data = nil
			pool.Put(env)
		}
	})
}

// Ensure the gorilla/websocket import is used (it's needed for the
// compilation of ws.NewClient, even though we pass nil).
var _ = websocket.ErrCloseSent
