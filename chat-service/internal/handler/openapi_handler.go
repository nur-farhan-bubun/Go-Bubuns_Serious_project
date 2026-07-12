package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/ride-sharing/chat-service/api"
	"github.com/ride-sharing/chat-service/internal/config"
	"github.com/ride-sharing/chat-service/internal/domain"
	kafkainfra "github.com/ride-sharing/chat-service/internal/kafka"
	"github.com/ride-sharing/chat-service/internal/store"
	userClient "github.com/ride-sharing/chat-service/internal/userclient"
	scyllarepo "github.com/ride-sharing/chat-service/internal/repository/scylladb"
	"github.com/ride-sharing/chat-service/internal/service"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

// Compile-time check that OpenAPIHandler implements api.ServerInterface.
var _ api.ServerInterface = (*OpenAPIHandler)(nil)

// OpenAPIHandler implements the generated api.ServerInterface and
// manages WebSocket connections via a sharded Hub, publishing chat
// events to Kafka for cross-instance fan-out.
type OpenAPIHandler struct {
	cfg             *config.Config
	svc             *service.Service
	hub             *ws.Hub
	producer        *kafkainfra.Producer
	log             *slog.Logger
	userStore       *store.MemoryUserStore
	httpClient      *http.Client
	userGRPCClient  *userClient.Client
}

// NewOpenAPIHandler creates a new handler.
func NewOpenAPIHandler(cfg *config.Config, svc *service.Service, hub *ws.Hub, producer *kafkainfra.Producer, log *slog.Logger, userStore *store.MemoryUserStore, userGRPCClient *userClient.Client) *OpenAPIHandler {
	return &OpenAPIHandler{
		cfg:        cfg,
		svc:        svc,
		hub:        hub,
		producer:   producer,
		log:        log,
		userStore:  userStore,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		userGRPCClient: userGRPCClient,
	}
}

// errResp returns a map suitable for JSON error responses.
func errResp(msg string) map[string]string {
	return map[string]string{"error": msg}
}

// extractUserID extracts the authenticated user ID from the request.
// The API gateway sets the X-User-ID header after JWT validation.
// Falls back to query param for WebSocket connections.
func (h *OpenAPIHandler) extractUserID(c echo.Context) string {
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		userID = c.QueryParam("user_id")
	}
	return userID
}

func (h *OpenAPIHandler) isBlocked(ctx context.Context, senderID, otherUserID string) bool {
	if h.userGRPCClient == nil || otherUserID == "" {
		return false
	}
	isBlocked, err := h.userGRPCClient.CheckBlockStatus(ctx, senderID, otherUserID)
	if err != nil {
		h.log.Warn("block status check failed, allowing message",
			slog.String("error", err.Error()),
			slog.String("sender_id", senderID),
			slog.String("other_user_id", otherUserID),
		)
		return false
	}
	return isBlocked
}

// GetConversations handles GET /v1/conversations.
func (h *OpenAPIHandler) GetConversations(ctx echo.Context) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	convs, err := h.svc.GetConversations(ctx.Request().Context(), userID)
	if err != nil {
		h.log.Error("failed to get conversations", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get conversations"))
	}

	result := make([]api.Conversation, 0, len(convs))
	for _, c := range convs {
		result = append(result, toAPIConversation(c))
	}
	return ctx.JSON(http.StatusOK, result)
}

