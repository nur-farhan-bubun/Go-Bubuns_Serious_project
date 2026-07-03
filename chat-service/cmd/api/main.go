package main

import (
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/chat-service/internal/config"
	"github.com/ride-sharing/chat-service/internal/handler"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	e := echo.New()

	handler.RegisterRoutes(e, cfg)

	addr := ":" + cfg.Port
	logger.Info("starting Chat Service", "addr", addr)
	e.Logger.Fatal(e.Start(addr))
}
