package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/chat-service/api"
	"github.com/ride-sharing/chat-service/internal/config"
	"github.com/ride-sharing/chat-service/internal/handler"
	kafkainfra "github.com/ride-sharing/chat-service/internal/kafka"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Create the sharded WebSocket hub.
	hub := ws.NewHub(logger)

	// Initialise the Kafka producer for publishing chat events.
	producer := kafkainfra.NewProducer(cfg.KafkaBrokers, logger)

	// Initialise the Kafka consumer that fans out messages to local clients.
	consumer := kafkainfra.NewConsumer(cfg.KafkaBrokers, hub, logger)

	e := echo.New()

	// Register routes via generated oapi-codegen handler
	openapiHandler := handler.NewOpenAPIHandler(cfg, hub, producer, logger)
	api.RegisterHandlers(e, openapiHandler)

	// WebSocket endpoint (not part of OpenAPI spec)
	e.GET("/ws", openapiHandler.HandleWebSocket)

	addr := ":" + cfg.Port

	// Start HTTP server in background.
	go func() {
		logger.Info("starting Chat Service", slog.String("addr", addr))
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Error("server start failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Start Kafka consumer in background. It blocks until ctx is cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer.Run(ctx)

	// Wait for OS interrupt or termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutting down server", slog.String("signal", sig.String()))

	// 1. Cancel the consumer context so the read loop exits.
	cancel()

	// 2. Drain all WebSocket connections first.
	hub.Shutdown()

	// 3. Gracefully shut down the HTTP server with a timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	// 4. Flush and close Kafka producer (pending batched messages).
	if err := producer.Close(); err != nil {
		logger.Error("kafka producer close error", slog.String("error", err.Error()))
	}

	// 5. Close Kafka consumer (commits offsets, leaves group).
	if err := consumer.Close(); err != nil {
		logger.Error("kafka consumer close error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
