package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/user-service/internal/config"
	"github.com/ride-sharing/user-service/internal/handler"
	"github.com/ride-sharing/user-service/internal/repository/postgres"
	"github.com/ride-sharing/user-service/internal/service"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Repositories
	repo := postgres.New()

	// Services
	svc := service.New(repo, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.JWTSecret)

	// Handlers
	userHandler := handler.New(svc)
	authHandler := handler.NewAuthHandler(svc)

	// Echo server
	e := echo.New()

	// Register routes
	handler.RegisterRoutes(e, userHandler)
	handler.RegisterAuthRoutes(e, authHandler)

	addr := ":" + cfg.Port
	logger.Info("starting User Service", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
