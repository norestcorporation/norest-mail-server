package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Provider interface {
	CreateCustomer(ctx context.Context, email string) (string, error)
	CreateSubscription(ctx context.Context, customerID string, planCode string) (string, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
}

type Service struct {
	pool     *pgxpool.Pool
	provider Provider
}

func NewService(pool *pgxpool.Pool, provider Provider) *Service {
	return &Service{
		pool:     pool,
		provider: provider,
	}
}

// HandleWebhook processes a billing event idempotently.
func (s *Service) HandleWebhook(ctx context.Context, provider, eventID, eventType, payloadHash string, accountID uuid.UUID, planCode string) (bool, error) {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := s.pool.Begin(txCtx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(txCtx)

	// Try to insert the event as PENDING
	var status string
	err = tx.QueryRow(txCtx, `
		INSERT INTO billing_events (provider, provider_event_id, event_type, payload_hash, status)
		VALUES ($1, $2, $3, $4, 'PENDING')
		ON CONFLICT (provider, provider_event_id) DO NOTHING
		RETURNING status
	`, provider, eventID, eventType, payloadHash).Scan(&status)
	if err != nil {
		return false, err
	}

	// If no row was returned (conflict), fetch the existing status
	if status == "" {
		err = tx.QueryRow(txCtx, `
			SELECT status FROM billing_events 
			WHERE provider = $1 AND provider_event_id = $2
		`, provider, eventID).Scan(&status)
		if err != nil {
			return false, err
		}
	}

	if status == "PROCESSED" {
		// Idempotency: already processed
		return false, nil // return false indicating no mutation happened this time
	}

	// Update the subscription if a planCode is provided
	if planCode != "" {
		// Find the plan ID
		var planID uuid.UUID
		err = tx.QueryRow(txCtx, `SELECT id FROM plans WHERE code = $1`, planCode).Scan(&planID)
		if err == nil {
			// Update the subscription for this account
			_, err = tx.Exec(txCtx, `
				UPDATE subscriptions 
				SET plan_id = $1, updated_at = NOW() 
				WHERE product_account_id = $2 AND status IN ('ACTIVE', 'TRIALING')
			`, planID, accountID)
			if err != nil {
				return false, err
			}

			// Trigger QUOTA_SYNC job
			_, err = tx.Exec(txCtx, `
				INSERT INTO provisioning_jobs (type, resource_id, status)
				VALUES ('ACCOUNT_QUOTA_SYNC', $1, 'PENDING')
			`, accountID)
			if err != nil {
				return false, err
			}
		}
	}

	// Mark as processed
	_, err = tx.Exec(txCtx, `
		UPDATE billing_events 
		SET status = 'PROCESSED', processed_at = NOW() 
		WHERE provider = $1 AND provider_event_id = $2
	`, provider, eventID)
	if err != nil {
		return false, err
	}

	err = tx.Commit(txCtx)
	if err != nil {
		return false, err
	}

	return true, nil
}
