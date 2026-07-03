package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/chat-service/internal/config"
	"github.com/ride-sharing/chat-service/internal/service"
)

// Handler handles HTTP requests for chat.
type Handler struct {
	svc *service.Service
}

// New creates a new chat handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers chat routes on the Echo instance.
func RegisterRoutes(e *echo.Echo, cfg *config.Config) {
	h := New(nil) // TODO: inject service

	// WebSocket endpoint (no version prefix)
	e.GET("/ws", h.HandleWebSocket)

	v1 := e.Group("/v1")
	v1.GET("/conversations", h.GetConversations)
	v1.POST("/conversations", h.CreateConversation)
	v1.GET("/conversations/:id/messages", h.GetMessages)
	v1.POST("/conversations/:id/messages", h.SendMessage)
	v1.GET("/presence/:userID", h.GetPresence)
}

func (h *Handler) GetConversations(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) CreateConversation(c echo.Context) error {
	return c.JSON(201, nil)
}

func (h *Handler) GetMessages(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) SendMessage(c echo.Context) error {
	return c.JSON(201, nil)
}

func (h *Handler) GetPresence(c echo.Context) error {
	return c.JSON(200, nil)
}
