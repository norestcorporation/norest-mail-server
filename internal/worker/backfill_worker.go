package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/db"
	"github.com/norest-mail/server/internal/mail"
	"github.com/norest-mail/server/internal/stalwart"
)

type BackfillWorker struct {
	pool       *pgxpool.Pool
	stalwart   *stalwart.Client
	repository *db.MailRepository
}

func NewBackfillWorker(pool *pgxpool.Pool, stalwart *stalwart.Client, repo *db.MailRepository) *BackfillWorker {
	return &BackfillWorker{
		pool:       pool,
		stalwart:   stalwart,
		repository: repo,
	}
}

// RunBackfillBatch processes a single batch of emails for a given account.
// It is safely resumable and idempotent.
func (w *BackfillWorker) RunBackfillBatch(ctx context.Context, accountID string, stalwartAccountID string, limit int) (bool, error) {
	// 1. Fetch current backfill position
	var position *string
	err := w.pool.QueryRow(ctx, "SELECT backfill_position FROM sync_checkpoints WHERE account_id = $1", accountID).Scan(&position)
	if err != nil && err != pgx.ErrNoRows {
		return false, fmt.Errorf("getting backfill position: %w", err)
	}

	// 2. Fetch Email/query from Stalwart
	posInt := 0
	if position != nil && *position != "" {
		fmt.Sscanf(*position, "%d", &posInt)
	}

	queryResp, err := w.stalwart.EmailQueryWithPosition(ctx, stalwartAccountID, nil, nil, posInt, limit)
	if err != nil {
		return false, fmt.Errorf("email query: %w", err)
	}

	if len(queryResp.IDs) == 0 {
		// Complete
		return true, nil
	}

	// 3. Fetch full Email details
	emails, err := w.stalwart.EmailGet(ctx, stalwartAccountID, queryResp.IDs, nil)
	if err != nil {
		return false, fmt.Errorf("email get: %w", err)
	}

	// Process emails in a transaction
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, e := range emails.List {
		err = w.processEmail(ctx, tx, accountID, &e)
		if err != nil {
			return false, fmt.Errorf("process email %s: %w", e.ID, err)
		}
	}

	// Update checkpoint
	newPos := posInt + len(queryResp.IDs)
	newPosStr := fmt.Sprintf("%d", newPos)
	_, err = tx.Exec(ctx, `
		INSERT INTO sync_checkpoints (account_id, backfill_position, status, updated_at)
		VALUES ($1, $2, 'backfilling', NOW())
		ON CONFLICT (account_id) DO UPDATE SET 
			backfill_position = EXCLUDED.backfill_position,
			status = 'backfilling',
			updated_at = NOW()
	`, accountID, newPosStr)
	if err != nil {
		return false, fmt.Errorf("update checkpoint: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}

	slog.Info("Processed backfill batch", "account_id", accountID, "count", len(queryResp.IDs), "new_position", newPos)

	return false, nil // returning false means there might be more
}

func (w *BackfillWorker) processEmail(ctx context.Context, tx pgx.Tx, accountID string, email *stalwart.Email) error {
	// 1. Thread Resolution
	// Use placeholder time if empty
	now := time.Now()
	res, err := mail.ResolveThread(ctx, tx, accountID, email, &now)
	if err != nil {
		return fmt.Errorf("resolve thread: %w", err)
	}

	// 2. Upsert Message Projection
	senderBytes, _ := json.Marshal(email.From)
	recipBytes, _ := json.Marshal(email.To)

	msgID := ""
	if len(email.MessageId) > 0 {
		msgID = email.MessageId[0]
	}

	var inReplyTo *string
	if len(email.InReplyTo) > 0 {
		inReplyTo = &email.InReplyTo[0]
	}

	msg := &db.Message{
		AccountID:       accountID,
		ThreadID:        res.ThreadID,
		StalwartEmailID: email.ID,
		MessageID:       &msgID,
		InReplyTo:       inReplyTo,
		ReferencesChain: email.References,
		Subject:         &email.Subject,
		Sender:          senderBytes,
		Recipients:      recipBytes,
	}

	insertedMsg, err := w.repository.UpsertMessage(ctx, tx, msg)
	if err != nil {
		return fmt.Errorf("upsert message: %w", err)
	}

	// 3. Upsert Message Mailboxes
	for mboxID, isPresent := range email.MailboxIDs {
		if !isPresent {
			continue
		}
		isUnread := email.Keywords["$seen"] != true
		isDraft := email.Keywords["$draft"] == true
		isTrashed := false // To be derived from mailbox mappings
		isSent := false    // To be derived from mailbox mappings
		// In a full implementation, we lookup if mboxID is the Trash or Sent folder via mailbox_mappings table

		err = w.repository.UpsertMessageMailbox(ctx, tx, insertedMsg.ID, mboxID, isUnread, isDraft, isTrashed, isSent, nil)
		if err != nil {
			return fmt.Errorf("upsert message mailbox: %w", err)
		}
	}

	// 4. Update Thread Projection
	err = w.repository.UpdateThreadProjection(ctx, tx, res.ThreadID)
	if err != nil {
		return fmt.Errorf("update thread projection: %w", err)
	}

	// 5. Out-of-order Thread Merge Reconciliation
	// If this new message is a parent that earlier messages tried to reply to, we must merge them into this thread.
	if msgID != "" {
		err = w.reconcileOrphans(ctx, tx, accountID, msgID, res.ThreadID)
		if err != nil {
			return fmt.Errorf("reconcile orphans: %w", err)
		}
	}

	return nil
}

// reconcileOrphans finds existing messages that replied to msgID but were put into a different thread,
// and merges them into canonicalThreadID.
func (w *BackfillWorker) reconcileOrphans(ctx context.Context, tx pgx.Tx, accountID, msgID, canonicalThreadID string) error {
	// Find duplicate threads containing messages that reply to this msgID, excluding the canonical thread
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT thread_id FROM messages 
		WHERE account_id = $1 
		AND (in_reply_to = $2 OR $2 = ANY(references_chain))
		AND thread_id != $3
	`, accountID, msgID, canonicalThreadID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var duplicateThreadIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		duplicateThreadIDs = append(duplicateThreadIDs, id)
	}
	rows.Close()

	for _, dupID := range duplicateThreadIDs {
		// Move messages to canonical thread
		_, err = tx.Exec(ctx, "UPDATE messages SET thread_id = $1 WHERE thread_id = $2", canonicalThreadID, dupID)
		if err != nil {
			return err
		}

		// Rebuild canonical thread projections
		err = w.repository.UpdateThreadProjection(ctx, tx, canonicalThreadID)
		if err != nil {
			return err
		}

		// Delete duplicate thread (it should be empty now)
		_, err = tx.Exec(ctx, "DELETE FROM threads WHERE id = $1", dupID)
		if err != nil {
			return err
		}

		slog.Info("Merged thread", "duplicate_id", dupID, "canonical_id", canonicalThreadID)
	}

	return nil
}
