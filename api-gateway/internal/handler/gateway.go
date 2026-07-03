package handler

import (
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
)

// RegisterRoutes sets up all routes on the Echo instance.
func RegisterRoutes(e *echo.Echo, cfg *config.Config) {
	// Public
	e.GET("/health", healthCheck)

	// Auth routes (public — no JWT required)
	auth := e.Group("/v1/auth")
	auth.Any("/*", proxyTo(cfg.UserServiceURL))

	// Protected
	api := e.Group("/v1")
	// TODO: Add auth middleware once Clerk JWT validation is implemented

	api.Any("/users/*", proxyTo(cfg.UserServiceURL))
	api.Any("/me/*", proxyTo(cfg.UserServiceURL))
	api.Any("/swipe", proxyTo(cfg.MatchServiceURL))
	api.Any("/matches/*", proxyTo(cfg.MatchServiceURL))
	api.Any("/discover", proxyTo(cfg.MatchServiceURL))
	api.Any("/messages/*", proxyTo(cfg.ChatServiceURL))
	api.Any("/ws", proxyTo(cfg.ChatServiceURL))
	api.Any("/location/*", proxyTo(cfg.LocationServiceURL))
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
