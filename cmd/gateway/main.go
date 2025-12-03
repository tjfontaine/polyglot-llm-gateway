package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/api/middleware/tracer"
	"github.com/tjfontaine/polyglot-llm-gateway/internal/registration"
	"github.com/tjfontaine/polyglot-llm-gateway/pkg/gateway"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Initialize OpenTelemetry
	shutdown, err := tracer.InitTracer("polyglot-llm-gateway", logger)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			logger.Error("failed to shutdown tracer", slog.String("error", err.Error()))
		}
	}()

	// Register built-in providers and frontdoors
	registration.RegisterBuiltins()

	// Create gateway with default configuration
	// This uses:
	// - File-based config with hot-reload
	// - API key authentication
	// - SQLite storage
	// - Direct event publishing (no external bus)
	// - Basic quality policy (no rate limiting)
	gw, err := gateway.New(
		gateway.WithFileConfig("config.yaml"),
		gateway.WithAPIKeyAuth(),
		gateway.WithSQLite("./data/gateway.db"),
		gateway.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}

	// Start gateway
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := gw.Start(ctx); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}

	logger.Info("Gateway started successfully")
	logger.Info("Features enabled:")
	logger.Info("  - Config: file-based with hot-reload (config.yaml)")
	logger.Info("  - Auth: API key authentication")
	logger.Info("  - Storage: SQLite (./data/gateway.db)")
	logger.Info("  - Events: Direct to storage")
	logger.Info("  - Policy: No rate limiting")
	logger.Info("  - Control Plane: /admin")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received, stopping gateway...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := gw.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Gateway shutdown complete")
}
