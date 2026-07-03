package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/internal/config"
	"github.com/ride-sharing/api-gateway/internal/handler"
	"github.com/ride-sharing/api-gateway/internal/middleware"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e := echo.New()

	// Global middleware
	e.Use(middleware.RateLimit(cfg))

	// Routes
	handler.RegisterRoutes(e, cfg)

	addr := ":" + cfg.Port
	logger.Info("starting API Gateway", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
