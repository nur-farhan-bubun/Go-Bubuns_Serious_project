package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ride-sharing/chat-service/internal/domain"
)

// Repository handles chat data access in PostgreSQL.
type Repository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// New creates a new chat repository and connects to PostgreSQL.
func New(ctx context.Context, databaseURL string, logger *slog.Logger) (*Repository, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("connected to PostgreSQL",
		slog.String("database", config.ConnConfig.Database),
		slog.String("host", config.ConnConfig.Host),
	)

	return &Repository{pool: pool, logger: logger}, nil
}

// Close shuts down the connection pool.
func (r *Repository) Close() {
	r.pool.Close()
}

// Pool returns the underlying connection pool for reuse by other repositories.
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// NewUUID generates a new UUID v4 string.
func NewUUID() string {
	return uuid.New().String()
}

// CreateConversation creates a new direct conversation.
func (r *Repository) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	var matchID *string
	if conv.MatchID != "" {
		matchID = &conv.MatchID
	}
	query := `
		INSERT INTO conversations (id, user1_id, user2_id, match_id, created_at, type)
		VALUES ($1, $2, $3, $4, $5, 'direct')
	`
	_, err := r.pool.Exec(ctx, query, conv.ID, conv.User1ID, conv.User2ID, matchID, conv.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}
	return nil
}

// CreateGroupConversation creates a new group conversation and adds all members.
func (r *Repository) CreateGroupConversation(ctx context.Context, conv *domain.Conversation) error {
	// Insert into group_conversations table
	query := `
		INSERT INTO group_conversations (id, name, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.pool.Exec(ctx, query, conv.ID, conv.Name, conv.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create group conversation: %w", err)
	}

	// Add all members
	for _, memberID := range conv.MemberIDs {
		if err := r.AddGroupMember(ctx, conv.ID, memberID, conv.CreatedAt); err != nil {
			return fmt.Errorf("failed to add member %s: %w", memberID, err)
		}
	}

	return nil
}

// FindExistingConversation checks if a direct conversation already exists
// between two users (in either order). Returns nil if none found.
func (r *Repository) FindExistingConversation(ctx context.Context, user1ID, user2ID string) (*domain.Conversation, error) {
	query := `
		SELECT id, user1_id, user2_id, COALESCE(match_id, ''), created_at
		FROM conversations
		WHERE (user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1)
		LIMIT 1
	`

	var conv domain.Conversation
	var matchID string
	err := r.pool.QueryRow(ctx, query, user1ID, user2ID).Scan(
		&conv.ID, &conv.User1ID, &conv.User2ID, &matchID, &conv.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query existing conversation: %w", err)
	}

	conv.Type = domain.ConversationTypeDirect
	conv.MatchID = matchID
	return &conv, nil
}

// AddGroupMember adds a user to a group conversation.
func (r *Repository) AddGroupMember(ctx context.Context, conversationID string, userID string, joinedAt time.Time) error {
	query := `
		INSERT INTO conversation_members (conversation_id, user_id, joined_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.pool.Exec(ctx, query, conversationID, userID, joinedAt)
	if err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}
	return nil
}

// GetConversation retrieves a conversation by ID.
// Checks both direct conversations and group conversations.
func (r *Repository) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	// Try direct conversation first
	var conv domain.Conversation
	var matchID string
	query := `
		SELECT id, user1_id, user2_id, COALESCE(match_id, ''), created_at
		FROM conversations WHERE id = $1
	`
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.User1ID, &conv.User2ID, &matchID, &conv.CreatedAt,
	)
	if err == nil {
		conv.Type = domain.ConversationTypeDirect
		conv.MatchID = matchID
		return &conv, nil
	}

	// Try group conversation
	groupConv, err := r.GetGroupConversation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	return groupConv, nil
}

// GetGroupConversation retrieves a group conversation by ID.
func (r *Repository) GetGroupConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	var conv domain.Conversation
	query := `
		SELECT id, COALESCE(name, ''), created_at
		FROM group_conversations WHERE id = $1
	`
	err := r.pool.QueryRow(ctx, query, id).Scan(&conv.ID, &conv.Name, &conv.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("group conversation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get group conversation: %w", err)
	}
	conv.Type = domain.ConversationTypeGroup

	// Load members
	members, err := r.GetGroupMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	conv.MemberIDs = members

	return &conv, nil
}

// GetGroupMembers returns all member user IDs for a group conversation.
func (r *Repository) GetGroupMembers(ctx context.Context, conversationID string) ([]string, error) {
	query := `SELECT user_id FROM conversation_members WHERE conversation_id = $1 ORDER BY joined_at`

	rows, err := r.pool.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan group member: %w", err)
		}
		members = append(members, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate group members: %w", err)
	}
	return members, nil
}

// GetConversationsByUserID returns all conversations for a user.
// Includes both direct conversations and group conversations the user is a member of.
func (r *Repository) GetConversationsByUserID(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	seen := make(map[string]*domain.Conversation)

	// Query direct conversations where user is either user1 or user2
	query := `
		SELECT id, user1_id, user2_id, COALESCE(match_id, ''), created_at
		FROM conversations
		WHERE user1_id = $1 OR user2_id = $1
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query direct conversations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var conv domain.Conversation
		var matchID string
		if err := rows.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &matchID, &conv.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conv.Type = domain.ConversationTypeDirect
		conv.MatchID = matchID
		seen[conv.ID] = &conv
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate conversations: %w", err)
	}

	// Query group conversations where user is a member
	groupIDs, err := r.GetUserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, groupID := range groupIDs {
		if _, exists := seen[groupID]; !exists {
			groupConv, err := r.GetGroupConversation(ctx, groupID)
			if err == nil {
				seen[groupID] = groupConv
			}
		}
	}

	result := make([]*domain.Conversation, 0, len(seen))
	for _, c := range seen {
		result = append(result, c)
	}
	return result, nil
}

// GetUserGroupIDs returns all group conversation IDs a user is a member of.
func (r *Repository) GetUserGroupIDs(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT conversation_id FROM conversation_members WHERE user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user group IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan group ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate group IDs: %w", err)
	}
	return ids, nil
}

// IsGroupMember checks if a user is a member of a group conversation.
func (r *Repository) IsGroupMember(ctx context.Context, conversationID string, userID string) (bool, error) {
	query := `SELECT user_id FROM conversation_members WHERE conversation_id = $1 AND user_id = $2`

	var found string
	err := r.pool.QueryRow(ctx, query, conversationID, userID).Scan(&found)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SaveMessage stores a new message.
func (r *Repository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	// Generate a UUID v4 for the message ID
	msgID := uuid.New().String()
	msg.ID = msgID

	query := `
		INSERT INTO messages (id, conversation_id, sender_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, msgID, msg.ConversationID, msg.SenderID, msg.Content, msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

// GetMessagesByConversationID returns messages for a conversation with pagination.
func (r *Repository) GetMessagesByConversationID(ctx context.Context, conversationID string, limit, offset int) ([]*domain.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate messages: %w", err)
	}
	return messages, nil
}
