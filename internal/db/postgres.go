package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds configurable database connection pool settings.
type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultPoolConfig returns sensible defaults for development.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	}
}

// NewPool creates a new PostgreSQL connection pool from the given DSN.
// It validates connectivity before returning.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return NewPoolWithConfig(ctx, dsn, DefaultPoolConfig())
}

// NewPoolWithConfig creates a new PostgreSQL connection pool with custom configuration.
func NewPoolWithConfig(ctx context.Context, dsn string, cfg *PoolConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = cfg.MaxConnLifetime
	config.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Enable health checks on connections in the pool
	config.HealthCheckPeriod = 1 * time.Minute
	
	// Configure connection retries for better resilience
	config.MaxConnLifetimeJitter = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	slog.Info("database connection established",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
		"max_conn_lifetime", cfg.MaxConnLifetime,
		"max_conn_idle_time", cfg.MaxConnIdleTime,
	)
	return pool, nil
}

// HealthCheck performs a lightweight database connectivity check.
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var result int
	err := pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// WithTimeout creates a context with a timeout for database operations.
// It returns a context that will be cancelled after the specified duration.
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithDefaultTimeout creates a context with a default 30-second timeout for database operations.
func WithDefaultTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, 30*time.Second)
}
