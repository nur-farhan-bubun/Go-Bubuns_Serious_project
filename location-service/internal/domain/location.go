package domain

import "time"

// Location represents a user's geographic position.
type Location struct {
	UserID    string    `json:"user_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MapPost represents a permanent map pin with user-generated content.
type MapPost struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	Category  string    `json:"category"` // RESTAURANT, SOCIAL_LIFE, EVENT
	ImageURLs []string  `json:"image_urls,omitempty"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
