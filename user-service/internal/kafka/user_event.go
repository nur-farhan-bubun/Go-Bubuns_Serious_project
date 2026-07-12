package kafka

const (
	// UserTopic is the Kafka topic for user lifecycle events (created, updated, deleted).
	UserTopic = "user-events"
)

// UserEventType enumerates the types of user lifecycle events.
type UserEventType string

const (
	UserCreated UserEventType = "user.created"
	UserUpdated UserEventType = "user.updated"
	UserDeleted UserEventType = "user.deleted"
)

// UserEvent is the payload published to the user-events Kafka topic.
// The chat-service consumes these events to keep its user cache eventually consistent.
type UserEvent struct {
	Type        UserEventType `json:"type"`
	UserID      string        `json:"user_id"`
	Email       string        `json:"email"`
	DisplayName string        `json:"display_name"`
	AvatarURL   string        `json:"avatar_url,omitempty"`
}
