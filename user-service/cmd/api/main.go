package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/shared/proto/user"
	"github.com/ride-sharing/user-service/api"
	"github.com/ride-sharing/user-service/internal/config"
	"github.com/ride-sharing/user-service/internal/handler"
	kafkainfra "github.com/ride-sharing/user-service/internal/kafka"
	"github.com/ride-sharing/user-service/internal/repository/postgres"
	"github.com/ride-sharing/user-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	blockRepo := postgres.NewBlockRepository(pool)

	// Initialise Kafka producer for user lifecycle events.
	// If Kafka is unavailable, the service still works — events are dropped.
	eventProducer := kafkainfra.NewProducer(cfg.KafkaBrokers, logger)

	// Services
	svc := service.New(repo, profileRepo, datingRepo, workerRepo, photoRepo, blockRepo,
		cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL, cfg.JWTSecret,
		eventProducer,
	)

	// Echo server
	e := echo.New()

	// Register routes via generated oapi-codegen handler
	openapiHandler := handler.NewOpenAPIHandler(svc, svc)
	api.RegisterHandlers(e, openapiHandler)

	grpcAddr := ":" + cfg.GRPCPort
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("failed to listen for gRPC", slog.String("error", err.Error()))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	grpcHandler := handler.NewGrpcHandler(svc)
	user.RegisterUserServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

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
		logger.Info("starting User Service HTTP", slog.String("addr", addr))
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Error("server start failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Start gRPC server in background.
	go func() {
		logger.Info("starting User Service gRPC", slog.String("addr", grpcAddr))
		if err := grpcServer.Serve(grpcLis); err != nil {
			logger.Error("gRPC server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for OS interrupt or termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutting down server", slog.String("signal", sig.String()))

	// 1. Gracefully stop gRPC server.
	grpcServer.GracefulStop()

	// 2. Gracefully shut down the HTTP server with a timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
	}

	// 3. Flush and close Kafka producer on shutdown.
	if err := eventProducer.Close(); err != nil {
		logger.Error("kafka producer close error", "error", err)
	}

	logger.Info("server stopped")
}
