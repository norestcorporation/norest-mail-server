package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/mail"
)

type MailRepository struct {
	pool *pgxpool.Pool
}

func NewMailRepository(pool *pgxpool.Pool) *MailRepository {
	return &MailRepository{pool: pool}
}

// GetMailboxByUserID retrieves the primary mailbox for a user.
// For platform domains, this queries through addresses claimed by the user (works for both user-owned and platform domains).
func (p *MailRepository) GetMailboxByUserID(ctx context.Context, userID string) (mail.Mailbox, error) {
	// Query through addresses that are claimed by the user (works for both user-owned and platform domains)
	query := `
		SELECT m.id, m.address_id, m.status, m.stalwart_account_id
		FROM mailboxes m
		JOIN addresses a ON m.address_id = a.id
		WHERE a.claimed_by = $1
		LIMIT 1
	`
	var m mail.Mailbox
	err := p.pool.QueryRow(ctx, query, userID).Scan(
		&m.ID, &m.AddressID, &m.Status, &m.StalwartAccountID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return m, fmt.Errorf("no mailbox found for user")
		}
		// Try pgx error mapping since we are using pgxpool
		if err.Error() == "no rows in result set" {
			return m, fmt.Errorf("no mailbox found for user")
		}
		return m, fmt.Errorf("querying mailbox: %w", err)
	}
	return m, nil
}

func (p *MailRepository) GetAddressByID(ctx context.Context, id string) (mail.Address, error) {
	query := `
		SELECT id, local_part, domain_id, status
		FROM addresses
		WHERE id = $1
	`
	var a mail.Address
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.LocalPart, &a.DomainID, &a.Status,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return a, fmt.Errorf("address not found")
		}
		return a, fmt.Errorf("querying address: %w", err)
	}
	return a, nil
}

// GetMailboxMappingByRole retrieves the Stalwart mailbox ID for a specific role (e.g., "drafts", "sent", "inbox").
func (p *MailRepository) GetMailboxMappingByRole(ctx context.Context, mailboxID, role string) (string, error) {
	query := `
		SELECT stalwart_mailbox_id
		FROM mailbox_mappings
		WHERE mailbox_id = $1 AND role = $2
	`
	var stalwartMailboxID string
	err := p.pool.QueryRow(ctx, query, mailboxID, role).Scan(&stalwartMailboxID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", fmt.Errorf("mailbox mapping not found for role: %s", role)
		}
		return "", fmt.Errorf("querying mailbox mapping: %w", err)
	}
	return stalwartMailboxID, nil
}

// GetSyncState retrieves the current sync state for a mailbox.
func (p *MailRepository) GetSyncState(ctx context.Context, mailboxID string) (*mail.SyncState, error) {
	query := `
		SELECT state, last_synced_at, status, error_message
		FROM mail_sync_state
		WHERE mailbox_id = $1
	`
	var s mail.SyncState
	var lastSyncedAt sql.NullTime
	var errMsg sql.NullString
	var state sql.NullString

	err := p.pool.QueryRow(ctx, query, mailboxID).Scan(
		&state, &lastSyncedAt, &s.Status, &errMsg,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // No state yet
		}
		return nil, fmt.Errorf("querying sync state: %w", err)
	}

	if state.Valid {
		s.State = state.String
	}
	if lastSyncedAt.Valid {
		timeStr := lastSyncedAt.Time.Format(time.RFC3339)
		s.LastSyncedAt = &timeStr
	}
	if errMsg.Valid {
		s.ErrorMessage = &errMsg.String
	}

	return &s, nil
}

// UpdateSyncState updates or inserts the sync state.
func (p *MailRepository) UpdateSyncState(ctx context.Context, mailboxID, state, status, errMsg string) error {
	query := `
		INSERT INTO mail_sync_state (mailbox_id, state, last_synced_at, status, error_message, updated_at)
		VALUES ($1, $2, NOW(), $3, $4, NOW())
		ON CONFLICT (mailbox_id) DO UPDATE
		SET state = EXCLUDED.state,
		    last_synced_at = EXCLUDED.last_synced_at,
		    status = EXCLUDED.status,
		    error_message = EXCLUDED.error_message,
		    updated_at = NOW()
	`
	var errStr sql.NullString
	if errMsg != "" {
		errStr = sql.NullString{String: errMsg, Valid: true}
	}
	_, err := p.pool.Exec(ctx, query, mailboxID, state, status, errStr)
	if err != nil {
		return fmt.Errorf("updating sync state: %w", err)
	}
	return nil
}
