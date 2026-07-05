package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/chat-service/api"
	"github.com/ride-sharing/chat-service/internal/config"
)

// Compile-time check that OpenAPIHandler implements api.ServerInterface.
var _ api.ServerInterface = (*OpenAPIHandler)(nil)

// OpenAPIHandler implements the generated api.ServerInterface.
// Currently delegates to simple stubs matching the original handler behavior.
type OpenAPIHandler struct {
	cfg *config.Config
}

// NewOpenAPIHandler creates a new handler satisfying the generated ServerInterface.
func NewOpenAPIHandler(cfg *config.Config) *OpenAPIHandler {
	return &OpenAPIHandler{cfg: cfg}
}

// GetConversations handles GET /v1/conversations.
func (h *OpenAPIHandler) GetConversations(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// CreateConversation handles POST /v1/conversations.
func (h *OpenAPIHandler) CreateConversation(ctx echo.Context) error {
	return ctx.JSON(http.StatusCreated, nil)
}

// GetMessages handles GET /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) GetMessages(ctx echo.Context, id string, params api.GetMessagesParams) error {
	return ctx.JSON(http.StatusOK, nil)
}

// SendMessage handles POST /v1/conversations/{id}/messages.
func (h *OpenAPIHandler) SendMessage(ctx echo.Context, id string) error {
	return ctx.JSON(http.StatusCreated, nil)
}

// GetPresence handles GET /v1/presence/{userID}.
func (h *OpenAPIHandler) GetPresence(ctx echo.Context, userID string) error {
	return ctx.JSON(http.StatusOK, nil)
}

// HandleWebSocket handles the /ws WebSocket endpoint.
func (h *OpenAPIHandler) HandleWebSocket(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "websocket endpoint")
}
