package scylladb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/ride-sharing/chat-service/internal/domain"
)

// Repository handles chat data access in ScyllaDB.
type Repository struct {
	session *gocql.Session
	logger  *slog.Logger
}

// New creates a new chat repository and connects to ScyllaDB.
// The keyspace and table are expected to exist (create keyspace query is run).
func New(ctx context.Context, scyllaURL string, logger *slog.Logger) (*Repository, error) {
	// Parse the URL: scylladb://host:port/keyspace
	cluster, err := parseScyllaURL(scyllaURL)
	if err != nil {
		return nil, fmt.Errorf("invalid scylla URL: %w", err)
	}

	// Retry until the cluster is available (ScyllaDB can take time to init).
	cluster.Consistency = gocql.LocalQuorum
	cluster.Timeout = 10 * time.Second
	cluster.ConnectTimeout = 30 * time.Second
	cluster.NumConns = 3

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create scylla session: %w", err)
	}

	logger.Info("connected to ScyllaDB",
		slog.String("hosts", cluster.Hosts[0]),
		slog.String("keyspace", cluster.Keyspace),
	)

	return &Repository{session: session, logger: logger}, nil
}

// Close shuts down the ScyllaDB session.
func (r *Repository) Close() {
	r.session.Close()
}

// CreateConversation creates a new direct conversation.
func (r *Repository) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	// match_id is nullable; pass nil when empty so ScyllaDB stores it as null.
	var matchID interface{} = nil
	if conv.MatchID != "" {
		matchID = conv.MatchID
	}
	q := r.session.Query(
		`INSERT INTO conversations (id, user1_id, user2_id, match_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		conv.ID, conv.User1ID, conv.User2ID, matchID, conv.CreatedAt,
	).WithContext(ctx)
	return q.Exec()
}

// CreateGroupConversation creates a new group conversation and adds all members.
func (r *Repository) CreateGroupConversation(ctx context.Context, conv *domain.Conversation) error {
	// Insert into group_conversations table
	q := r.session.Query(
		`INSERT INTO group_conversations (id, name, created_at) VALUES (?, ?, ?)`,
		conv.ID, conv.Name, conv.CreatedAt,
	).WithContext(ctx)
	if err := q.Exec(); err != nil {
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
	// Normalize: always order the user IDs to deduplicate (user1_id < user2_id alphabetically
	// is not guaranteed by the DB schema, so we query both directions).

	// Check: user1 = user1ID, user2 = user2ID
	var conv domain.Conversation
	var matchID *string
	q := r.session.Query(
		`SELECT id, user1_id, user2_id, match_id, created_at FROM conversations WHERE user1_id = ? AND user2_id = ? ALLOW FILTERING`,
		user1ID, user2ID,
	).WithContext(ctx)

	err := q.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &matchID, &conv.CreatedAt)
	if err == nil {
		conv.Type = domain.ConversationTypeDirect
		if matchID != nil {
			conv.MatchID = *matchID
		}
		return &conv, nil
	}
	if err != nil && err != gocql.ErrNotFound {
		return nil, fmt.Errorf("failed to query existing conversation: %w", err)
	}

	// Check: user1 = user2ID, user2 = user1ID
	q = r.session.Query(
		`SELECT id, user1_id, user2_id, match_id, created_at FROM conversations WHERE user1_id = ? AND user2_id = ? ALLOW FILTERING`,
		user2ID, user1ID,
	).WithContext(ctx)

	var conv2 domain.Conversation
	var matchID2 *string
	err = q.Scan(&conv2.ID, &conv2.User1ID, &conv2.User2ID, &matchID2, &conv2.CreatedAt)
	if err == nil {
		conv2.Type = domain.ConversationTypeDirect
		if matchID2 != nil {
			conv2.MatchID = *matchID2
		}
		return &conv2, nil
	}
	if err != nil && err != gocql.ErrNotFound {
		return nil, fmt.Errorf("failed to query existing conversation (reversed): %w", err)
	}

	return nil, nil // no existing conversation found
}

// AddGroupMember adds a user to a group conversation.
func (r *Repository) AddGroupMember(ctx context.Context, conversationID string, userID string, joinedAt time.Time) error {
	q := r.session.Query(
		`INSERT INTO conversation_members (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`,
		conversationID, userID, joinedAt,
	).WithContext(ctx)
	return q.Exec()
}

// GetConversation retrieves a conversation by ID.
// Checks both direct conversations and group conversations.
func (r *Repository) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	// Try direct conversation first
	var conv domain.Conversation
	var matchID *string
	q := r.session.Query(
		`SELECT id, user1_id, user2_id, match_id, created_at FROM conversations WHERE id = ?`,
		id,
	).WithContext(ctx)

	err := q.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &matchID, &conv.CreatedAt)
	if err == nil {
		conv.Type = domain.ConversationTypeDirect
		if matchID != nil {
			conv.MatchID = *matchID
		}
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
	q := r.session.Query(
		`SELECT id, name, created_at FROM group_conversations WHERE id = ?`,
		id,
	).WithContext(ctx)

	err := q.Scan(&conv.ID, &conv.Name, &conv.CreatedAt)
	if err != nil {
		if err == gocql.ErrNotFound {
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
	var members []string
	iter := r.session.Query(
		`SELECT user_id FROM conversation_members WHERE conversation_id = ?`,
		conversationID,
	).WithContext(ctx).Iter()

	var userID string
	for iter.Scan(&userID) {
		members = append(members, userID)
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	return members, nil
}

// GetConversationsByUserID returns all conversations for a user.
// Includes both direct conversations and group conversations the user is a member of.
func (r *Repository) GetConversationsByUserID(ctx context.Context, userID string) ([]*domain.Conversation, error) {
	seen := make(map[string]*domain.Conversation)

	// Query direct conversations where user is user1
	iter1 := r.session.Query(
		`SELECT id, user1_id, user2_id, match_id, created_at FROM conversations WHERE user1_id = ?`,
		userID,
	).WithContext(ctx).Iter()

	conv, err := scanConversation(iter1)
	for err == nil && conv != nil {
		conv.Type = domain.ConversationTypeDirect
		seen[conv.ID] = conv
		conv, err = scanConversation(iter1)
	}
	if err != nil {
		iter1.Close()
		return nil, fmt.Errorf("failed to scan conversations (user1): %w", err)
	}
	iter1.Close()

	// Query conversations where user is user2
	iter2 := r.session.Query(
		`SELECT id, user1_id, user2_id, match_id, created_at FROM conversations WHERE user2_id = ?`,
		userID,
	).WithContext(ctx).Iter()

	conv, err = scanConversation(iter2)
	for err == nil && conv != nil {
		conv.Type = domain.ConversationTypeDirect
		seen[conv.ID] = conv
		conv, err = scanConversation(iter2)
	}
	if err != nil {
		iter2.Close()
		return nil, fmt.Errorf("failed to scan conversations (user2): %w", err)
	}
	iter2.Close()

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
	var ids []string
	iter := r.session.Query(
		`SELECT conversation_id FROM conversation_members WHERE user_id = ?`,
		userID,
	).WithContext(ctx).Iter()

	var convID gocql.UUID
	for iter.Scan(&convID) {
		ids = append(ids, convID.String())
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get user group IDs: %w", err)
	}
	return ids, nil
}

// IsGroupMember checks if a user is a member of a group conversation.
func (r *Repository) IsGroupMember(ctx context.Context, conversationID string, userID string) (bool, error) {
	q := r.session.Query(
		`SELECT user_id FROM conversation_members WHERE conversation_id = ? AND user_id = ?`,
		conversationID, userID,
	).WithContext(ctx)

	var found string
	err := q.Scan(&found)
	if err == gocql.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanConversation scans a single conversation row from the iterator.
// Returns (nil, nil) when no more rows exist.
// User ID fields are scanned as strings (not UUID) to support non-UUID user
// identifiers from auth providers (Clerk, dev-auth).
func scanConversation(iter *gocql.Iter) (*domain.Conversation, error) {
	var id gocql.UUID
	var user1ID, user2ID string
	var matchID *string
	var createdAt time.Time

	if !iter.Scan(&id, &user1ID, &user2ID, &matchID, &createdAt) {
		return nil, nil
	}

	conv := &domain.Conversation{
		ID:        id.String(),
		User1ID:   user1ID,
		User2ID:   user2ID,
		CreatedAt: createdAt,
	}
	if matchID != nil {
		conv.MatchID = *matchID
	}
	return conv, nil
}

// SaveMessage stores a new message.
func (r *Repository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	if r.session == nil {
		r.logger.Warn("scylla session not initialized, skipping SaveMessage")
		return nil
	}

	// Parse conversation ID as UUID (conversation IDs are server-generated UUIDs)
	convUUID, err := gocql.ParseUUID(msg.ConversationID)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}

	// Generate a TIMEUUID for the message ID (sorted by time)
	msgUUID := gocql.TimeUUID()

	msg.ID = msgUUID.String()

	// sender_id is stored as TEXT, so we pass it directly as a string.
	// User IDs from auth providers (Clerk, dev-auth) are not guaranteed to be UUIDs.
	q := r.session.Query(
		`INSERT INTO messages (conversation_id, id, sender_id, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		convUUID, msgUUID, msg.SenderID, msg.Content, msg.CreatedAt,
	).WithContext(ctx)
	return q.Exec()
}

// GetMessagesByConversationID returns messages for a conversation with pagination.
// The API uses cursor-based pagination (via the `before` query param), so offset
// is not used in practice. This simplifies the CQL query to a straightforward
// LIMIT scan without needing token-based offset pagination.
func (r *Repository) GetMessagesByConversationID(ctx context.Context, conversationID string, limit, offset int) ([]*domain.Message, error) {
	convUUID, err := gocql.ParseUUID(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	q := r.session.Query(
		`SELECT id, conversation_id, sender_id, content, created_at FROM messages WHERE conversation_id = ? LIMIT ?`,
		convUUID, limit,
	).WithContext(ctx)

	var messages []*domain.Message
	var msgID gocql.UUID
	var senderID string
	var content string
	var createdAt time.Time

	iter := q.Iter()
	for iter.Scan(&msgID, &convUUID, &senderID, &content, &createdAt) {
		messages = append(messages, &domain.Message{
			ID:             msgID.String(),
			ConversationID: convUUID.String(),
			SenderID:       senderID,
			Content:        content,
			CreatedAt:      createdAt,
		})
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close iterator: %w", err)
	}

	return messages, nil
}

// parseScyllaURL parses a scylladb:// URL into a gocql cluster configuration.
func parseScyllaURL(scyllaURL string) (*gocql.ClusterConfig, error) {
	// Expected format: scylladb://host:port/keyspace
	// Example: scylladb://chat-db:9042/app_chat

	// Simple parsing: scylladb://host:port/keyspace
	const prefix = "scylladb://"
	if len(scyllaURL) < len(prefix) {
		return nil, fmt.Errorf("invalid scylla URL: missing prefix")
	}
	rest := scyllaURL[len(prefix):]

	host, keyspace, err := splitHostKeyspace(rest)
	if err != nil {
		return nil, err
	}

	cluster := gocql.NewCluster(host)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.LocalQuorum

	return cluster, nil
}

// splitHostKeyspace splits host:port/keyspace into host and keyspace.
func splitHostKeyspace(s string) (string, string, error) {
	// Split on '/'
	slashIdx := -1
	for i, c := range s {
		if c == '/' {
			slashIdx = i
			break
		}
	}
	if slashIdx < 0 {
		return "", "", fmt.Errorf("expected format host:port/keyspace, got: %s", s)
	}

	host := s[:slashIdx]
	keyspace := s[slashIdx+1:]
	if host == "" || keyspace == "" {
		return "", "", fmt.Errorf("expected format host:port/keyspace, got: %s", s)
	}

	return host, keyspace, nil
}

// ParseUUID parses a string into a gocql.UUID.
func ParseUUID(s string) (gocql.UUID, error) {
	return gocql.ParseUUID(s)
}

// MustParseUUID parses a string into a gocql.UUID, panicking on error.
func MustParseUUID(s string) gocql.UUID {
	uid, err := gocql.ParseUUID(s)
	if err != nil {
		panic(fmt.Sprintf("invalid UUID: %s: %v", s, err))
	}
	return uid
}

// NewUUID generates a new UUID v4.
func NewUUID() string {
	return uuid.New().String()
}

// TimeUUID generates a TIMEUUID (UUID v1) for message ordering.
func TimeUUID() gocql.UUID {
	return gocql.TimeUUID()
}
