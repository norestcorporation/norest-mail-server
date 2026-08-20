package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/norest-mail/server/internal/db"
	"github.com/norest-mail/server/internal/mail"
	"github.com/norest-mail/server/internal/stalwart"
)

// ProcessEmailSync handles inserting or updating an email and resolving its thread.
func ProcessEmailSync(ctx context.Context, tx pgx.Tx, repo *db.MailRepository, accountID string, email *stalwart.Email) error {
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

	insertedMsg, err := repo.UpsertMessage(ctx, tx, msg)
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

		err = repo.UpsertMessageMailbox(ctx, tx, insertedMsg.ID, mboxID, isUnread, isDraft, isTrashed, isSent, nil)
		if err != nil {
			return fmt.Errorf("upsert message mailbox: %w", err)
		}
	}

	// 4. Update Thread Projection
	err = repo.UpdateThreadProjection(ctx, tx, res.ThreadID)
	if err != nil {
		return fmt.Errorf("update thread projection: %w", err)
	}

	// 5. Out-of-order Thread Merge Reconciliation
	if msgID != "" {
		err = ReconcileOrphans(ctx, tx, repo, accountID, msgID, res.ThreadID)
		if err != nil {
			return fmt.Errorf("reconcile orphans: %w", err)
		}
	}

	return nil
}

// ReconcileOrphans finds existing messages that replied to msgID but were put into a different thread,
// and merges them into canonicalThreadID.
func ReconcileOrphans(ctx context.Context, tx pgx.Tx, repo *db.MailRepository, accountID, msgID, canonicalThreadID string) error {
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
		err = repo.UpdateThreadProjection(ctx, tx, canonicalThreadID)
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
