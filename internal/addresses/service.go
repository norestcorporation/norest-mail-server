package addresses

import (
	"context"

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
	// 1. Verify user owns domain and domain is not deleted.
	_, err := s.domainsRepo.GetByIDAndUser(ctx, domainID, userID)
	if err != nil {
		return nil, err // ErrDomainNotFound bubble up
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

	return s.repo.CreateAddressTx(ctx, domainID, normalized)
}

func (s *Service) ListAddresses(ctx context.Context, userID, domainID uuid.UUID) ([]Address, error) {
	// 1. Verify user owns domain
	_, err := s.domainsRepo.GetByIDAndUser(ctx, domainID, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListByDomainID(ctx, domainID)
}
