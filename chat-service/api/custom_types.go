package api

// UserInfo is the public user profile returned by the chat-service user info endpoints.
// It is populated via the user-events Kafka topic for eventual consistency.
type UserInfo struct {
	UserId      *string `json:"user_id,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	AvatarUrl   *string `json:"avatar_url,omitempty"`
}
