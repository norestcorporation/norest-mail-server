package db

import (
	"context"
	"database/sql"
	"fmt"

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
func (p *MailRepository) GetMailboxByUserID(ctx context.Context, userID string) (mail.Mailbox, error) {
	// For Chapter 3, we assume a user has one primary mailbox.
	// We get the first address associated with a domain they own (or simply join mailboxes to addresses to domains to users).
	query := `
		SELECT m.id, m.address_id, m.status, m.stalwart_account_id
		FROM mailboxes m
		JOIN addresses a ON m.address_id = a.id
		JOIN domains d ON a.domain_id = d.id
		WHERE d.user_id = $1
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
