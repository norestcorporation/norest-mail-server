package policy

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/product"
)

var (
	ErrQuotaExceeded    = errors.New("quota exceeded for current plan")
	ErrAccountSuspended = errors.New("account is suspended")
)

type Service struct {
	pool        *pgxpool.Pool
	productRepo *product.Repository
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:        pool,
		productRepo: product.NewRepository(pool),
	}
}

// Entitlement wraps the plan and current usage.
type Entitlement struct {
	Plan             *product.Plan
	Account          *product.Account
	CurrentDomains   int
	CurrentMailboxes int
	CurrentAddresses int
}

func (s *Service) GetEntitlement(ctx context.Context, userID uuid.UUID) (*Entitlement, error) {
	account, err := s.productRepo.GetAccountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("no product account found")
	}

	_, plan, err := s.productRepo.GetSubscription(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("no active subscription found")
	}

	var e Entitlement
	e.Plan = plan
	e.Account = account

	// Get usage
	err = s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM domains WHERE product_account_id = $1 AND status != 'disabled'),
			(SELECT count(*) FROM mailboxes m JOIN addresses a ON m.address_id = a.id JOIN domains d ON a.domain_id = d.id WHERE d.product_account_id = $1 AND m.status != 'inactive'),
			(SELECT count(*) FROM addresses a JOIN domains d ON a.domain_id = d.id WHERE d.product_account_id = $1 AND a.status != 'inactive')
	`, account.ID).Scan(&e.CurrentDomains, &e.CurrentMailboxes, &e.CurrentAddresses)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (s *Service) SuspendAccount(ctx context.Context, accountID uuid.UUID) error {
	return s.productRepo.UpdateAccountStatus(ctx, accountID, "SUSPENDED")
}

func (s *Service) ReactivateAccount(ctx context.Context, accountID uuid.UUID) error {
	return s.productRepo.UpdateAccountStatus(ctx, accountID, "ACTIVE")
}

func (s *Service) CanCreateDomain(ctx context.Context, userID uuid.UUID) (*product.Account, error) {
	ent, err := s.GetEntitlement(ctx, userID)
	if err != nil {
		return nil, err
	}
	if ent.Account.Status == "SUSPENDED" {
		return nil, ErrAccountSuspended
	}
	if ent.CurrentDomains >= ent.Plan.MaxDomains {
		return nil, ErrQuotaExceeded
	}
	return ent.Account, nil
}

func (s *Service) CanCreateMailbox(ctx context.Context, userID uuid.UUID) error {
	ent, err := s.GetEntitlement(ctx, userID)
	if err != nil {
		return err
	}
	if ent.Account.Status == "SUSPENDED" {
		return ErrAccountSuspended
	}
	if ent.CurrentMailboxes >= ent.Plan.MaxMailboxes {
		return ErrQuotaExceeded
	}
	return nil
}

func (s *Service) CanCreateAddress(ctx context.Context, userID uuid.UUID) error {
	ent, err := s.GetEntitlement(ctx, userID)
	if err != nil {
		return err
	}
	if ent.Account.Status == "SUSPENDED" {
		return ErrAccountSuspended
	}
	if ent.CurrentAddresses >= ent.Plan.MaxAddresses {
		return ErrQuotaExceeded
	}
	return nil
}
