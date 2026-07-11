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
	"github.com/ride-sharing/user-service/api"
	"github.com/ride-sharing/user-service/internal/config"
	"github.com/ride-sharing/user-service/internal/handler"
	kafkainfra "github.com/ride-sharing/user-service/internal/kafka"
	"github.com/ride-sharing/user-service/internal/repository/postgres"
	"github.com/ride-sharing/user-service/internal/service"
)

const shutdownTimeout = 10 * time.Second

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

	pool := repo.Pool() // reuse the same connection pool for all profile repos

	profileRepo := postgres.NewProfileRepository(pool)
	datingRepo := postgres.NewDatingProfileRepository(pool)
	workerRepo := postgres.NewWorkerProfileRepository(pool)
	photoRepo := postgres.NewProfilePhotoRepository(pool)

	// Initialise Kafka producer for user lifecycle events.
	// If Kafka is unavailable, the service still works — events are dropped.
	eventProducer := kafkainfra.NewProducer(cfg.KafkaBrokers, logger)

	// Services
	svc := service.New(repo, profileRepo, datingRepo, workerRepo, photoRepo,
		cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.JWTSecret,
		eventProducer,
	)

	// Echo server
	e := echo.New()

	// Register routes via generated oapi-codegen handler
	openapiHandler := handler.NewOpenAPIHandler(svc, svc)
	api.RegisterHandlers(e, openapiHandler)

	addr := ":" + cfg.Port

	// Backfill existing users to Kafka for eventual consistency sync.
	// This runs in the background so it doesn't block the HTTP server startup.
	go func() {
		if err := svc.Backfill(context.Background()); err != nil {
			logger.Warn("backfill failed, users may not appear in chat-service",
				slog.String("error", err.Error()),
			)
		}
	}()

	// Start HTTP server in background.
	go func() {
		logger.Info("starting User Service", slog.String("addr", addr))
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Error("server start failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for OS interrupt or termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutting down server", slog.String("signal", sig.String()))

	// Gracefully shut down the HTTP server with a timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	// Flush and close Kafka producer on shutdown.
	if err := eventProducer.Close(); err != nil {
		logger.Error("kafka producer close error", "error", err)
	}

	logger.Info("server stopped")
}
