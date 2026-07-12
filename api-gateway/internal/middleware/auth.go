package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
)

// contextKey is used for storing values in Echo context.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user ID.
	UserIDKey contextKey = "user_id"
)

// AuthMiddleware validates JWT tokens (Clerk-compatible) and injects
// X-User-ID header into downstream proxied requests.
//
// Supports:
//   - RS256 (Clerk default) — expects PEM-encoded public key in cfg.ClerkJWTKey
//   - HS256 (development) — expects a shared secret in cfg.ClerkJWTKey
//
// The middleware automatically detects the signing method from the token header.
func AuthMiddleware(cfg *config.Config, logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing authorization header",
				})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid authorization format — expected 'Bearer <token>'",
				})
			}

			claims, err := parseAndValidateToken(tokenString, cfg)
			if err != nil {
				logger.Warn("jwt validation failed",
					slog.String("error", err.Error()),
					slog.String("token_prefix", safeTokenPrefix(tokenString)),
				)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid or expired token",
				})
			}

			// Extract user ID from the standard "sub" claim.
			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				logger.Warn("jwt missing sub claim",
					slog.Any("claims", claims),
				)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "token missing subject (user ID)",
				})
			}

			// Inject into downstream headers so proxied services can identify the user.
			c.Request().Header.Set("X-User-ID", userID)

			// Also store in Echo context for handlers within the gateway itself.
			c.Set(string(UserIDKey), userID)

			return next(c)
		}
	}
}

// parseAndValidateToken parses and validates a JWT token string.
// Returns the token claims or an error.
func parseAndValidateToken(tokenString string, cfg *config.Config) (jwt.MapClaims, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return resolveKey(t, cfg)
	}

	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256", "HS256"}),
	}
	if cfg.ClerkJWTIssuer != "" {
		opts = append(opts, jwt.WithIssuer(cfg.ClerkJWTIssuer))
	}
	token, err := jwt.Parse(tokenString, keyFunc, opts...)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// resolveKey returns the appropriate verification key based on the token's
// signing method header.
func resolveKey(token *jwt.Token, cfg *config.Config) (interface{}, error) {
	if cfg.ClerkJWTKey == "" {
		return nil, fmt.Errorf("CLERK_JWT_KEY not configured")
	}

	switch token.Method.(type) {
	case *jwt.SigningMethodRSA:
		key, err := parseRSAPublicKey(cfg.ClerkJWTKey)
		if err != nil {
			return nil, fmt.Errorf("parse rsa key: %w", err)
		}
		return key, nil

	case *jwt.SigningMethodHMAC:
		return []byte(cfg.ClerkJWTKey), nil

	default:
		return nil, fmt.Errorf("unsupported signing method: %v", token.Method)
	}
}

// parseRSAPublicKey parses a PEM-encoded RSA public key.
func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse pkix: %w", err)
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaKey, nil
}

// safeTokenPrefix returns a truncated representation of a token for logging.
func safeTokenPrefix(token string) string {
	if len(token) > 6 {
		return token[:6] + "..."
	}
	return token
}

// ExtractUserID extracts the authenticated user ID from Echo context.
func ExtractUserID(c echo.Context) (string, bool) {
	uid, ok := c.Get(string(UserIDKey)).(string)
	return uid, ok
}

// DevAuthMiddleware extracts the user ID from a JWT token without signature
// verification. Used in development mode (CLERK_JWT_KEY empty) so the frontend
// can pass its self-signed JWT and the gateway will set the X-User-ID header
// for downstream services.
//
// This is safe for dev only: it merely base64-decodes the JWT payload without
// verifying the signature. In production, AuthMiddleware performs full
// validation before setting the header.
func DevAuthMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				// No token — skip (let downstream handle it)
				return next(c)
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return next(c)
			}

			// Extract user ID from JWT payload without verification
			userID := extractSubFromJWT(tokenString)
			if userID != "" {
				c.Request().Header.Set("X-User-ID", userID)
			}

			return next(c)
		}
	}
}

// extractSubFromJWT splits a JWT into parts and extracts the `sub` claim from
// the payload (middle segment) using only base64 decoding — no signature
// verification. This is safe only for development mode.
func extractSubFromJWT(tokenString string) string {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode base64url payload (RFC 4648 §5) — no padding needed
	decoded, err := jwt.NewParser().DecodeSegment(parts[1])
	if err != nil {
		return ""
	}

	// Parse JSON and extract sub claim
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}

	return claims.Sub
}

// WebSocketAuthMiddleware validates JWT tokens passed as query params
// for WebSocket connections (which cannot set custom headers).
func WebSocketAuthMiddleware(cfg *config.Config, logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// WebSocket token comes as a query parameter
			tokenString := c.QueryParam("token")
			if tokenString == "" {
				logger.Warn("ws auth: missing token query param")
				return c.String(http.StatusUnauthorized, "missing token parameter")
			}

			claims, err := parseAndValidateToken(tokenString, cfg)
			if err != nil {
				logger.Warn("ws auth: jwt validation failed",
					slog.String("error", err.Error()),
				)
				return c.String(http.StatusUnauthorized, "invalid or expired token")
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				logger.Warn("ws auth: token missing sub claim")
				return c.String(http.StatusUnauthorized, "token missing user ID")
			}

			// Override the user_id query param with the authenticated user ID
			// so the chat service can trust it.
			q := c.Request().URL.Query()
			q.Set("user_id", userID)
			c.Request().URL.RawQuery = q.Encode()

			return next(c)
		}
	}
}
