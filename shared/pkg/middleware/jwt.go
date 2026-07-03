package middleware

import (
	"github.com/labstack/echo/v4"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user ID.
	UserIDKey contextKey = "user_id"
)

// ExtractUserID extracts the user ID from the Echo context.
func ExtractUserID(c echo.Context) (string, bool) {
	uid, ok := c.Get(string(UserIDKey)).(string)
	return uid, ok
}

// ParseJWT parses and validates a JWT token, returning the user ID.
// This is a boilerplate — actual implementation depends on Clerk JWKS.
func ParseJWT(tokenString string) (string, error) {
	return "", nil
}

// InjectUserID extracts user ID from the X-User-ID header and stores it in Echo context.
func InjectUserID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		uid := c.Request().Header.Get("X-User-ID")
		if uid != "" {
			c.Set(string(UserIDKey), uid)
		}
		return next(c)
	}
}
