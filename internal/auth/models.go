package auth

import (
	"strings"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Status       string    `json:"status"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"` // Token expiration time in seconds
}

type UserResponse struct {
	ID     uuid.UUID `json:"id"`
	Email  string    `json:"email"`
	Status string    `json:"status"`
}

// RegistrationStatus represents the current stage of user registration.
type RegistrationStatus string

const (
	RegistrationStatusPending       RegistrationStatus = "pending"        // Initial registration, no domain
	RegistrationStatusDomainAdded   RegistrationStatus = "domain_added"   // Domain added but not verified
	RegistrationStatusVerifying     RegistrationStatus = "verifying"      // Domain verification in progress
	RegistrationStatusVerified      RegistrationStatus = "verified"       // Domain verified, ready for address
	RegistrationStatusProvisioning  RegistrationStatus = "provisioning"    // Address provisioning in progress
	RegistrationStatusActive        RegistrationStatus = "active"         // Fully active
)

// DomainType represents the type of domain in registration flow.
type DomainType string

const (
	DomainTypePlatform DomainType = "platform_owned" // Platform-owned domain
	DomainTypeCustom   DomainType = "custom"          // User-owned custom domain
)

// RegistrationFlowResponse provides detailed registration status.
type RegistrationFlowResponse struct {
	ID                 uuid.UUID          `json:"id"`
	Email              string             `json:"email"`
	DomainType         DomainType         `json:"domain_type"`
	Status             RegistrationStatus `json:"status"`
	RequiresAction     *string            `json:"requires_action,omitempty"`
	DomainID           *uuid.UUID         `json:"domain_id,omitempty"`
	DomainName         *string            `json:"domain_name,omitempty"`
	DomainVerified     bool               `json:"domain_verified"`
	AddressID          *uuid.UUID         `json:"address_id,omitempty"`
	MailboxProvisioned bool               `json:"mailbox_provisioned"`
	ReadyForMail       bool               `json:"ready_for_mail"`
}

// ExtractDomainFromEmail extracts the domain part from an email address.
func ExtractDomainFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return strings.ToLower(parts[1])
	}
	return ""
}
