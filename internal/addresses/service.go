package addresses

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/domains"
	"github.com/norest-mail/server/internal/policy"
)

type Service struct {
	repo        *Repository
	domainsRepo *domains.Repository
	policySvc   *policy.Service
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo:        NewRepository(pool),
		domainsRepo: domains.NewRepository(pool),
		policySvc:   policy.NewService(pool),
	}
}

func (s *Service) CreateAddress(ctx context.Context, userID, domainID uuid.UUID, localPart string) (*Address, error) {
	// 1. Verify domain exists and is available for registration
	domain, err := s.domainsRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, err // ErrDomainNotFound bubble up
	}

	// Check if domain is active and registration is enabled
	if domain.Status != string(domains.StatusActive) {
		return nil, errors.New("domain is not active")
	}

	if !domain.RegistrationEnabled {
		return nil, errors.New("domain registration is not enabled")
	}

	// For custom domains, verify ownership and verification status
	if domain.OwnershipType == string(domains.OwnershipTypeUser) {
		// Check if domain belongs to the user
		if domain.UserID == nil || *domain.UserID != userID {
			return nil, errors.New("domain does not belong to the user")
		}

		// Check if domain is verified
		if domain.VerificationStatus != string(domains.VerificationVerified) {
			return nil, errors.New("domain must be verified before registering addresses")
		}
	}

	normalized, err := NormalizeLocalPart(localPart)
	if err != nil {
		return nil, err
	}

	// 2. Check entitlements
	err = s.policySvc.CanCreateAddress(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = s.policySvc.CanCreateMailbox(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 3. Reserve the address (race-safe database operation)
	address, err := s.repo.ReserveAddress(ctx, domainID, normalized, userID, 2) // 2 hour reservation
	if err != nil {
		return nil, err
	}

	// 4. Create mailbox record and provisioning job
	return s.createMailboxForAddress(ctx, address.ID, userID)
}

// ReserveAddress reserves an address without creating the mailbox (for signup flow).
func (s *Service) ReserveAddress(ctx context.Context, userID, domainID uuid.UUID, localPart string) (*Address, error) {
	// 1. Verify domain exists and is available for registration
	domain, err := s.domainsRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, err
	}

	// Check if domain is active and registration is enabled
	if domain.Status != string(domains.StatusActive) {
		return nil, errors.New("domain is not active")
	}

	if !domain.RegistrationEnabled {
		return nil, errors.New("domain registration is not enabled")
	}

	// For custom domains, verify ownership and verification status
	if domain.OwnershipType == string(domains.OwnershipTypeUser) {
		// Check if domain belongs to the user
		if domain.UserID == nil || *domain.UserID != userID {
			return nil, errors.New("domain does not belong to the user")
		}

		// Check if domain is verified
		if domain.VerificationStatus != string(domains.VerificationVerified) {
			return nil, errors.New("domain must be verified before registering addresses")
		}
	}

	normalized, err := NormalizeLocalPart(localPart)
	if err != nil {
		return nil, err
	}

	// 2. Check entitlements
	err = s.policySvc.CanCreateAddress(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 3. Reserve the address (race-safe database operation)
	return s.repo.ReserveAddress(ctx, domainID, normalized, userID, 2) // 2 hour reservation
}

// ClaimAddress claims a reserved address and creates the mailbox.
func (s *Service) ClaimAddress(ctx context.Context, addressID uuid.UUID, userID uuid.UUID) error {
	// 1. Claim the address (race-safe database operation)
	err := s.repo.ClaimAddress(ctx, addressID, userID)
	if err != nil {
		return err
	}

	// 2. Create mailbox record and provisioning job
	_, err = s.createMailboxForAddress(ctx, addressID, userID)
	return err
}

// createMailboxForAddress creates a mailbox record and provisioning job for an address.
func (s *Service) createMailboxForAddress(ctx context.Context, addressID uuid.UUID, userID uuid.UUID) (*Address, error) {
	// Check entitlements
	err := s.policySvc.CanCreateMailbox(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create mailbox record for the already-reserved address
	err = s.repo.CreateMailboxForAddress(ctx, addressID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, addressID)
}

// CheckAddressAvailability checks if an address is available for reservation.
func (s *Service) CheckAddressAvailability(ctx context.Context, domainID uuid.UUID, localPart string) (bool, error) {
	// Verify domain exists and is available for registration
	domain, err := s.domainsRepo.GetByID(ctx, domainID)
	if err != nil {
		return false, err
	}

	// Check if domain is active and registration is enabled
	if domain.Status != string(domains.StatusActive) {
		return false, nil
	}

	if !domain.RegistrationEnabled {
		return false, nil
	}

	// For custom domains, check verification status
	if domain.OwnershipType == string(domains.OwnershipTypeUser) {
		if domain.VerificationStatus != string(domains.VerificationVerified) {
			return false, nil
		}
	}

	normalized, err := NormalizeLocalPart(localPart)
	if err != nil {
		return false, err
	}

	return s.repo.CheckAddressAvailability(ctx, domainID, normalized)
}

func (s *Service) ListAddresses(ctx context.Context, userID, domainID uuid.UUID) ([]Address, error) {
	// 1. Verify domain exists and check ownership for custom domains
	domain, err := s.domainsRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, err
	}

	// For custom domains, verify ownership
	if domain.OwnershipType == string(domains.OwnershipTypeUser) {
		if domain.UserID == nil || *domain.UserID != userID {
			return nil, errors.New("domain does not belong to the user")
		}
	}

	return s.repo.ListByDomainID(ctx, domainID)
}
