package domain

import (
	"encoding/json"
	"time"
)

// Conversation represents a chat between two users.
type Conversation struct {
	ID        string    `json:"id"`
	User1ID   string    `json:"user1_id"`
	User2ID   string    `json:"user2_id"`
	MatchID   string    `json:"match_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a single message in a conversation.
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// Presence represents a user's online status.
type Presence struct {
	UserID   string    `json:"user_id"`
	Status   string    `json:"status"` // "online", "offline", "away"
	LastSeen time.Time `json:"last_seen"`
}

// WSMessageType enumerates the types of WebSocket messages.
type WSMessageType string

const (
	WSMsgTypeChat     WSMessageType = "chat_message"
	WSMsgTypeSystem   WSMessageType = "system"
	WSMsgTypePresence WSMessageType = "presence"
	WSMsgTypeTyping   WSMessageType = "typing"
	WSMsgTypeAck      WSMessageType = "ack"
	WSMsgTypeError    WSMessageType = "error"
)

// WSEnvelope is the outer wrapper for all WebSocket messages.
// It is marshalled exactly once before fan-out.
type WSEnvelope struct {
	Type   WSMessageType `json:"type"`
	RoomID string        `json:"room_id,omitempty"`
	Data   interface{}   `json:"data"`
}

// WSChatMessage is the payload for a chat_message WebSocket event.
type WSChatMessage struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// WSPresenceUpdate is the payload for a presence WebSocket event.
type WSPresenceUpdate struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

// WSTypingIndicator is the payload for a typing WebSocket event.
type WSTypingIndicator struct {
	UserID         string `json:"user_id"`
	ConversationID string `json:"conversation_id"`
	IsTyping       bool   `json:"is_typing"`
}

// WSIncomingMessage is the structure clients send over WebSocket.
type WSIncomingMessage struct {
	Type   WSMessageType   `json:"type"`
	RoomID string          `json:"room_id,omitempty"`
	Data   json.RawMessage `json:"data"`
}
