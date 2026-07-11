package kafka

// UserTopic is the Kafka topic for user lifecycle events.
const UserTopic = "user-events"

// UserEventType enumerates the types of user lifecycle events.
type UserEventType string

const (
	UserCreated UserEventType = "user.created"
	UserUpdated UserEventType = "user.updated"
	UserDeleted UserEventType = "user.deleted"
)

// UserEvent is the payload received from the user-events Kafka topic.
type UserEvent struct {
	Type        UserEventType `json:"type"`
	UserID      string        `json:"user_id"`
	Email       string        `json:"email"`
	DisplayName string        `json:"display_name"`
	AvatarURL   string        `json:"avatar_url,omitempty"`
}