// CreateConversation handles POST /v1/conversations.
// Checks for an existing conversation between the two users first, then:
//  1. Creates a new conversation if none exists.
//  2. Emits a ConversationCreatedEvent to the Kafka chat-lifecycle topic.
//  3. Broadcasts a room_ready event to the recipient via WebSocket.
func (h *OpenAPIHandler) CreateConversation(ctx echo.Context) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	var req api.CreateConversationRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResp("invalid request body: "+err.Error()))
	}

	if req.UserId == "" {
		return ctx.JSON(http.StatusBadRequest, errResp("user_id is required"))
	}

	// Validate that the other user is not the same as the current user
	if req.UserId == userID {
		return ctx.JSON(http.StatusBadRequest, errResp("cannot create conversation with yourself"))
	}

	// Check if either user has blocked the other before creating a conversation
	if h.isBlocked(ctx.Request().Context(), userID, req.UserId) {
		h.log.Info("conversation creation blocked — user is blocked",
			slog.String("initiator_id", userID),
			slog.String("target_id", req.UserId),
		)
		return ctx.JSON(http.StatusForbidden, errResp("cannot create conversation: user is blocked"))
	}

	// Check if conversation already exists between these two users
	// (checked after block check to avoid exposing existing convos when blocked)
	existing, err := h.svc.FindExistingConversation(ctx.Request().Context(), userID, req.UserId)
	if err != nil {
		h.log.Error("failed to check existing conversation", slog.String("error", err.Error()))
		// Proceed to create anyway — the check is best-effort
	}
	if existing != nil {
		h.log.Info("returning existing conversation",
			slog.String("conversation_id", existing.ID),
		)
		return ctx.JSON(http.StatusOK, toAPIConversation(existing))
	}

	matchID := ""
	if req.MatchId != nil {
		matchID = *req.MatchId
	}

	now := time.Now().UTC()
	conv := &domain.Conversation{
		ID:        scyllarepo.NewUUID(),
		User1ID:   userID,
		User2ID:   req.UserId,
		MatchID:   matchID,
		CreatedAt: now,
	}

	if err := h.svc.CreateConversation(ctx.Request().Context(), conv); err != nil {
		h.log.Error("failed to create conversation", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to create conversation"))
	}

	// Emit ConversationCreatedEvent to the Kafka chat-lifecycle topic
	lifecycleEvent := &domain.ConversationCreatedEvent{
		ConversationID: conv.ID,
		InitiatorID:    userID,
		RecipientID:    req.UserId,
		Timestamp:      now,
	}

	if h.producer != nil {
		rawData, err := json.Marshal(lifecycleEvent)
		if err != nil {
			h.log.Error("failed to marshal ConversationCreatedEvent",
				slog.String("error", err.Error()),
			)
		} else {
			if err := h.producer.PublishToTopic(ctx.Request().Context(), kafkainfra.LifecycleTopic, &kafkainfra.Message{
				RoomID: conv.ID,
				Type:   domain.WSMsgTypeRoomReady,
				Data:   rawData,
			}); err != nil {
				h.log.Warn("failed to emit ConversationCreatedEvent to Kafka",
					slog.String("error", err.Error()),
				)
			} else {
				h.log.Info("emitted ConversationCreatedEvent to Kafka",
					slog.String("conversation_id", conv.ID),
					slog.String("initiator_id", userID),
					slog.String("recipient_id", req.UserId),
					slog.String("topic", kafkainfra.LifecycleTopic),
				)
			}
		}
	}

	// Broadcast room_ready to the recipient if they're globally connected
	h.broadcastRoomReady(req.UserId, conv.ID)

	return ctx.JSON(http.StatusCreated, toAPIConversation(conv))
}

// GetMessages handles GET /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) GetMessages(ctx echo.Context, id string, params api.GetMessagesParams) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	// Verify the user is a participant in this conversation
	conv, err := h.svc.GetConversation(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, errResp("conversation not found"))
	}
	if !h.svc.IsConversationParticipant(ctx.Request().Context(), conv, userID) {
		return ctx.JSON(http.StatusForbidden, errResp("not a participant in this conversation"))
	}

	limit := 50
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
		if limit > 200 {
			limit = 200
		}
	}

	messages, err := h.svc.GetMessages(ctx.Request().Context(), id, limit, 0)
	if err != nil {
		h.log.Error("failed to get messages", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get messages"))
	}

	result := make([]api.Message, 0, len(messages))
	for _, m := range messages {
		result = append(result, toAPIMessage(m))
	}
	return ctx.JSON(http.StatusOK, result)
}

