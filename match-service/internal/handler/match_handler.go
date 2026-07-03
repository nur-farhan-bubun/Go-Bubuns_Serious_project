package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/match-service/internal/config"
	"github.com/ride-sharing/match-service/internal/service"
)

// Handler handles HTTP requests for matches.
type Handler struct {
	svc *service.Service
}

// New creates a new match handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers match routes on the Echo instance.
func RegisterRoutes(e *echo.Echo, cfg *config.Config) {
	h := New(nil) // TODO: inject service with real repository

	v1 := e.Group("/v1")
	v1.POST("/swipe", h.Swipe)
	v1.GET("/matches", h.GetMatches)
	v1.GET("/discover", h.GetDiscover)

	// Internal endpoints for service-to-service calls
	internal := e.Group("/v1/internal")
	internal.GET("/matches/:id", h.GetByID)
}

func (h *Handler) Swipe(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) GetMatches(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) GetDiscover(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) GetByID(c echo.Context) error {
	return c.JSON(200, nil)
}
