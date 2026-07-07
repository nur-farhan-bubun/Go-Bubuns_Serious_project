package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ride-sharing/location-service/internal/domain"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// LocationMessage is the JSON message sent to WebSocket clients.
type LocationMessage struct {
	Type      string  `json:"type"`
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Hub manages WebSocket connections for real-time location streaming.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool // userID -> set of connections
	logger  *slog.Logger
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[string]map[*websocket.Conn]bool),
		logger:  logger,
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and registers it.
// The client should connect with ?user_id=xxx to subscribe to their own location updates.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id query parameter required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade to WebSocket", "error", err)
		return
	}

	// Register the connection
	h.register(userID, conn)

	h.logger.Info("WebSocket client connected", "user_id", userID)

	// Keep connection alive and handle disconnection
	go func() {
		defer func() {
			h.unregister(userID, conn)
			conn.Close()
			h.logger.Info("WebSocket client disconnected", "user_id", userID)
		}()

		for {
			// Read messages to detect disconnection and handle pings
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// BroadcastLocation sends a location update to ALL connected WebSocket clients.
// Every client sees every user's real-time location, enabling multi-user tracking on the map.
func (h *Hub) BroadcastLocation(loc *domain.Location) {
	msg := LocationMessage{
		Type:      "location_update",
		UserID:    loc.UserID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal location message", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conns := range h.clients {
		for conn := range conns {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				h.logger.Warn("failed to write WebSocket message", "user_id", loc.UserID, "error", err)
			}
		}
	}
}

// register adds a connection to the hub.
func (h *Hub) register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]bool)
	}
	h.clients[userID][conn] = true
}

// unregister removes a connection from the hub.
func (h *Hub) unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if connections, ok := h.clients[userID]; ok {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(h.clients, userID)
		}
	}
}
