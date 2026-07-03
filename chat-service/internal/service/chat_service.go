package service

import (
	"context"

	"github.com/ride-sharing/chat-service/internal/domain"
	scyllarepo "github.com/ride-sharing/chat-service/internal/repository/scylladb"
	redisrepo "github.com/ride-sharing/chat-service/internal/repository/redis"
)

// Service handles chat business logic.
type Service struct {
	chatRepo    *scyllarepo.Repository
	presenceRepo *redisrepo.PresenceRepository
}

// New creates a new chat service.
func New(chatRepo *scyllarepo.Repository, presenceRepo *redisrepo.PresenceRepository) *Service {
	return &Service{chatRepo: chatRepo, presenceRepo: presenceRepo}
}

// SendMessage sends a message in a conversation.
func (s *Service) SendMessage(ctx context.Context, msg *domain.Message) error {
	return s.chatRepo.SaveMessage(ctx, msg)
}

// GetMessages returns paginated messages for a conversation.
func (s *Service) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*domain.Message, error) {
	return s.chatRepo.GetMessagesByConversationID(ctx, conversationID, limit, offset)
}

// GetConversations returns all conversations for a user.
func (s *Service) GetConversations(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	return s.chatRepo.GetConversationsByUserID(ctx, userID)
}
