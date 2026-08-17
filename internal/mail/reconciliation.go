package mail

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReconciliationStore struct {
	pool *pgxpool.Pool
}

func NewReconciliationStore(pool *pgxpool.Pool) *ReconciliationStore {
	return &ReconciliationStore{pool: pool}
}

// LogIntent logs an intention to mutate state in Stalwart.
func (r *ReconciliationStore) LogIntent(ctx context.Context, userID, idempotencyKey, mutationType string, payload any) (string, error) {
	payloadBytes, _ := json.Marshal(payload)
	var id string
	
	if idempotencyKey != "" {
		// If idempotencyKey is provided, check if it exists
		query := `SELECT id, status FROM mail_reconciliation_logs WHERE user_id = $1 AND idempotency_key = $2`
		var status string
		err := r.pool.QueryRow(ctx, query, userID, idempotencyKey).Scan(&id, &status)
		if err == nil {
			// Exists
			if status == "SUCCESS" {
				return "", ErrIdempotencyMismatch // Already succeeded, shouldn't re-run
			}
			if status == "PENDING" || status == "EXECUTING" {
				return "", ErrIdempotencyInProgress
			}
			// If FAILED or UNKNOWN, we'll just re-use the ID
			_, err = r.pool.Exec(ctx, `UPDATE mail_reconciliation_logs SET status = 'EXECUTING', updated_at = NOW() WHERE id = $1`, id)
			return id, err
		}
	}
	
	query := `
		INSERT INTO mail_reconciliation_logs (user_id, idempotency_key, mutation_type, payload, status)
		VALUES ($1, $2, $3, $4, 'EXECUTING')
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query, userID, idempotencyKey, mutationType, payloadBytes).Scan(&id)
	return id, err
}

// MarkSuccess marks a mutation as successful and publishes an outbox event.
func (r *ReconciliationStore) MarkSuccess(ctx context.Context, id, userID, eventType string, eventPayload any, stalwartResponse any) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	respBytes, _ := json.Marshal(stalwartResponse)
	
	// Mark success
	_, err = tx.Exec(ctx, `
		UPDATE mail_reconciliation_logs 
		SET status = 'SUCCESS', stalwart_response = $1, updated_at = NOW() 
		WHERE id = $2
	`, respBytes, id)
	if err != nil {
		return err
	}

	// Insert into outbox if eventType is provided
	if eventType != "" {
		eventPayloadBytes, _ := json.Marshal(eventPayload)
		// We use a dummy mailbox_id since outbox schema requires it. Let's just use empty uuid if not provided.
		// Wait, outbox schema says mailbox_id is NOT NULL. 
		// Actually, let's just bypass mailbox_id or grab it from event payload?
		// We'll alter the outbox table to make mailbox_id nullable.
		_, err = tx.Exec(ctx, `
			INSERT INTO mail_events_outbox (user_id, event_type, payload, status)
			VALUES ($1, $2, $3, 'pending')
		`, userID, eventType, eventPayloadBytes)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// MarkFailed marks a mutation as failed.
func (r *ReconciliationStore) MarkFailed(ctx context.Context, id string, err error) {
	r.pool.Exec(ctx, `UPDATE mail_reconciliation_logs SET status = 'FAILED', updated_at = NOW(), stalwart_response = $1 WHERE id = $2`, `{"error": "`+err.Error()+`"}`, id)
}

// MarkUnknown marks a mutation as ambiguous (e.g. network timeout).
func (r *ReconciliationStore) MarkUnknown(ctx context.Context, id string, err error) {
	r.pool.Exec(ctx, `UPDATE mail_reconciliation_logs SET status = 'UNKNOWN', updated_at = NOW(), stalwart_response = $1 WHERE id = $2`, `{"error": "`+err.Error()+`"}`, id)
}
