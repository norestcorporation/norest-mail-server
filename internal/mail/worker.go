package mail

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/stalwart"
)

// ReconciliationWorker periodically checks for stale intents and out-of-band Stalwart changes.
type ReconciliationWorker struct {
	pool           *pgxpool.Pool
	stalwartClient *stalwart.Client
	pollInterval   time.Duration
}

func NewReconciliationWorker(pool *pgxpool.Pool, stalwartClient *stalwart.Client) *ReconciliationWorker {
	return &ReconciliationWorker{
		pool:           pool,
		stalwartClient: stalwartClient,
		pollInterval:   10 * time.Second, // Poll every 10s
	}
}

func (w *ReconciliationWorker) Run(ctx context.Context) error {
	slog.Info("starting reconciliation worker")
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.reconcileStaleIntents(ctx)
			w.syncOutboxes(ctx)
		}
	}
}

func (w *ReconciliationWorker) reconcileStaleIntents(ctx context.Context) {
	query := `
		SELECT id, user_id, idempotency_key, mutation_type, payload
		FROM mail_reconciliation_logs
		WHERE status IN ('PENDING', 'EXECUTING', 'UNKNOWN')
		AND updated_at < NOW() - INTERVAL '5 minutes'
	`
	rows, err := w.pool.Query(ctx, query)
	if err != nil {
		slog.Error("failed to query stale intents", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, userID, key, mutationType string
		var payload []byte
		if err := rows.Scan(&id, &userID, &key, &mutationType, &payload); err != nil {
			continue
		}
		
		// Very basic deterministic reconciliation: if it was a send, we check if there's an email with this idempotency key in a header
		if mutationType == "Email/set" {
			// For full deterministic reconciliation, we would need the accountID.
			// Since we don't have it easily here without a join, we preserve UNKNOWN.
			w.pool.Exec(ctx, `UPDATE mail_reconciliation_logs SET status = 'UNKNOWN', updated_at = NOW(), stalwart_response = '{"error": "requires_manual_reconciliation"}' WHERE id = $1`, id)
		} else {
			// For other operations, preserve UNKNOWN if we can't verify
			w.pool.Exec(ctx, `UPDATE mail_reconciliation_logs SET status = 'UNKNOWN', updated_at = NOW(), stalwart_response = '{"error": "reconciliation_impossible"}' WHERE id = $1`, id)
		}
	}
}

func (w *ReconciliationWorker) syncOutboxes(ctx context.Context) {
	// 1. Fetch mailboxes and their sync state using FOR UPDATE SKIP LOCKED
	query := `
		SELECT m.id, m.user_id, a.stalwart_account_id, s.state
		FROM mailboxes m
		JOIN addresses a ON a.mailbox_id = m.id AND a.is_primary = true
		JOIN mail_sync_state s ON s.mailbox_id = m.id
		WHERE s.status = 'idle'
		ORDER BY s.last_synced_at ASC NULLS FIRST
		LIMIT 10
		FOR UPDATE OF s SKIP LOCKED
	`
	rows, err := w.pool.Query(ctx, query)
	if err != nil {
		slog.Error("failed to fetch mailboxes for sync", "error", err)
		return
	}
	defer rows.Close()

	type syncJob struct {
		MailboxID         string
		UserID            string
		StalwartAccountID string
		State             string
	}
	var jobs []syncJob

	for rows.Next() {
		var j syncJob
		if err := rows.Scan(&j.MailboxID, &j.UserID, &j.StalwartAccountID, &j.State); err != nil {
			slog.Error("failed to scan sync job", "error", err)
			continue
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	for _, job := range jobs {
		w.processSyncJob(ctx, job.MailboxID, job.UserID, job.StalwartAccountID, job.State)
	}
}

func (w *ReconciliationWorker) processSyncJob(ctx context.Context, mailboxID, userID, stalwartAcctID, state string) {
	// Mark as syncing
	w.pool.Exec(ctx, "UPDATE mail_sync_state SET status = 'syncing' WHERE mailbox_id = $1", mailboxID)

	if state == "" {
		// Just clear it
		w.pool.Exec(ctx, "UPDATE mail_sync_state SET status = 'idle', last_synced_at = NOW() WHERE mailbox_id = $1", mailboxID)
		return
	}

	// Fetch changes
	ec, err := w.stalwartClient.GetChanges(ctx, stalwartAcctID, "Email", state)
	if err != nil {
		if err.Error() == "cannotCalculateChanges" {
			w.pool.Exec(ctx, "UPDATE mail_sync_state SET state = '', status = 'error', error_message = 'cannotCalculateChanges', updated_at = NOW() WHERE mailbox_id = $1", mailboxID)
		} else {
			w.pool.Exec(ctx, "UPDATE mail_sync_state SET status = 'idle' WHERE mailbox_id = $1", mailboxID)
		}
		return
	}

	newState := ec.NewState

	// Publish events
	tx, err := w.pool.Begin(ctx)
	if err == nil {
		defer tx.Rollback(ctx)

		for _, id := range ec.Created {
			w.emitEvent(ctx, tx, userID, "message.created", map[string]any{"message_id": id})
		}
		for _, id := range ec.Updated {
			w.emitEvent(ctx, tx, userID, "message.updated", map[string]any{"message_id": id})
		}
		for _, id := range ec.Destroyed {
			w.emitEvent(ctx, tx, userID, "message.deleted", map[string]any{"message_id": id})
		}

		// Update state
		_, err = tx.Exec(ctx, "UPDATE mail_sync_state SET state = $1, status = 'idle', last_synced_at = NOW(), updated_at = NOW() WHERE mailbox_id = $2", newState, mailboxID)
		if err == nil {
			tx.Commit(ctx)
		}
	}
}

func (w *ReconciliationWorker) emitEvent(ctx context.Context, tx pgx.Tx, userID string, eventType string, payload any) {
	b, _ := json.Marshal(payload)
	tx.Exec(ctx, `
		INSERT INTO mail_events_outbox (user_id, event_type, payload, status)
		VALUES ($1, $2, $3, 'pending')
	`, userID, eventType, b)
}