// SendMessage handles POST /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) SendMessage(ctx echo.Context, id string) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	var req api.SendMessageRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResp("invalid request body: " + err.Error()))
	}

	if req.Content == "" {
		return ctx.JSON(http.StatusBadRequest, errResp("content is required"))
	}

	// Verify the conversation exists and user is a participant
	conv, err := h.svc.GetConversation(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, errResp("conversation not found"))
	}
	if !h.svc.IsConversationParticipant(ctx.Request().Context(), conv, userID) {
		return ctx.JSON(http.StatusForbidden, errResp("not a participant in this conversation"))
	}

	if conv.Type == domain.ConversationTypeDirect {
		var otherUserID string
		if conv.User1ID == userID {
			otherUserID = conv.User2ID
		} else {
			otherUserID = conv.User1ID
		}

		if h.isBlocked(ctx.Request().Context(), userID, otherUserID) {
			h.log.Info("REST message blocked — user is blocked",
				slog.String("sender_id", userID),
				slog.String("recipient_id", otherUserID),
				slog.String("conversation_id", id),
			)
			return ctx.JSON(http.StatusForbidden, errResp("message blocked: you cannot message this user"))
		}
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ConversationID: id,
		SenderID:       userID,
		Content:        req.Content,
		CreatedAt:      now,
	}

	if err := h.svc.SendMessage(ctx.Request().Context(), msg); err != nil {
		h.log.Error("failed to save message", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to send message"))
	}

	// Broadcast the message via WebSocket for real-time delivery
	h.broadcastNewMessage(id, msg)

	return ctx.JSON(http.StatusCreated, toAPIMessage(msg))
}

// ─── Group Conversation Endpoints ───────────────────────────────────────

// CreateGroupConversationRequest is the request body for creating a group.
type CreateGroupConversationRequest struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// CreateGroupConversation handles POST /v1/groups.
func (h *OpenAPIHandler) CreateGroupConversation(ctx echo.Context) error {
	userID := h.extractUserID(ctx)
	if userID == "" {
		return ctx.JSON(http.StatusUnauthorized, errResp("missing user identification"))
	}

	var req CreateGroupConversationRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResp("invalid request body: "+err.Error()))
	}

	if req.Name == "" {
		return ctx.JSON(http.StatusBadRequest, errResp("name is required"))
	}
	if len(req.Members) == 0 {
		return ctx.JSON(http.StatusBadRequest, errResp("at least one member is required"))
	}

	// Build member list: include the creator + specified members, deduplicate
	memberSet := make(map[string]bool)
	memberSet[userID] = true
	for _, m := range req.Members {
		if m != "" {
			memberSet[m] = true
		}
	}
	memberIDs := make([]string, 0, len(memberSet))
	for m := range memberSet {
		memberIDs = append(memberIDs, m)
	}

	now := time.Now().UTC()
	conv := &domain.Conversation{
		ID:        scyllarepo.NewUUID(),
		Type:      domain.ConversationTypeGroup,
		Name:      req.Name,
		MemberIDs: memberIDs,
		CreatedAt: now,
	}

	if err := h.svc.CreateGroupConversation(ctx.Request().Context(), conv); err != nil {
		h.log.Error("failed to create group conversation", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to create group"))
	}

	return ctx.JSON(http.StatusCreated, toAPIConversation(conv))
}



// GetPresence handles GET /v1/presence/{userID}.
func (h *OpenAPIHandler) GetPresence(ctx echo.Context, userID string) error {
	presence, err := h.svc.GetPresence(ctx.Request().Context(), userID)
	if err != nil {
		h.log.Error("failed to get presence", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get presence"))
	}

	status := api.PresenceStatus(presence.Status)
	return ctx.JSON(http.StatusOK, api.Presence{
		UserId:   &presence.UserID,
		Status:   &status,
		LastSeen: &presence.LastSeen,
	})
}

// fetchUsersFromUserService calls the user-service API to sync users
// into the local cache when Kafka is unavailable or hasn't delivered yet.
func (h *OpenAPIHandler) fetchUsersFromUserService(ctx context.Context) ([]api.UserInfo, error) {
	url := h.cfg.UserServiceURL + "/v1/users"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// user-service returns a ListUsersResponse with a nested "users" array
	var listResp struct {
		Users *[]struct {
			ID    string  `json:"id"`
			Email string  `json:"email"`
			Profile *struct {
				DisplayName string `json:"display_name"`
				AvatarURL   string `json:"avatar_url"`
			} `json:"profile"`
		} `json:"users"`
		Total *int `json:"total"`
	}

	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, err
	}

	result := make([]api.UserInfo, 0)
	if listResp.Users != nil {
		for _, u := range *listResp.Users {
			displayName := u.ID
			avatarURL := ""
			if u.Profile != nil {
				if u.Profile.DisplayName != "" {
					displayName = u.Profile.DisplayName
				}
				avatarURL = u.Profile.AvatarURL
			}
			result = append(result, api.UserInfo{
				UserId:      &u.ID,
				Email:       &u.Email,
				DisplayName: &displayName,
				AvatarUrl:   &avatarURL,
			})
			// Seed local cache so subsequent requests don't need to re-fetch
			_ = h.userStore.Upsert(ctx, &domain.UserInfo{
				UserID:      u.ID,
				Email:       u.Email,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
			})
		}
		h.log.Info("seeded user cache from user-service API",
			slog.Int("count", len(*listResp.Users)),
		)
	}

	return result, nil
}

