package handler

import (
	"log/slog"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
	"github.com/ride-sharing/api-gateway/internal/middleware"
)

// RegisterRoutes sets up all routes on the Echo instance.
func RegisterRoutes(e *echo.Echo, cfg *config.Config, logger *slog.Logger) {
	// Public
	e.GET("/health", healthCheck)

	// Swagger UI documentation
	RegisterDocsRoutes(e)

	// Auth routes (public — no JWT required)
	auth := e.Group("/v1/auth")
	auth.Any("/*", proxyTo(cfg.UserServiceURL))

	// Protected — only require JWT when Clerk key is configured (production)
	// In development (CLERK_JWT_KEY empty), auth is bypassed so simulated logins work.
	var api *echo.Group
	if cfg.ClerkJWTKey != "" {
		api = e.Group("/v1", middleware.AuthMiddleware(cfg, logger))
		// WebSocket — validate JWT via token query param since headers can't be set
		e.Any("/ws", proxyTo(cfg.ChatServiceURL), middleware.WebSocketAuthMiddleware(cfg, logger))
	} else {
		// Dev mode: extract user ID from JWT without signature verification
		// so the frontend's self-signed dev tokens set the X-User-ID header.
		api = e.Group("/v1", middleware.DevAuthMiddleware(logger))
		e.Any("/ws", proxyTo(cfg.ChatServiceURL))
	}

	// User Service — exact paths + wildcard sub-paths
	api.Any("/users", proxyTo(cfg.UserServiceURL))
	api.Any("/users/*", proxyTo(cfg.UserServiceURL))
	api.Any("/me", proxyTo(cfg.UserServiceURL))
	api.Any("/me/*", proxyTo(cfg.UserServiceURL))

	// Match Service — exact paths + wildcard sub-paths
	api.Any("/swipe", proxyTo(cfg.MatchServiceURL))
	api.Any("/matches", proxyTo(cfg.MatchServiceURL))
	api.Any("/matches/*", proxyTo(cfg.MatchServiceURL))
	api.Any("/discover", proxyTo(cfg.MatchServiceURL))

	// Chat Service — REST + WebSocket
	api.Any("/conversations", proxyTo(cfg.ChatServiceURL))
	api.Any("/conversations/*", proxyTo(cfg.ChatServiceURL))
	api.Any("/messages", proxyTo(cfg.ChatServiceURL))
	api.Any("/messages/*", proxyTo(cfg.ChatServiceURL))
	api.Any("/presence", proxyTo(cfg.ChatServiceURL))
	api.Any("/presence/*", proxyTo(cfg.ChatServiceURL))


	// Location Service — exact paths + wildcard sub-paths
	api.Any("/location", proxyTo(cfg.LocationServiceURL))
	api.Any("/location/*", proxyTo(cfg.LocationServiceURL))

	// Map Posts — proxied to location service (PostGIS-backed)
	api.Any("/posts", proxyTo(cfg.LocationServiceURL))
	api.Any("/posts/*", proxyTo(cfg.LocationServiceURL))
}

func healthCheck(c echo.Context) error {
	return c.String(200, "ok")
}

func proxyTo(target string) echo.HandlerFunc {
	return func(c echo.Context) error {
		remote, err := url.Parse(target)
		if err != nil {
			return err
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
