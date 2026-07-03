package scylladb

import (
	"context"

	"github.com/ride-sharing/chat-service/internal/domain"
)

// Repository handles chat data access in ScyllaDB.
type Repository struct {
	// TODO: add *gocql.Session
}

// New creates a new chat repository.
func New() *Repository {
	return &Repository{}
}

// CreateConversation creates a new conversation.
func (r *Repository) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	return nil
}

// GetConversation retrieves a conversation by ID.
func (r *Repository) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	return nil, nil
}

// GetConversationsByUserID returns all conversations for a user.
func (r *Repository) GetConversationsByUserID(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	return nil, nil
}

// SaveMessage stores a new message.
func (r *Repository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	return nil
}

// GetMessagesByConversationID returns messages for a conversation.
func (r *Repository) GetMessagesByConversationID(ctx context.Context, conversationID string, limit, offset int) ([]*domain.Message, error) {
	return nil, nil
}
