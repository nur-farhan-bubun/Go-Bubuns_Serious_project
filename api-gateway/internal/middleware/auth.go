package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
)

// AuthMiddleware validates Clerk JWT tokens and injects X-User-ID header.
func AuthMiddleware(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}
			// TODO: Validate Clerk JWT using cfg.ClerkJWTIssuer and cfg.ClerkJWTKey
			// Extract user ID and set as X-User-ID header
			c.Request().Header.Set("X-User-ID", "placeholder-user-id")
			return next(c)
		}
	}
}
