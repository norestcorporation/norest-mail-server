package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/norest-mail/server/internal/mail"
)

// Message represents the Norest-owned message projection.
type Message struct {
	ID              string     `json:"id"`
	AccountID       string     `json:"account_id"`
	ThreadID        string     `json:"thread_id"`
	StalwartEmailID string     `json:"stalwart_email_id"`
	MessageID       *string    `json:"message_id"`
	InReplyTo       *string    `json:"in_reply_to"`
	ReferencesChain []string   `json:"references_chain"`
	Subject         *string    `json:"subject"`
	Sender          []byte     `json:"sender"`
	Recipients      []byte     `json:"recipients"`
	ReceivedAt      *time.Time `json:"received_at"`
	SentAt          *time.Time `json:"sent_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UpsertMessage inserts or updates a message projection.
func (p *MailRepository) UpsertMessage(ctx context.Context, tx pgx.Tx, m *Message) (*Message, error) {
	query := `
		INSERT INTO messages (
			account_id, thread_id, stalwart_email_id, message_id, in_reply_to, 
			references_chain, subject, sender, recipients, received_at, sent_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (account_id, stalwart_email_id) DO UPDATE SET
			thread_id = EXCLUDED.thread_id,
			message_id = EXCLUDED.message_id,
			in_reply_to = EXCLUDED.in_reply_to,
			references_chain = EXCLUDED.references_chain,
			subject = EXCLUDED.subject,
			sender = EXCLUDED.sender,
			recipients = EXCLUDED.recipients,
			received_at = EXCLUDED.received_at,
			sent_at = EXCLUDED.sent_at,
			updated_at = NOW()
		RETURNING id, account_id, thread_id, stalwart_email_id, message_id, in_reply_to, references_chain, subject, sender, recipients, received_at, sent_at, created_at, updated_at
	`
	var res Message
	err := tx.QueryRow(ctx, query,
		m.AccountID, m.ThreadID, m.StalwartEmailID, m.MessageID, m.InReplyTo,
		m.ReferencesChain, m.Subject, m.Sender, m.Recipients, m.ReceivedAt, m.SentAt,
	).Scan(
		&res.ID, &res.AccountID, &res.ThreadID, &res.StalwartEmailID, &res.MessageID,
		&res.InReplyTo, &res.ReferencesChain, &res.Subject, &res.Sender, &res.Recipients,
		&res.ReceivedAt, &res.SentAt, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting message: %w", err)
	}
	return &res, nil
}

// UpsertMessageMailbox creates or updates the mailbox membership relationship.
func (p *MailRepository) UpsertMessageMailbox(ctx context.Context, tx pgx.Tx, messageID, stalwartMailboxID string, isUnread, isDraft, isTrashed, isSent bool, keywords []string) error {
	query := `
		INSERT INTO message_mailboxes (
			message_id, stalwart_mailbox_id, is_unread, is_draft, is_trashed, is_sent, keywords
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (message_id, stalwart_mailbox_id) DO UPDATE SET
			is_unread = EXCLUDED.is_unread,
			is_draft = EXCLUDED.is_draft,
			is_trashed = EXCLUDED.is_trashed,
			is_sent = EXCLUDED.is_sent,
			keywords = EXCLUDED.keywords
	`
	_, err := tx.Exec(ctx, query, messageID, stalwartMailboxID, isUnread, isDraft, isTrashed, isSent, keywords)
	if err != nil {
		return fmt.Errorf("upserting message mailbox: %w", err)
	}
	return nil
}

// GetMessagesByThread retrieves all messages for a specific thread, ordered chronologically.
func (p *MailRepository) GetMessagesByThread(ctx context.Context, accountID, threadID string) ([]mail.MessageData, error) {
	query := `
		SELECT m.id, m.account_id, m.thread_id, m.stalwart_email_id, m.message_id, 
		       m.in_reply_to, m.references_chain, m.subject, m.sender, m.recipients, 
		       m.received_at, m.sent_at, m.created_at, m.updated_at
		FROM messages m
		WHERE m.thread_id = $1 AND m.account_id = $2
		ORDER BY COALESCE(m.received_at, m.sent_at, m.created_at) ASC, m.id ASC
	`
	rows, err := p.pool.Query(ctx, query, threadID, accountID)
	if err != nil {
		return nil, fmt.Errorf("getting messages by thread: %w", err)
	}
	defer rows.Close()

	var messages []mail.MessageData
	for rows.Next() {
		var m mail.MessageData
		if err := rows.Scan(
			&m.ID, &m.AccountID, &m.ThreadID, &m.StalwartEmailID, &m.MessageID,
			&m.InReplyTo, &m.ReferencesChain, &m.Subject, &m.Sender, &m.Recipients,
			&m.ReceivedAt, &m.SentAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating messages: %w", err)
	}

	return messages, nil
}

// GetThreadIDByStalwartID returns the Norest thread ID for a given Stalwart Email ID.
func (p *MailRepository) GetThreadIDByStalwartID(ctx context.Context, accountID, stalwartEmailID string) (string, error) {
var threadID string
query := `SELECT thread_id FROM messages WHERE account_id = $1 AND stalwart_email_id = $2`
err := p.pool.QueryRow(ctx, query, accountID, stalwartEmailID).Scan(&threadID)
if err != nil {
return "", fmt.Errorf("getting thread_id by stalwart id: %w", err)
}
return threadID, nil
}
