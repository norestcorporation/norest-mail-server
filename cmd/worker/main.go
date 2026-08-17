package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/norest-mail/server/internal/config"
	"github.com/norest-mail/server/internal/db"
	"github.com/norest-mail/server/internal/mail"
	"github.com/norest-mail/server/internal/provisioning"
	"github.com/norest-mail/server/internal/stalwart"
)

func main() {
	// Structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("norest-worker starting")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded", "env", cfg.AppEnv)

	// Connect to PostgreSQL with configured pool settings
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poolCfg := &db.PoolConfig{
		MaxConns:        int32(cfg.DBMaxConns),
		MinConns:        int32(cfg.DBMinConns),
		MaxConnLifetime: time.Duration(cfg.DBMaxConnLifetime) * time.Second,
		MaxConnIdleTime: time.Duration(cfg.DBMaxConnIdleTime) * time.Second,
	}
	pool, err := db.NewPoolWithConfig(ctx, cfg.DatabaseURL, poolCfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Create Stalwart client
	stalwartClient := stalwart.NewClient(
		cfg.StalwartBaseURL,
		cfg.StalwartAdminUser,
		cfg.StalwartAdminPassword,
	)

	// Generate worker identity
	workerID := provisioning.GenerateWorkerID()
	if cfg.WorkerID != "" {
		workerID = cfg.WorkerID
	}

	slog.Info("worker identity", "worker_id", workerID)

	// Create and start the provisioning worker
	worker := provisioning.NewWorker(
		pool,
		stalwartClient,
		workerID,
		cfg.JobLeaseSeconds,
		cfg.JobHeartbeatSeconds,
		cfg.JobMaxAttempts,
		cfg.JobMaxBackoffSeconds,
	)

	// Start worker in goroutine
	errCh := make(chan error, 3) // buffer size 3 since we have three workers now
	go func() {
		errCh <- worker.Run(ctx)
	}()

	outboxProcessor := mail.NewOutboxProcessor(pool)
	go func() {
		errCh <- outboxProcessor.Run(ctx)
	}()

	reconciliationWorker := mail.NewReconciliationWorker(pool, stalwartClient)
	go func() {
		errCh <- reconciliationWorker.Run(ctx)
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig)
		cancel()
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			slog.Error("worker error", "error", err)
			os.Exit(1)
		}
	}

	// Wait for worker to gracefully stop
	slog.Info("waiting for worker to finish processing")
	select {
	case <-time.After(30 * time.Second):
		slog.Warn("worker shutdown timeout, forcing exit")
	case <-errCh:
		slog.Info("worker stopped gracefully")
	}

	// Close database connection
	slog.Info("closing database connection")
	pool.Close()

	slog.Info("norest-worker stopped")
}
