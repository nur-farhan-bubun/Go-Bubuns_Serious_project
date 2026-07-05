package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/location-service/api"
	"github.com/ride-sharing/location-service/internal/config"
	"github.com/ride-sharing/location-service/internal/handler"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e := echo.New()

	// Register routes via generated oapi-codegen handler
	openapiHandler := handler.NewOpenAPIHandler(cfg)
	api.RegisterHandlers(e, openapiHandler)

	addr := ":" + cfg.Port
	logger.Info("starting Location Service", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
