package config

import (
	"os"
	"testing"
)

func TestLoad_MissingRequired(t *testing.T) {
	// Clear all required env vars
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("STALWART_BASE_URL")
	os.Unsetenv("STALWART_ADMIN_USER")
	os.Unsetenv("STALWART_ADMIN_PASSWORD")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when required env vars are missing")
	}
}

func TestLoad_AllPresent(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("STALWART_BASE_URL", "http://localhost:8080")
	t.Setenv("STALWART_ADMIN_USER", "admin")
	t.Setenv("STALWART_ADMIN_PASSWORD", "password")
	t.Setenv("JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test?sslmode=disable" {
		t.Errorf("unexpected DATABASE_URL: %s", cfg.DatabaseURL)
	}
	if cfg.StalwartBaseURL != "http://localhost:8080" {
		t.Errorf("unexpected STALWART_BASE_URL: %s", cfg.StalwartBaseURL)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("unexpected APP_ENV default: %s", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("unexpected HTTP_ADDR default: %s", cfg.HTTPAddr)
	}
}

func TestLoad_ProductionRejectsDevelopmentDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("STALWART_BASE_URL", "http://localhost:8080")
	t.Setenv("STALWART_ADMIN_USER", "admin")
	t.Setenv("STALWART_ADMIN_PASSWORD", "change-me-development-only")
	t.Setenv("JWT_SECRET", "secure-secret")
	t.Setenv("APP_ENV", "production")
	t.Setenv("ALLOWED_ORIGINS", "https://example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when production uses development password default")
	}
}

func TestLoad_ProductionRejectsWildcardCORS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("STALWART_BASE_URL", "http://localhost:8080")
	t.Setenv("STALWART_ADMIN_USER", "admin")
	t.Setenv("STALWART_ADMIN_PASSWORD", "secure-password")
	t.Setenv("JWT_SECRET", "secure-secret")
	t.Setenv("APP_ENV", "production")
	t.Setenv("ALLOWED_ORIGINS", "*")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when production uses wildcard CORS")
	}
}

func TestLoad_ProductionRequiresAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("STALWART_BASE_URL", "http://localhost:8080")
	t.Setenv("STALWART_ADMIN_USER", "admin")
	t.Setenv("STALWART_ADMIN_PASSWORD", "secure-password")
	t.Setenv("JWT_SECRET", "secure-secret")
	t.Setenv("APP_ENV", "production")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when production has no ALLOWED_ORIGINS")
	}
}

func TestLoad_CustomDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
	t.Setenv("STALWART_BASE_URL", "http://localhost:8080")
	t.Setenv("STALWART_ADMIN_USER", "admin")
	t.Setenv("STALWART_ADMIN_PASSWORD", "secure-production-password")
	t.Setenv("JWT_SECRET", "secure-production-secret")
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("ALLOWED_ORIGINS", "https://example.com,https://app.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected APP_ENV=production, got: %s", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("expected HTTP_ADDR=:9090, got: %s", cfg.HTTPAddr)
	}
}
