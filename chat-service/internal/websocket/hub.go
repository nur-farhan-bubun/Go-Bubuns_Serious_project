package websocket

import (
	"encoding/json"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ride-sharing/chat-service/internal/domain"
)

const numShards = 32

// RoomShard holds a subset of rooms to reduce lock contention.
type RoomShard struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
}

// newRoomShard creates an initialised RoomShard.
func newRoomShard() *RoomShard {
	return &RoomShard{
		rooms: make(map[string]map[*Client]struct{}),
	}
}

// Hub manages all active WebSocket connections and room subscriptions using
// sharded room maps for low-latency concurrent access.
type Hub struct {
	clients  map[string]*Client // userID -> *Client
	clientMu sync.RWMutex

	shards [numShards]*RoomShard

	logger *slog.Logger

	// OnPresenceChange is called whenever a user connects or disconnects.
	// It can be used by the handler layer to broadcast presence to Kafka/rooms.
	OnPresenceChange func(userID string, status string)
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger) *Hub {
	h := &Hub{
		clients: make(map[string]*Client),
		logger:  logger.With(slog.String("component", "ws_hub")),
	}
	for i := 0; i < numShards; i++ {
		h.shards[i] = newRoomShard()
	}
	return h
}

// hashRoomID deterministically maps a room ID to a shard index.
func (h *Hub) hashRoomID(roomID string) int {
	hw := fnv.New32a()
	_, _ = hw.Write([]byte(roomID))
	return int(hw.Sum32() % numShards)
}

// Register adds a client to the hub's client map.
func (h *Hub) Register(client *Client) {
	h.clientMu.Lock()
	h.clients[client.UserID] = client
	h.clientMu.Unlock()
	h.logger.Info("client registered", slog.String("user_id", client.UserID))

	// Broadcast presence online event
	if h.OnPresenceChange != nil {
		h.OnPresenceChange(client.UserID, "online")
	}
}

// Unregister removes a client from the hub and the room it was in.
func (h *Hub) Unregister(client *Client) {
	// Remove from client map
	h.clientMu.Lock()
	if _, ok := h.clients[client.UserID]; ok {
		delete(h.clients, client.UserID)
		if !client.closed.Swap(true) {
			close(client.Send)
		}
	}
	h.clientMu.Unlock()

	// Remove from room if subscribed
	if roomID, ok := client.RoomID.Load().(string); ok && roomID != "" {
		shard := h.shards[h.hashRoomID(roomID)]
		shard.mu.Lock()
		if clients, ok := shard.rooms[roomID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(shard.rooms, roomID)
			}
		}
		shard.mu.Unlock()
	}

	h.logger.Info("client unregistered", slog.String("user_id", client.UserID))

	// Broadcast presence offline event
	if h.OnPresenceChange != nil {
		h.OnPresenceChange(client.UserID, "offline")
	}
}

// JoinRoom subscribes a client to a room.
func (h *Hub) JoinRoom(client *Client, roomID string) {
	// Leave current room first if any
	if currentRoom, ok := client.RoomID.Load().(string); ok && currentRoom != "" && currentRoom != roomID {
		h.LeaveRoom(client, currentRoom)
	}

	shard := h.shards[h.hashRoomID(roomID)]
	shard.mu.Lock()
	if _, ok := shard.rooms[roomID]; !ok {
		shard.rooms[roomID] = make(map[*Client]struct{}, 64)
	}
	shard.rooms[roomID][client] = struct{}{}
	client.RoomID.Store(roomID)
	shard.mu.Unlock()

	h.logger.Info("client joined room",
		slog.String("user_id", client.UserID),
		slog.String("room_id", roomID),
	)
}

// LeaveRoom unsubscribes a client from a room.
func (h *Hub) LeaveRoom(client *Client, roomID string) {
	shard := h.shards[h.hashRoomID(roomID)]
	shard.mu.Lock()
	if clients, ok := shard.rooms[roomID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(shard.rooms, roomID)
		}
	}
	shard.mu.Unlock()

	client.RoomID.CompareAndSwap(roomID, "")

	h.logger.Info("client left room",
		slog.String("user_id", client.UserID),
		slog.String("room_id", roomID),
	)
}

// SendToRoom fans out pre-marshalled data to every client in a room.
// data must be the result of json.Marshal — call it once before fan-out.
func (h *Hub) SendToRoom(roomID string, data []byte) {
	shard := h.shards[h.hashRoomID(roomID)]
	shard.mu.RLock()
	clients := shard.rooms[roomID]
	shard.mu.RUnlock()

	for client := range clients {
		client.SendBytes(data)
	}
}

// Broadcast sends pre-marshalled data to ALL connected clients regardless of room.
// Used for global events like presence changes.
func (h *Hub) Broadcast(data []byte) {
	h.clientMu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.clientMu.RUnlock()

	for _, client := range clients {
		client.SendBytes(data)
	}
}

// BroadcastPresence marshals and sends a presence event to all connected clients.
func (h *Hub) BroadcastPresence(userID string, status string) {
	envelope := &domain.WSEnvelope{
		Type: domain.WSMsgTypePresence,
		Data: domain.WSPresenceUpdate{
			UserID: userID,
			Status: status,
		},
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		h.logger.Error("failed to marshal presence envelope", slog.String("error", err.Error()))
		return
	}
	h.Broadcast(data)
}

// GetClient returns a client by userID.
func (h *Hub) GetClient(userID string) (*Client, bool) {
	h.clientMu.RLock()
	defer h.clientMu.RUnlock()
	client, ok := h.clients[userID]
	return client, ok
}

// RoomClientCount returns the number of clients currently in a room.
func (h *Hub) RoomClientCount(roomID string) int {
	shard := h.shards[h.hashRoomID(roomID)]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return len(shard.rooms[roomID])
}

// ClientCount returns the total number of connected clients.
func (h *Hub) ClientCount() int {
	h.clientMu.RLock()
	defer h.clientMu.RUnlock()
	return len(h.clients)
}

// Shutdown gracefully closes all active WebSocket connections and clears
// all room state. It sends a close frame to each client, drains the send
// channel, and cleans up references so the goroutine pumps exit cleanly.
func (h *Hub) Shutdown() {
	h.logger.Info("hub shutting down, draining connections")

	// Snapshot all clients under lock, then clear maps immediately so that
	// any concurrent Unregister calls from the read pumps become no-ops.
	h.clientMu.Lock()
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.clients = make(map[string]*Client)

	for i := range h.shards {
		h.shards[i].mu.Lock()
		h.shards[i].rooms = make(map[string]map[*Client]struct{})
		h.shards[i].mu.Unlock()
	}
	h.clientMu.Unlock()

	// Close every client outside the lock so pump defers can acquire it.
	for _, c := range clients {
		// Send a close frame so the peer sees a clean shutdown.
		_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
		_ = c.Conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutting down"),
		)
		// Closing the TCP connection causes ReadPump's ReadMessage to error
		// out, which triggers its deferred Unregister + conn.Close.
		c.Conn.Close()

		// Close the send channel so WritePump exits. The closed flag
		// prevents Unregister from double-closing if it fires first.
		if !c.closed.Swap(true) {
			close(c.Send)
		}
	}

	h.logger.Info("hub shutdown complete", slog.Int("disconnected", len(clients)))
}
