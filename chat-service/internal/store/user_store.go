package store

import (
	"context"
	"sync"

	"github.com/ride-sharing/chat-service/internal/domain"
)

// MemoryUserStore is an in-memory, thread-safe cache of user info
// populated by the user-events Kafka consumer for eventual consistency.
type MemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]*domain.UserInfo
}

// NewMemoryUserStore creates a new in-memory user store.
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users: make(map[string]*domain.UserInfo),
	}
}

// Upsert creates or updates a user in the cache.
func (s *MemoryUserStore) Upsert(_ context.Context, user *domain.UserInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.UserID] = user
	return nil
}

// GetByID retrieves a user from the cache by ID.
func (s *MemoryUserStore) GetByID(_ context.Context, userID string) (*domain.UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[userID]
	if !ok {
		return nil, nil // not found is not an error
	}
	return user, nil
}

// List returns all cached users.
func (s *MemoryUserStore) List(_ context.Context) ([]*domain.UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*domain.UserInfo, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result, nil
}

// Delete removes a user from the cache.
func (s *MemoryUserStore) Delete(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, userID)
	return nil
}
