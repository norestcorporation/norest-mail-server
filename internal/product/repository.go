package product

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetAccountByUserID returns the product account for a user.
func (r *Repository) GetAccountByUserID(ctx context.Context, userID uuid.UUID) (*Account, error) {
	var account Account
	err := r.pool.QueryRow(ctx, `
		SELECT pa.id, pa.status, pa.created_at, pa.updated_at
		FROM product_accounts pa
		JOIN user_product_accounts upa ON pa.id = upa.product_account_id
		WHERE upa.user_id = $1
	`, userID).Scan(
		&account.ID,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // or error
		}
		return nil, err
	}
	return &account, nil
}

// GetSubscription returns the active subscription and plan for a product account.
func (r *Repository) GetSubscription(ctx context.Context, accountID uuid.UUID) (*Subscription, *Plan, error) {
	var sub Subscription
	var plan Plan

	err := r.pool.QueryRow(ctx, `
		SELECT 
			s.id, s.product_account_id, s.plan_id, s.status, s.provider, s.provider_customer_id, s.provider_subscription_id, s.current_period_start, s.current_period_end, s.cancel_at_period_end, s.created_at, s.updated_at,
			p.id, p.code, p.name, p.status, p.max_domains, p.max_mailboxes, p.max_addresses, p.max_storage_bytes
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.product_account_id = $1 AND s.status IN ('TRIALING', 'ACTIVE', 'PAST_DUE')
		ORDER BY s.created_at DESC LIMIT 1
	`, accountID).Scan(
		&sub.ID, &sub.ProductAccountID, &sub.PlanID, &sub.Status, &sub.Provider, &sub.ProviderCustomerID, &sub.ProviderSubscriptionID, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CancelAtPeriodEnd, &sub.CreatedAt, &sub.UpdatedAt,
		&plan.ID, &plan.Code, &plan.Name, &plan.Status, &plan.MaxDomains, &plan.MaxMailboxes, &plan.MaxAddresses, &plan.MaxStorageBytes,
	)
	if err != nil {
		return nil, nil, err
	}

	return &sub, &plan, nil
}

// UpdateAccountStatus updates the status of a product account and queues a job.
func (r *Repository) UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status string) error {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(txCtx)

	tag, err := tx.Exec(txCtx, `UPDATE product_accounts SET status = $1, updated_at = NOW() WHERE id = $2`, status, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	jobType := ""
	if status == "SUSPENDED" {
		jobType = "ACCOUNT_SUSPEND"
	} else if status == "ACTIVE" {
		jobType = "ACCOUNT_REACTIVATE"
	}

	if jobType != "" {
		_, err = tx.Exec(txCtx, `
			INSERT INTO provisioning_jobs (type, resource_id, status)
			VALUES ($1, $2, 'PENDING')
		`, jobType, accountID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(txCtx)
}
