package service

import (
	"context"
	"time"

	"github.com/ride-sharing/chat-service/internal/domain"
	scyllarepo "github.com/ride-sharing/chat-service/internal/repository/scylladb"
	redisrepo "github.com/ride-sharing/chat-service/internal/repository/redis"
)

// Service handles chat business logic.
type Service struct {
	chatRepo     *scyllarepo.Repository
	presenceRepo *redisrepo.PresenceRepository
}

// New creates a new chat service.
func New(chatRepo *scyllarepo.Repository, presenceRepo *redisrepo.PresenceRepository) *Service {
	return &Service{chatRepo: chatRepo, presenceRepo: presenceRepo}
}

// CreateConversation creates a new conversation between two users.
func (s *Service) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	return s.chatRepo.CreateConversation(ctx, conv)
}

// FindExistingConversation checks if a direct conversation already exists
// between the two users (in either order).
func (s *Service) FindExistingConversation(ctx context.Context, user1ID, user2ID string) (*domain.Conversation, error) {
	return s.chatRepo.FindExistingConversation(ctx, user1ID, user2ID)
}

// CreateGroupConversation creates a new group conversation.
func (s *Service) CreateGroupConversation(ctx context.Context, conv *domain.Conversation) error {
	return s.chatRepo.CreateGroupConversation(ctx, conv)
}

// GetConversations returns all conversations for a user.
func (s *Service) GetConversations(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	return s.chatRepo.GetConversationsByUserID(ctx, userID)
}

// GetConversation returns a single conversation by ID.
func (s *Service) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	return s.chatRepo.GetConversation(ctx, id)
}

// IsConversationParticipant checks if a user is a participant in a conversation.
func (s *Service) IsConversationParticipant(ctx context.Context, conv *domain.Conversation, userID string) bool {
	if conv.Type == domain.ConversationTypeDirect {
		return conv.User1ID == userID || conv.User2ID == userID
	}
	for _, memberID := range conv.MemberIDs {
		if memberID == userID {
			return true
		}
	}
	return false
}

// SendMessage saves a message in a conversation.
func (s *Service) SendMessage(ctx context.Context, msg *domain.Message) error {
	return s.chatRepo.SaveMessage(ctx, msg)
}

// GetMessages returns paginated messages for a conversation.
func (s *Service) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*domain.Message, error) {
	return s.chatRepo.GetMessagesByConversationID(ctx, conversationID, limit, offset)
}

// GetPresence retrieves a user's online presence.
// Returns a default offline status if the presence repo is not configured.
func (s *Service) GetPresence(ctx context.Context, userID string) (*domain.Presence, error) {
	if s.presenceRepo == nil {
		return &domain.Presence{
			UserID:   userID,
			Status:   "offline",
			LastSeen: time.Now(),
		}, nil
	}
	return s.presenceRepo.GetPresence(ctx, userID)
}

// SetPresence updates a user's online presence.
// No-op if the presence repo is not configured.
func (s *Service) SetPresence(ctx context.Context, presence *domain.Presence) error {
	if s.presenceRepo == nil {
		return nil
	}
	return s.presenceRepo.SetPresence(ctx, presence)
}
