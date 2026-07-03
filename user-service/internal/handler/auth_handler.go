package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/user-service/internal/domain"
)

// AuthService defines the service contract used by the auth handler.
type AuthService interface {
	GenerateGoogleLoginURL(state string) string
	HandleGoogleCallback(ctx context.Context, code string) (*domain.AuthResponse, error)
}

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	svc AuthService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// RegisterAuthRoutes registers auth routes on the Echo instance.
func RegisterAuthRoutes(e *echo.Echo, h *AuthHandler) {
	auth := e.Group("/v1/auth")
	auth.GET("/google/login", h.GoogleLogin)
	auth.GET("/google/callback", h.GoogleCallback)
}

// GoogleLogin redirects the user to Google's OAuth consent screen.
func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	state := generateStateToken()
	// TODO: store state token in session/redis to validate on callback
	url := h.svc.GenerateGoogleLoginURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles the OAuth callback from Google.
func (h *AuthHandler) GoogleCallback(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing authorization code"})
	}
	if state == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing state parameter"})
	}

	// TODO: validate state token matches what we stored in GoogleLogin

	authResp, err := h.svc.HandleGoogleCallback(c.Request().Context(), code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "authentication failed: " + err.Error()})
	}

	return c.JSON(http.StatusOK, authResp)
}

func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
