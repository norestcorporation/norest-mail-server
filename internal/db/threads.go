package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/norest-mail/server/internal/mail"
)

// Thread represents the Norest-owned conversation state.
type Thread struct {
	ID            string    `json:"id"`
	AccountID     string    `json:"account_id"`
	Subject       string    `json:"subject"`
	Participants  []byte    `json:"participants"`
	MessageCount  int       `json:"message_count"`
	UnreadCount   int       `json:"unread_count"`
	Snippet       *string   `json:"snippet"`
	LastMessageAt time.Time `json:"last_message_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateThread inserts a new thread projection.
func (p *MailRepository) CreateThread(ctx context.Context, tx pgx.Tx, accountID, subject string) (*Thread, error) {
	query := `
		INSERT INTO threads (account_id, subject)
		VALUES ($1, $2)
		RETURNING id, account_id, subject, participants, message_count, unread_count, snippet, last_message_at, created_at, updated_at
	`
	var t Thread
	err := tx.QueryRow(ctx, query, accountID, subject).Scan(
		&t.ID, &t.AccountID, &t.Subject, &t.Participants, &t.MessageCount, &t.UnreadCount, &t.Snippet, &t.LastMessageAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating thread: %w", err)
	}
	return &t, nil
}

// GetThread retrieves a single thread by ID.
func (p *MailRepository) GetThread(ctx context.Context, tx pgx.Tx, id string) (*Thread, error) {
	query := `
		SELECT id, account_id, subject, participants, message_count, unread_count, snippet, last_message_at, created_at, updated_at
		FROM threads
		WHERE id = $1
	`
	var t Thread
	err := tx.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.AccountID, &t.Subject, &t.Participants, &t.MessageCount, &t.UnreadCount, &t.Snippet, &t.LastMessageAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("thread not found")
		}
		return nil, fmt.Errorf("getting thread: %w", err)
	}
	return &t, nil
}

// UpdateThreadProjection recalculates and updates the thread denormalized fields.
func (p *MailRepository) UpdateThreadProjection(ctx context.Context, tx pgx.Tx, threadID string) error {
	// Calculate active message count (excluding messages only in trash)
	// Calculate unread count (excluding messages only in trash)
	// Calculate participants
	// Calculate snippet and last_message_at
	// For now, this is a placeholder for the complex query.
	query := `
		WITH thread_messages AS (
			SELECT m.id, m.subject, m.sender, m.received_at, m.sent_at,
			       bool_and(mm.is_trashed) as all_trashed,
			       bool_or(mm.is_unread) as any_unread
			FROM messages m
			JOIN message_mailboxes mm ON m.id = mm.message_id
			WHERE m.thread_id = $1
			GROUP BY m.id
		),
		active_messages AS (
			SELECT * FROM thread_messages WHERE all_trashed = false
		),
		counts AS (
			SELECT count(*) as msg_count, 
			       count(*) FILTER (WHERE any_unread = true) as unread_cnt,
			       max(COALESCE(received_at, sent_at)) as last_msg_at
			FROM active_messages
		)
		UPDATE threads t
		SET message_count = COALESCE((SELECT msg_count FROM counts), 0),
		    unread_count = COALESCE((SELECT unread_cnt FROM counts), 0),
		    last_message_at = COALESCE((SELECT last_msg_at FROM counts), NOW()),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, threadID)
	if err != nil {
		return fmt.Errorf("updating thread projection: %w", err)
	}
	return nil
}

// GetThreadAccountScoped retrieves a single thread by ID, strictly enforcing account ownership.
func (p *MailRepository) GetThreadAccountScoped(ctx context.Context, accountID, threadID string) (*mail.ThreadData, error) {
	query := `
		SELECT id, account_id, subject, participants, message_count, unread_count, snippet, last_message_at, created_at, updated_at
		FROM threads
		WHERE id = $1 AND account_id = $2
	`
	var t mail.ThreadData
	err := p.pool.QueryRow(ctx, query, threadID, accountID).Scan(
		&t.ID, &t.AccountID, &t.Subject, &t.Participants, &t.MessageCount, &t.UnreadCount, &t.Snippet, &t.LastMessageAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("thread not found or unauthorized")
		}
		return nil, fmt.Errorf("getting thread: %w", err)
	}
	return &t, nil
}

// ListThreads retrieves paginated threads for an account, with optional mailbox filtering.
// Uses keyset pagination on (last_message_at DESC, id DESC).
func (p *MailRepository) ListThreads(ctx context.Context, accountID, mailboxID string, limit int, cursorTime *time.Time, cursorID string) ([]mail.ThreadData, error) {
	// Base query
	query := `
		SELECT t.id, t.account_id, t.subject, t.participants, t.message_count, t.unread_count, t.snippet, t.last_message_at, t.created_at, t.updated_at
		FROM threads t
	`

	args := []any{accountID, limit}

	// If mailbox is specified, we need to join messages and message_mailboxes
	// to ensure the thread has at least one message in this mailbox (that isn't entirely trashed, unless the mailbox IS trash)
	if mailboxID != "" {
		query += `
		INNER JOIN messages m ON m.thread_id = t.id
		INNER JOIN message_mailboxes mm ON mm.message_id = m.id
		WHERE t.account_id = $1 AND mm.stalwart_mailbox_id = $3
		`
		args = append(args, mailboxID)
	} else {
		query += ` WHERE t.account_id = $1 `
	}

	// Keyset pagination: (last_message_at, id) < (cursorTime, cursorID)
	if cursorTime != nil && cursorID != "" {
		query += fmt.Sprintf(` AND (t.last_message_at, t.id) < ($%d, $%d) `, len(args)+1, len(args)+2)
		args = append(args, *cursorTime, cursorID)
	}

	// Group by if we joined (to avoid duplicates for multiple messages in the same mailbox in the same thread)
	if mailboxID != "" {
		query += ` GROUP BY t.id `
	}

	query += ` ORDER BY t.last_message_at DESC, t.id DESC LIMIT $2 `

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}
	defer rows.Close()

	var threads []mail.ThreadData
	for rows.Next() {
		var t mail.ThreadData
		if err := rows.Scan(
			&t.ID, &t.AccountID, &t.Subject, &t.Participants, &t.MessageCount, &t.UnreadCount, &t.Snippet, &t.LastMessageAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning thread: %w", err)
		}
		threads = append(threads, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating threads: %w", err)
	}

	return threads, nil
}
