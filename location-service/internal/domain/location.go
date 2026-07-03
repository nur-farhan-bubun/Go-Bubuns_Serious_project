package domain

import "time"

// Location represents a user's geographic position.
type Location struct {
	UserID    string    `json:"user_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NearbyUser represents a user within a radius.
type NearbyUser struct {
	UserID   string  `json:"user_id"`
	Distance float64 `json:"distance"` // in meters
}
