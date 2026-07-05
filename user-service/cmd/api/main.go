package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/user-service/api"
	"github.com/ride-sharing/user-service/internal/config"
	"github.com/ride-sharing/user-service/internal/handler"
	"github.com/ride-sharing/user-service/internal/repository/postgres"
	"github.com/ride-sharing/user-service/internal/service"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	// Repositories
	repo, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer repo.Close()
	logger.Info("connected to database")

	// Services
	svc := service.New(repo, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.JWTSecret)

	// Echo server
	e := echo.New()

	// Register routes via generated oapi-codegen handler
	openapiHandler := handler.NewOpenAPIHandler(svc, svc)
	api.RegisterHandlers(e, openapiHandler)

	addr := ":" + cfg.Port
	logger.Info("starting User Service", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
