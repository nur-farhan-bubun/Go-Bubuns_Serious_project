package handler

import (
	"github.com/labstack/echo/v4"
)

// HandleWebSocket upgrades the HTTP connection to a WebSocket connection.
func (h *Handler) HandleWebSocket(c echo.Context) error {
	// TODO: upgrade to WebSocket and register with hub
	return c.String(200, "websocket endpoint")
}
