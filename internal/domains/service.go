package domains

import (
	"context"
	"crypto/rand"
	"encoding/base64"

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

	return d, nil
}
