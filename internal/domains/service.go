package domains

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/policy"
)

type Service struct {
	repo      *Repository
	policySvc *policy.Service
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo:      NewRepository(pool),
		policySvc: policy.NewService(pool),
	}
}

func (s *Service) CreateDomain(ctx context.Context, userID uuid.UUID, name string) (*Domain, error) {
	normalized, err := NormalizeDomainName(name)
	if err != nil {
		return nil, err
	}

	// Check Entitlements
	account, err := s.policySvc.CanCreateDomain(ctx, userID)
	if err != nil {
		return nil, err
	}

	d, err := s.repo.CreateDomainTx(ctx, userID, account.ID, normalized)
	if err != nil {
		return nil, err
	}

	// Generate and set verification token
	token := make([]byte, 16)
	rand.Read(token)
	tokenHash := base64.RawURLEncoding.EncodeToString(token)

	err = s.repo.SetVerificationToken(ctx, d.ID, tokenHash)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// CreatePlatformDomain creates a platform-owned domain (admin operation).
func (s *Service) CreatePlatformDomain(ctx context.Context, name, ownershipType string, registrationEnabled bool) (*Domain, error) {
	normalized, err := NormalizeDomainName(name)
	if err != nil {
		return nil, err
	}

	// Validate ownership type
	if ownershipType != string(OwnershipTypePlatform) && ownershipType != string(OwnershipTypeUser) {
		return nil, errors.New("invalid ownership type")
	}

	d, err := s.repo.CreatePlatformDomainTx(ctx, normalized, ownershipType, registrationEnabled)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// GetDomainForAddressCheck returns a domain if it's available for address registration.
func (s *Service) GetDomainForAddressCheck(ctx context.Context, domainID uuid.UUID) (*Domain, error) {
	domain, err := s.repo.GetByID(ctx, domainID)
	if err != nil {
		return nil, err
	}

	// Check if domain is active and registration is enabled
	if domain.Status != string(StatusActive) {
		return nil, errors.New("domain is not active")
	}

	if !domain.RegistrationEnabled {
		return nil, errors.New("domain registration is not enabled")
	}

	return domain, nil
}

// GetDomainByName returns a domain by name (for checking existence).
func (s *Service) GetDomainByName(ctx context.Context, name string) (*Domain, error) {
	return s.repo.GetByName(ctx, name)
}

// ListPlatformDomains returns all platform domains available for registration.
func (s *Service) ListPlatformDomains(ctx context.Context) ([]Domain, error) {
	return s.repo.ListPlatformDomains(ctx)
}

func (s *Service) ListDomains(ctx context.Context, userID uuid.UUID) ([]Domain, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) GetDomain(ctx context.Context, id, userID uuid.UUID) (*Domain, error) {
	return s.repo.GetByIDAndUser(ctx, id, userID)
}

func (s *Service) DeleteDomain(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.DeleteByIDAndUser(ctx, id, userID)
}

func (s *Service) StartVerification(ctx context.Context, id, userID uuid.UUID) (*Domain, error) {
	d, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if d.VerificationStatus == string(VerificationVerified) {
		return d, nil
	}

	// Generate new verification token
	token := make([]byte, 16)
	rand.Read(token)
	tokenHash := base64.RawURLEncoding.EncodeToString(token)
	plaintextToken := base64.RawURLEncoding.EncodeToString(token)

	// Store the hash and update status
	err = s.repo.SetVerificationToken(ctx, d.ID, tokenHash)
	if err != nil {
		return nil, err
	}

	// Create DOMAIN_VERIFY job
	err = s.repo.CreateVerificationJob(ctx, d.ID)
	if err != nil {
		return nil, err
	}

	err = s.repo.UpdateVerificationStatus(ctx, d.ID, string(VerificationVerifying))
	if err != nil {
		return nil, err
	}
	d.VerificationStatus = string(VerificationVerifying)

	// Return the plaintext token in the response for the user to configure DNS
	// This is set in the temporary field that won't be persisted
	d.VerificationToken = &plaintextToken

	return d, nil
}
