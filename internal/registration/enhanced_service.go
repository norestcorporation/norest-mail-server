package registration

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/addresses"
	"github.com/norest-mail/server/internal/auth"
	"github.com/norest-mail/server/internal/domains"
	"github.com/norest-mail/server/internal/policy"
)

var (
	ErrDomainNotAvailable = errors.New("domain not available for registration")
	ErrAddressNotAvailable = errors.New("address not available")
)

type EnhancedService struct {
	pool           *pgxpool.Pool
	authService    *auth.Service
	domainsService *domains.Service
	addressesService *addresses.Service
	policyService  *policy.Service
}

func NewEnhancedService(pool *pgxpool.Pool, authService *auth.Service, domainsService *domains.Service, addressesService *addresses.Service, policyService *policy.Service) *EnhancedService {
	return &EnhancedService{
		pool:           pool,
		authService:    authService,
		domainsService: domainsService,
		addressesService: addressesService,
		policyService:  policyService,
	}
}

// RegisterWithDomainDetection handles user registration with automatic domain type detection.
func (s *EnhancedService) RegisterWithDomainDetection(ctx context.Context, email, password string) (*auth.AuthResponse, *auth.RegistrationFlowResponse, error) {
	// 1. Register the user
	authRes, err := s.authService.Register(ctx, email, password)
	if err != nil {
		return nil, nil, err
	}

	// 2. Extract domain from email
	domainName := auth.ExtractDomainFromEmail(email)
	if domainName == "" {
		return authRes, s.createPendingFlowResponse(authRes), nil
	}

	// 3. Check if domain exists and determine type
	domain, err := s.domainsService.GetDomainByName(ctx, domainName)
	if err != nil {
		// Domain doesn't exist, treat as custom domain requiring user to add it
		return authRes, s.createCustomDomainFlowResponse(authRes, domainName), nil
	}

	// 4. Determine flow based on domain type
	if domain.OwnershipType == string(domains.OwnershipTypePlatform) {
		// Platform domain flow - auto-provision
		return s.handlePlatformDomainFlow(ctx, authRes, domain, email)
	} else {
		// Custom domain flow - user owns it
		return s.handleCustomDomainFlow(ctx, authRes, domain, email)
	}
}

// handlePlatformDomainFlow handles registration for platform-owned domains.
func (s *EnhancedService) handlePlatformDomainFlow(ctx context.Context, authRes *auth.AuthResponse, domain *domains.Domain, email string) (*auth.AuthResponse, *auth.RegistrationFlowResponse, error) {
	slog.Info("handlePlatformDomainFlow called", "email", email, "domain", domain.Name, "domain_status", domain.Status, "registration_enabled", domain.RegistrationEnabled)
	
	// Extract local part from email
	atIndex := strings.LastIndex(email, "@")
	if atIndex == -1 {
		slog.Error("invalid email format", "email", email)
		return authRes, s.createPlatformDomainFlowResponse(authRes, domain, nil, "invalid_email"), errors.New("invalid email format")
	}
	
	localPart := email[:atIndex]
	slog.Info("extracted local part", "local_part", localPart)

	// Check if domain is ready for registration
	if domain.Status != string(domains.StatusActive) || !domain.RegistrationEnabled {
		slog.Error("domain not ready for registration", "domain_status", domain.Status, "registration_enabled", domain.RegistrationEnabled)
		return authRes, s.createPlatformDomainFlowResponse(authRes, domain, nil, "domain_not_ready"), errors.New("domain not ready for registration")
	}

	// Check address availability
	available, err := s.addressesService.CheckAddressAvailability(ctx, domain.ID, localPart)
	if err != nil {
		slog.Error("address availability check failed", "error", err, "domain_id", domain.ID, "local_part", localPart)
		return authRes, s.createPlatformDomainFlowResponse(authRes, domain, nil, "availability_check_failed"), err
	}
	slog.Info("address availability check", "available", available, "local_part", localPart)

	if !available {
		slog.Error("address not available", "local_part", localPart)
		return authRes, s.createPlatformDomainFlowResponse(authRes, domain, nil, "address_not_available"), ErrAddressNotAvailable
	}

	// Reserve address
	address, err := s.addressesService.ReserveAddress(ctx, authRes.ID, domain.ID, localPart)
	if err != nil {
		slog.Error("address reservation failed", "error", err, "user_id", authRes.ID, "domain_id", domain.ID, "local_part", localPart)
		return authRes, s.createPlatformDomainFlowResponse(authRes, domain, nil, "reservation_failed"), err
	}
	slog.Info("address reserved successfully", "address_id", address.ID, "local_part", localPart)

	// Claim address to trigger provisioning
	err = s.addressesService.ClaimAddress(ctx, address.ID, authRes.ID)
	if err != nil {
		slog.Error("address claim failed", "error", err, "address_id", address.ID)
		return authRes, s.createPlatformDomainFlowResponse(authRes, domain, &address.ID, "claim_failed"), err
	}
	slog.Info("address claimed successfully, provisioning triggered", "address_id", address.ID)

	// Return provisioning status
	return authRes, s.createPlatformDomainFlowResponse(authRes, domain, &address.ID, "provisioning"), nil
}

