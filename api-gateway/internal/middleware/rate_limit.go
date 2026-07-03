package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
)

// RateLimit returns middleware that rate-limits requests.
func RateLimit(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Implement rate limiting (e.g., token bucket per IP/user)
			return next(c)
		}
	}
}
