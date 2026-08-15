package domains

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidDomain  = errors.New("invalid domain name")
	ErrDomainExists   = errors.New("domain already exists")
	ErrDomainNotFound = errors.New("domain not found")
)

// Status represents the lifecycle state of a domain.
type Status string

const (
	StatusPending   Status = "pending"
	StatusVerifying Status = "verifying"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDisabled  Status = "disabled"
)

// OwnershipType represents who owns the domain.
type OwnershipType string

const (
	OwnershipTypePlatform OwnershipType = "PLATFORM"
	OwnershipTypeUser     OwnershipType = "USER"
)

// VerificationStatus represents the domain ownership verification state.
type VerificationStatus string

const (
	VerificationPending   VerificationStatus = "pending"
	VerificationVerifying VerificationStatus = "verifying"
	VerificationVerified  VerificationStatus = "verified"
	VerificationFailed    VerificationStatus = "failed"
)

// Domain represents a mail domain registered in the Norest control plane.
type Domain struct {
	ID                    uuid.UUID       `json:"id"`
	UserID                *uuid.UUID      `json:"user_id,omitempty"`
	ProductAccountID      *uuid.UUID      `json:"product_account_id,omitempty"`
	Name                  string          `json:"name"`
	StalwartDomainID      *string         `json:"stalwart_domain_id,omitempty"`
	Status                string          `json:"status"`
	VerificationStatus    string          `json:"verification_status"`
	VerificationTokenHash *string         `json:"-"`
	OwnershipType         string          `json:"ownership_type"`
	RegistrationEnabled   bool            `json:"registration_enabled"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// NormalizeDomainName normalizes a domain name for storage and validation.
func NormalizeDomainName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)

	if name == "" {
		return "", ErrInvalidDomain
	}

	// Remove trailing dot if present
	if strings.HasSuffix(name, ".") {
		name = name[:len(name)-1]
	}

	// Basic check for obvious invalid characters
	if strings.ContainsAny(name, " \t\n\r") {
		return "", ErrInvalidDomain
	}

	return name, nil
}

type CreateDomainRequest struct {
	Name              string `json:"name"`
	OwnershipType     string `json:"ownership_type,omitempty"`     // "PLATFORM" or "USER"
	RegistrationEnabled bool  `json:"registration_enabled,omitempty"`
}
