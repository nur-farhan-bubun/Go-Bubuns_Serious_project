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
	scyllarepo "github.com/ride-sharing/chat-service/internal/repository/scylladb"
	redisrepo "github.com/ride-sharing/chat-service/internal/repository/redis"
	"github.com/ride-sharing/chat-service/internal/service"
	userGRPCClient "github.com/ride-sharing/chat-service/internal/userclient"
	"github.com/ride-sharing/chat-service/internal/store"
	ws "github.com/ride-sharing/chat-service/internal/websocket"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	// Connect to ScyllaDB
	chatRepo, err := scyllarepo.New(ctx, cfg.ScyllaURL, logger)
	if err != nil {
		logger.Error("failed to connect to ScyllaDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer chatRepo.Close()
	logger.Info("connected to ScyllaDB", slog.String("url", cfg.ScyllaURL))

	// Connect to Redis for presence
	presenceRepo, err := redisrepo.NewPresenceRepository(ctx, cfg.RedisURL)
	if err != nil {
		logger.Warn("failed to connect to Redis, presence will be unavailable",
			slog.String("error", err.Error()),
		)
		presenceRepo = nil
	} else {
		defer presenceRepo.Close()
		logger.Info("connected to Redis")
	}

	// Create the chat service
	svc := service.New(chatRepo, presenceRepo)

	// Create the sharded WebSocket hub.
	hub := ws.NewHub(logger)

	// Initialise the in-memory user store for eventual consistency.
	userStore := store.NewMemoryUserStore()

	// Initialise the Kafka producer for publishing chat events.
	producer := kafkainfra.NewProducer(cfg.KafkaBrokers, logger)

	// Initialise the Kafka consumer that fans out messages to local clients.
	consumer := kafkainfra.NewConsumer(cfg.KafkaBrokers, hub, logger)

	// Initialise the Kafka consumer for user lifecycle events (user.created, etc.).
	userConsumer := kafkainfra.NewUserConsumer(cfg.KafkaBrokers, userStore, logger)

	var grpcUserClient *userGRPCClient.Client
	if cfg.UserServiceGRPC != "" {
		var err error
		grpcUserClient, err = userGRPCClient.NewClient(cfg.UserServiceGRPC)
		if err != nil {
			logger.Warn("failed to connect to user-service gRPC, block checks disabled",
				slog.String("error", err.Error()),
			)
			grpcUserClient = nil
		} else {
			defer grpcUserClient.Close()
			logger.Info("connected to user-service gRPC", slog.String("addr", cfg.UserServiceGRPC))
		}
	}

	e := echo.New()

	// Register routes via generated oapi-codegen handler
	openapiHandler := handler.NewOpenAPIHandler(cfg, svc, hub, producer, logger, userStore, grpcUserClient)

	// Wire up presence broadcasting: when a user connects/disconnects, the hub
	// calls this callback which broadcasts to all local clients + publishes to Kafka.
	hub.OnPresenceChange = openapiHandler.PresenceBroadcast

	api.RegisterHandlers(e, openapiHandler)

	// User info routes (not in the OpenAPI spec — added manually)
	// Note: /v1/chat/users prefix to avoid conflict with the API gateway's /v1/users proxy to user-service.
	e.GET("/v1/chat/users", openapiHandler.GetUsers)
	e.GET("/v1/chat/users/:id", openapiHandler.GetUserByIDHandler)

	// Group conversation endpoints (not in the OpenAPI spec — added manually)
	e.POST("/v1/groups", openapiHandler.CreateGroupConversation)

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

	// Start Kafka consumers in background. They block until ctx is cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer.Run(ctx)
	go userConsumer.Run(ctx)

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

	// 5. Close Kafka consumers (commit offsets, leave groups).
	if err := consumer.Close(); err != nil {
		logger.Error("kafka consumer close error", slog.String("error", err.Error()))
	}
	if err := userConsumer.Close(); err != nil {
		logger.Error("user kafka consumer close error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