// handleCustomDomainFlow handles registration for custom domains.
func (s *EnhancedService) handleCustomDomainFlow(ctx context.Context, authRes *auth.AuthResponse, domain *domains.Domain, email string) (*auth.AuthResponse, *auth.RegistrationFlowResponse, error) {
	// For custom domains, the user needs to verify ownership
	// Return response indicating domain verification is needed
	var action string
	switch domain.VerificationStatus {
	case string(domains.VerificationPending):
		action = "start_verification"
	case string(domains.VerificationVerifying):
		action = "wait_for_verification"
	case string(domains.VerificationVerified):
		action = "register_address"
	case string(domains.VerificationFailed):
		action = "retry_verification"
	default:
		action = "verify_domain"
	}

	return authRes, &auth.RegistrationFlowResponse{
		ID:             authRes.ID,
		Email:          authRes.Email,
		DomainType:     auth.DomainTypeCustom,
		Status:         auth.RegistrationStatusDomainAdded,
		RequiresAction: &action,
		DomainID:       &domain.ID,
		DomainName:     &domain.Name,
		DomainVerified: domain.VerificationStatus == string(domains.VerificationVerified),
		AddressID:      nil,
		MailboxProvisioned: false,
		ReadyForMail:   false,
	}, nil
}

// Helper functions for creating flow responses

func (s *EnhancedService) createPendingFlowResponse(authRes *auth.AuthResponse) *auth.RegistrationFlowResponse {
	action := "add_domain"
	return &auth.RegistrationFlowResponse{
		ID:             authRes.ID,
		Email:          authRes.Email,
		DomainType:     "", // Unknown yet
		Status:         auth.RegistrationStatusPending,
		RequiresAction: &action,
		DomainID:       nil,
		DomainName:     nil,
		DomainVerified: false,
		AddressID:      nil,
		MailboxProvisioned: false,
		ReadyForMail:   false,
	}
}

func (s *EnhancedService) createCustomDomainFlowResponse(authRes *auth.AuthResponse, domainName string) *auth.RegistrationFlowResponse {
	action := "add_domain"
	return &auth.RegistrationFlowResponse{
		ID:             authRes.ID,
		Email:          authRes.Email,
		DomainType:     auth.DomainTypeCustom,
		Status:         auth.RegistrationStatusPending,
		RequiresAction: &action,
		DomainID:       nil,
		DomainName:     &domainName,
		DomainVerified: false,
		AddressID:      nil,
		MailboxProvisioned: false,
		ReadyForMail:   false,
	}
}

func (s *EnhancedService) createPlatformDomainFlowResponse(authRes *auth.AuthResponse, domain *domains.Domain, addressID *uuid.UUID, status string) *auth.RegistrationFlowResponse {
	var action *string
	var regStatus auth.RegistrationStatus

	switch status {
	case "provisioning":
		action = nil
		regStatus = auth.RegistrationStatusProvisioning
	case "domain_not_ready":
		a := "wait_for_domain"
		action = &a
		regStatus = auth.RegistrationStatusPending
	case "address_not_available":
		a := "choose_different_address"
		action = &a
		regStatus = auth.RegistrationStatusPending
	case "invalid_email":
		a := "provide_valid_email"
		action = &a
		regStatus = auth.RegistrationStatusPending
	default:
		action = nil
		regStatus = auth.RegistrationStatusProvisioning
	}

	return &auth.RegistrationFlowResponse{
		ID:             authRes.ID,
		Email:          authRes.Email,
		DomainType:     auth.DomainTypePlatform,
		Status:         regStatus,
		RequiresAction: action,
		DomainID:       &domain.ID,
		DomainName:     &domain.Name,
		DomainVerified: true,
		AddressID:      addressID,
		MailboxProvisioned: status == "provisioning",
		ReadyForMail:   false, // Will be true after provisioning completes
	}
}