// ─── User Info endpoints ──────────────────────────────────────────────

// GetUsers handles GET /v1/users — returns all users from the local cache
// (populated by user-events Kafka topic) with a fallback to the user-service
// API when the cache is empty or Kafka is unavailable.
func (h *OpenAPIHandler) GetUsers(ctx echo.Context) error {
	users, err := h.userStore.List(ctx.Request().Context())
	if err != nil {
		h.log.Error("failed to list users from cache", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to list users"))
	}

	// If cache has data, return it directly (eventual consistency path via Kafka)
	if len(users) > 0 {
		result := make([]api.UserInfo, 0, len(users))
		for _, u := range users {
			result = append(result, toAPIUserInfo(u))
		}
		return ctx.JSON(http.StatusOK, result)
	}

	// Cache is empty — fall back to fetching directly from user-service API
	h.log.Info("user cache empty, falling back to user-service API")
	fallbackUsers, err := h.fetchUsersFromUserService(ctx.Request().Context())
	if err != nil {
		h.log.Error("failed to fetch users from user-service", slog.String("error", err.Error()))
		return ctx.JSON(http.StatusOK, []api.UserInfo{}) // return empty rather than error
	}

	return ctx.JSON(http.StatusOK, fallbackUsers)
}

// GetUserByID handles GET /v1/chat/users/:id — returns a single cached user.
func (h *OpenAPIHandler) GetUserByID(ctx echo.Context, userID string) error {
	user, err := h.userStore.GetByID(ctx.Request().Context(), userID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResp("failed to get user"))
	}
	if user == nil {
		return ctx.JSON(http.StatusNotFound, errResp("user not found"))
	}
	return ctx.JSON(http.StatusOK, toAPIUserInfo(user))
}

// GetUserByIDHandler wraps GetUserByID as an echo.HandlerFunc by extracting
// the user ID from the URL path parameter.
func (h *OpenAPIHandler) GetUserByIDHandler(ctx echo.Context) error {
	userID := ctx.Param("id")
	if userID == "" {
		return ctx.JSON(http.StatusBadRequest, errResp("missing user id"))
	}
	return h.GetUserByID(ctx, userID)
}

func toAPIUserInfo(u *domain.UserInfo) api.UserInfo {
	return api.UserInfo{
		UserId:      &u.UserID,
		Email:       &u.Email,
		DisplayName: &u.DisplayName,
		AvatarUrl:   &u.AvatarURL,
	}
}



// ─── Conversion Helpers ─────────────────────────────────────────────────────

func toAPIConversation(c *domain.Conversation) api.Conversation {
	id := c.ID
	createdAt := c.CreatedAt

	// For group conversations, include name and type
	if c.Type == domain.ConversationTypeGroup {
		convType := string(c.Type)
		return api.Conversation{
			Id:        &id,
			Type:      &convType,
			Name:      &c.Name,
			CreatedAt: &createdAt,
		}
	}

	// Direct conversation
	user1ID := c.User1ID
	user2ID := c.User2ID
	matchID := c.MatchID
	convType := string(c.Type)
	return api.Conversation{
		Id:        &id,
		Type:      &convType,
		User1Id:   &user1ID,
		User2Id:   &user2ID,
		MatchId:   &matchID,
		CreatedAt: &createdAt,
	}
}

func toAPIMessage(m *domain.Message) api.Message {
	id := m.ID
	convID := m.ConversationID
	senderID := m.SenderID
	content := m.Content
	createdAt := m.CreatedAt

	return api.Message{
		Id:             &id,
		ConversationId: &convID,
		SenderId:       &senderID,
		Content:        &content,
		CreatedAt:      &createdAt,
	}
}
