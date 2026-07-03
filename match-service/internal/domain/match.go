package domain

import "time"

// Swipe represents a left/right swipe action.
type Swipe struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TargetID  string    `json:"target_id"`
	Direction string    `json:"direction"` // "left" or "right"
	CreatedAt time.Time `json:"created_at"`
}

// Match represents a mutual like between two users.
type Match struct {
	ID        string    `json:"id"`
	User1ID   string    `json:"user1_id"`
	User2ID   string    `json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`
}
