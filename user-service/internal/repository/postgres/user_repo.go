package postgres

import (
	"context"
	"fmt"
	"sync"

	"github.com/ride-sharing/user-service/internal/domain"
)

// Repository handles user data access.
// Uses in-memory storage until PostgreSQL is connected.
type Repository struct {
	mu    sync.RWMutex
	users map[string]*domain.User // keyed by ID
	emails map[string]string      // email → ID lookup
}

// New creates a new user repository.
func New() *Repository {
	return &Repository{
		users:  make(map[string]*domain.User),
		emails: make(map[string]string),
	}
}

// GetByID retrieves a user by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return user, nil
}

// FindByEmail retrieves a user by email.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.emails[email]
	if !ok {
		return nil, nil // not found is not an error
	}
	user, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("inconsistent state: email %s points to missing user %s", email, id)
	}
	return user, nil
}

// Create inserts a new user.
func (r *Repository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = user
	r.emails[user.Email] = user.ID
	return nil
}

// Update updates an existing user.
func (r *Repository) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[user.ID]; !ok {
		return fmt.Errorf("user not found: %s", user.ID)
	}
	// If email changed, update the email index
	old, ok := r.users[user.ID]
	if ok && old.Email != user.Email {
		delete(r.emails, old.Email)
		r.emails[user.Email] = user.ID
	}
	r.users[user.ID] = user
	return nil
}

// Delete removes a user by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	user, ok := r.users[id]
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	delete(r.emails, user.Email)
	delete(r.users, id)
	return nil
}

// List returns a paginated list of users.
func (r *Repository) List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.users)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	if start >= total {
		return []*domain.User{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	users := make([]*domain.User, 0, end-start)
	i := 0
	for _, u := range r.users {
		if i >= start && i < end {
			users = append(users, u)
		}
		i++
	}
	return users, total, nil
}
