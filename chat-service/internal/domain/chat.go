package domain

import "time"

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
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"` // "online", "offline", "away"
	LastSeen  time.Time `json:"last_seen"`
}
