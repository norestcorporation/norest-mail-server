package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/norest-mail/server/internal/config"
	"github.com/norest-mail/server/internal/db"
	httpserver "github.com/norest-mail/server/internal/http"
	"github.com/norest-mail/server/internal/stalwart"
)

func main() {
	// Structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("norest-api starting")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded", "env", cfg.AppEnv, "addr", cfg.HTTPAddr)

	// Connect to PostgreSQL with configured pool settings
	ctx := context.Background()
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

	// Create HTTP router
	router := httpserver.NewRouter(cfg, pool, stalwartClient)

	// Create HTTP server with hardening
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadTimeout:        15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutdown signal received", "signal", sig)

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop accepting new connections
	slog.Info("stopping http server, draining in-flight requests")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
		os.Exit(1)
	}

	// Close database connection
	slog.Info("closing database connection")
	pool.Close()

	slog.Info("norest-api stopped")
}
