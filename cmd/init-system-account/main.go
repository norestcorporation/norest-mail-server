package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/norest-mail/server/internal/config"
	"github.com/norest-mail/server/internal/stalwart"
)

func main() {
	// Structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("system account initialization starting")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Parse mailer daemon email
	email := cfg.MailerDaemonEmail
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		slog.Error("invalid mailer daemon email format", "email", email)
		os.Exit(1)
	}
	localPart := parts[0]
	domain := parts[1]

	slog.Info("initializing system account", 
		"email", email, 
		"name", cfg.MailerDaemonName,
		"local_part", localPart,
		"domain", domain)

	// Parse STALWART_RECOVERY_ADMIN to get username and password
	recoveryParts := strings.Split(cfg.StalwartRecoveryAdmin, ":")
	if len(recoveryParts) != 2 {
		slog.Error("invalid STALWART_RECOVERY_ADMIN format", "recovery_admin", cfg.StalwartRecoveryAdmin)
		os.Exit(1)
	}
	stalwartUser := recoveryParts[0]
	stalwartPassword := recoveryParts[1]

	// Create Stalwart client
	stalwartClient := stalwart.NewClient(
		cfg.StalwartBaseURL,
		stalwartUser,
		stalwartPassword,
	)

	ctx := context.Background()

	// Check if domain exists
	domainID, err := stalwartClient.FindDomainByName(ctx, domain)
	if err != nil {
		slog.Error("failed to find domain", "domain", domain, "error", err)
		os.Exit(1)
	}
	if domainID == "" {
		slog.Info("domain not found, creating", "domain", domain)
		domainID, err = stalwartClient.CreateDomain(ctx, domain)
		if err != nil {
			slog.Error("failed to create domain", "domain", domain, "error", err)
			os.Exit(1)
		}
		slog.Info("domain created successfully", "domain", domain, "domain_id", domainID)
	} else {
		slog.Info("domain found", "domain", domain, "domain_id", domainID)
	}

	// Check if account already exists
	accountID, err := stalwartClient.FindAccountByName(ctx, email)
	if err != nil {
		slog.Error("failed to check account existence", "email", email, "error", err)
		os.Exit(1)
	}

	if accountID != "" {
		slog.Info("system account already exists", "email", email, "account_id", accountID)
		slog.Info("system account initialization completed successfully")
		fmt.Printf("System account ID: %s\n", accountID)
		return
	}

	// Generate secure random password
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("failed to generate random password", "error", err)
		os.Exit(1)
	}
	password := base64.RawURLEncoding.EncodeToString(b)

	// Create the system account
	accountID, err = stalwartClient.CreateAccount(ctx, localPart, domainID, password, cfg.MailerDaemonName)
	if err != nil {
		slog.Error("failed to create system account", "email", email, "error", err)
		os.Exit(1)
	}

	slog.Info("system account created successfully", 
		"email", email, 
		"account_id", accountID,
		"name", cfg.MailerDaemonName)

	slog.Info("system account initialization completed successfully")
	fmt.Printf("System account created:\n")
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Name: %s\n", cfg.MailerDaemonName)
	fmt.Printf("  Account ID: %s\n", accountID)
	fmt.Printf("  Password: %s\n", password)
	fmt.Printf("\nIMPORTANT: Store this password securely for system account authentication.\n")
}
