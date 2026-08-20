package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AppEnv   string
	HTTPAddr string

	DatabaseURL string

	StalwartBaseURL       string
	StalwartPublicURL     string
	StalwartAdminUser     string
	StalwartAdminPassword string
	StalwartRecoveryAdmin string

	JWTSecret string

	AllowedOrigins []string

	// Worker configuration
	ProvisioningWorkers    int
	WorkerID              string
	JobLeaseSeconds       int
	JobHeartbeatSeconds   int
	JobMaxAttempts        int
	JobMaxBackoffSeconds  int
	MaxConcurrentJobs     int

	// Database connection pool configuration
	DBMaxConns          int
	DBMinConns          int
	DBMaxConnLifetime   int
	DBMaxConnIdleTime   int
	DBOperationTimeout  int

	// Mailer Daemon (bounce notification system)
	MailerDaemonEmail    string
	MailerDaemonName     string
}

// Load reads configuration from environment variables and validates required values.
// It fails fast if any required variable is missing or if production configuration is unsafe.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:                getEnv("APP_ENV", "development"),
		HTTPAddr:              getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		StalwartBaseURL:       os.Getenv("STALWART_BASE_URL"),
		StalwartPublicURL:     os.Getenv("STALWART_PUBLIC_URL"),
		StalwartAdminUser:     os.Getenv("STALWART_ADMIN_USER"),
		StalwartAdminPassword: os.Getenv("STALWART_ADMIN_PASSWORD"),
		StalwartRecoveryAdmin: os.Getenv("STALWART_RECOVERY_ADMIN"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		AllowedOrigins:        parseAllowedOrigins(getEnv("ALLOWED_ORIGINS", "")),
		ProvisioningWorkers:   getEnvInt("PROVISIONING_WORKERS", 4),
		WorkerID:              getEnv("WORKER_ID", ""),
		JobLeaseSeconds:       getEnvInt("JOB_LEASE_SECONDS", 60),
		JobHeartbeatSeconds:   getEnvInt("JOB_HEARTBEAT_SECONDS", 20),
		JobMaxAttempts:        getEnvInt("JOB_MAX_ATTEMPTS", 10),
		JobMaxBackoffSeconds:  getEnvInt("JOB_MAX_BACKOFF_SECONDS", 300),
		MaxConcurrentJobs:     getEnvInt("MAX_CONCURRENT_JOBS", 5),
		DBMaxConns:            getEnvInt("DB_MAX_CONNS", 10),
		DBMinConns:            getEnvInt("DB_MIN_CONNS", 2),
		DBMaxConnLifetime:     getEnvInt("DB_MAX_CONN_LIFETIME", 1800), // 30 minutes in seconds
		DBMaxConnIdleTime:     getEnvInt("DB_MAX_CONN_IDLE_TIME", 300), // 5 minutes in seconds
		DBOperationTimeout:    getEnvInt("DB_OPERATION_TIMEOUT", 30), // 30 seconds
		MailerDaemonEmail:     getEnv("MAILER_DAEMON_EMAIL", "mailer-daemon@norest.in"),
		MailerDaemonName:      getEnv("MAILER_DAEMON_NAME", "Mail Delivery Subsystem"),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":            c.DatabaseURL,
		"STALWART_BASE_URL":       c.StalwartBaseURL,
		"STALWART_ADMIN_USER":     c.StalwartAdminUser,
		"STALWART_ADMIN_PASSWORD": c.StalwartAdminPassword,
		"JWT_SECRET":              c.JWTSecret,
	}

	for name, value := range required {
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", name)
		}
	}

	// Production-specific validation
	if c.AppEnv == "production" {
		if err := c.validateProduction(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) validateProduction() error {
	// Reject development defaults in production
	unsafeDefaults := map[string]string{
		"STALWART_ADMIN_PASSWORD": "change-me-development-only",
		"JWT_SECRET":              "change-me-development-only",
		"DATABASE_URL":            "postgres://norest:norest@",
	}

	for envVar, unsafeValue := range unsafeDefaults {
		value := os.Getenv(envVar)
		if strings.HasPrefix(value, unsafeValue) {
			return fmt.Errorf("production environment cannot use development default for %s", envVar)
		}
	}

	// Require allowed origins in production
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("production environment requires ALLOWED_ORIGINS to be set")
	}

	// Reject wildcard CORS in production
	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("production environment cannot use wildcard CORS origin")
		}
	}

	// Reject development bootstrap in production
	if strings.Contains(c.StalwartRecoveryAdmin, "change-me-development-only") {
		return fmt.Errorf("production environment cannot use development bootstrap credentials")
	}

	return nil
}

func parseAllowedOrigins(value string) []string {
	if value == "" {
		return nil
	}
	origins := strings.Split(value, ",")
	var result []string
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}
