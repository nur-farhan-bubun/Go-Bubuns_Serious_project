package websocket

import (
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ride-sharing/chat-service/internal/domain"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	// Must be > pingPeriod so the connection stays alive.
	pongWait = 35 * time.Second

	// Send pings to peer with this period (30 seconds as per spec).
	pingPeriod = 30 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 4096

	// sendBufSize is the capacity of the outbound message channel.
	sendBufSize = 256
)

// Client represents a single WebSocket connection.
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string
	RoomID atomic.Value // stores string
	closed atomic.Bool  // guards against double-close of Send channel
	logger *slog.Logger
}

// NewClient creates a new Client.
func NewClient(hub *Hub, conn *websocket.Conn, userID string, logger *slog.Logger) *Client {
	return &Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, sendBufSize),
		UserID: userID,
		logger: logger.With(slog.String("user_id", userID)),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
//
// ReadPump is expected to be called as a goroutine. It runs a read loop
// that sets a ReadDeadline and reacts to Pong frames to keep the connection
// alive. Any read error or unexpected close terminates the loop and triggers
// client cleanup.
func (c *Client) ReadPump(handler func(client *Client, message []byte)) {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Error("read error", slog.String("error", err.Error()))
			}
			break
		}
		if handler != nil {
			handler(c, message)
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
//
// WritePump is expected to be called as a goroutine. It is the **single writer**
// goroutine for this connection — no other goroutine may call WriteMessage on
// the underlying *websocket.Conn. It sends a periodic Ping to confirm liveness
// and applies a strict WriteDeadline before every write.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				// Hub closed the channel — send close frame and exit.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Error("write error", slog.String("error", err.Error()))
				return
			}

		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON marshals msg once and enqueues it on the outbound channel.
// This is the safe way to send a structured message — it guarantees the
// single-marshal fan-out rule when called for broadcast.
func (c *Client) SendJSON(msg *domain.WSEnvelope) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	select {
	case c.Send <- data:
	default:
		// Backpressure: drop the oldest queued message so the client can
		// receive fresh data without blocking the hub.
		select {
		case <-c.Send:
			c.Send <- data
		default:
		}
	}
	return nil
}

// SendBytes enqueues pre-marshaled bytes on the outbound channel.
// This is used during fan-out to avoid re-marshalling the same payload.
func (c *Client) SendBytes(data []byte) {
	select {
	case c.Send <- data:
	default:
		// Backpressure: drop oldest message.
		select {
		case <-c.Send:
			c.Send <- data
		default:
		}
	}
}
