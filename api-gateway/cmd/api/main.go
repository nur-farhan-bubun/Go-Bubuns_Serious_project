package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ride-sharing/api-gateway/internal/config"
	"github.com/ride-sharing/api-gateway/internal/handler"
	ratelimit "github.com/ride-sharing/api-gateway/internal/middleware"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e := echo.New()

	// Global middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-User-ID", "Upgrade", "Connection"},
		AllowCredentials: true,
	}))
	e.Use(ratelimit.RateLimit(cfg))

	// Routes
	handler.RegisterRoutes(e, cfg, logger)

	addr := ":" + cfg.Port
	logger.Info("starting API Gateway", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
