package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/location-service/api"
	"github.com/ride-sharing/location-service/internal/config"
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

// UpdateLocation handles PUT /v1/location.
func (h *OpenAPIHandler) UpdateLocation(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// GetNearby handles GET /v1/location/nearby.
func (h *OpenAPIHandler) GetNearby(ctx echo.Context, params api.GetNearbyParams) error {
	return ctx.JSON(http.StatusOK, nil)
}

// SetOnline handles POST /v1/presence/online.
func (h *OpenAPIHandler) SetOnline(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// SetOffline handles POST /v1/presence/offline.
func (h *OpenAPIHandler) SetOffline(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}
