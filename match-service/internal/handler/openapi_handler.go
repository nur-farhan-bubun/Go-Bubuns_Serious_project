package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/match-service/api"
	"github.com/ride-sharing/match-service/internal/config"
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

// Swipe handles POST /v1/swipe.
func (h *OpenAPIHandler) Swipe(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// GetMatches handles GET /v1/matches.
func (h *OpenAPIHandler) GetMatches(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// GetDiscover handles GET /v1/discover.
func (h *OpenAPIHandler) GetDiscover(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, nil)
}

// GetMatchById handles GET /v1/internal/matches/{id}.
func (h *OpenAPIHandler) GetMatchById(ctx echo.Context, id string) error {
	return ctx.JSON(http.StatusOK, nil)
}
