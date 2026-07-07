package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/location-service/api"
	"github.com/ride-sharing/location-service/internal/config"
	"github.com/ride-sharing/location-service/internal/handler"
	redisrepo "github.com/ride-sharing/location-service/internal/repository/redis"
	"github.com/ride-sharing/location-service/internal/repository/postgres"
	"github.com/ride-sharing/location-service/internal/service"
	ws "github.com/ride-sharing/location-service/internal/websocket"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// ─── Redis (real-time location) ──────────────────────────────────
	locationRepo, err := redisrepo.NewLocationRepository(cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer locationRepo.Close()

	// ─── PostgreSQL + PostGIS (map posts) ────────────────────────────
	ctx := context.Background()
	mapPostRepo, err := postgres.NewMapPostRepository(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to PostGIS database", "error", err)
		os.Exit(1)
	}
	defer mapPostRepo.Close()

	// ─── Service Layer ───────────────────────────────────────────────
	svc := service.New(locationRepo, mapPostRepo)

	// ─── WebSocket Hub ───────────────────────────────────────────────
	wsHub := ws.NewHub(logger)

	// Register the hub callback so location updates are broadcast in real-time
	svc.OnLocationUpdate(wsHub.BroadcastLocation)

	// ─── HTTP Server (Echo) ─────────────────────────────────────────
	e := echo.New()

	
	openapiHandler := handler.NewOpenAPIHandler(cfg, svc)
	api.RegisterHandlers(e, openapiHandler)


	e.GET("/v1/location/ws", func(c echo.Context) error {
		wsHub.HandleWebSocket(c.Response().Writer, c.Request())
		return nil
	})

	addr := ":" + cfg.Port
	logger.Info("starting Location Service", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
