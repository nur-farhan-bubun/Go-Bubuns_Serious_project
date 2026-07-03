package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/location-service/internal/config"
	"github.com/ride-sharing/location-service/internal/service"
)

// Handler handles HTTP requests for locations.
type Handler struct {
	svc *service.Service
}

// New creates a new location handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers location routes on the Echo instance.
func RegisterRoutes(e *echo.Echo, cfg *config.Config) {
	h := New(nil) // TODO: inject service

	v1 := e.Group("/v1")
	v1.PUT("/location", h.UpdateLocation)
	v1.GET("/location/nearby", h.GetNearby)
	v1.POST("/presence/online", h.SetOnline)
	v1.POST("/presence/offline", h.SetOffline)
}

func (h *Handler) UpdateLocation(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) GetNearby(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) SetOnline(c echo.Context) error {
	return c.JSON(200, nil)
}

func (h *Handler) SetOffline(c echo.Context) error {
	return c.JSON(200, nil)
}
