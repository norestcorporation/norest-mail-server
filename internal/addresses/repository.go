package addresses

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateAddressTx creates an address and mailbox, and a provisioning job atomically.
func (r *Repository) CreateAddressTx(ctx context.Context, domainID uuid.UUID, localPart string) (*Address, error) {
	return r.createAddressTxWithRetry(ctx, domainID, localPart, 0)
}

func (r *Repository) createAddressTxWithRetry(ctx context.Context, domainID uuid.UUID, localPart string, attempt int) (*Address, error) {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(txCtx)

	var a Address
	err = tx.QueryRow(txCtx,
		`INSERT INTO addresses (domain_id, local_part, status)
		 VALUES ($1, $2, $3)
		 RETURNING id, domain_id, local_part, status, created_at, updated_at`,
		domainID, localPart, StatusReserved,
	).Scan(&a.ID, &a.DomainID, &a.LocalPart, &a.Status, &a.CreatedAt, &a.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "idx_addresses_domain_local_unique") {
			return nil, ErrAddressExists
		}
		return nil, err
	}

	// Create mailbox record
	var mailboxID uuid.UUID
	err = tx.QueryRow(txCtx,
		`INSERT INTO mailboxes (address_id, status)
		 VALUES ($1, $2)
		 RETURNING id`,
		a.ID, "provisioning",
	).Scan(&mailboxID)
	if err != nil {
		return nil, err
	}

	// Insert provisioning job
	_, err = tx.Exec(txCtx,
		`INSERT INTO provisioning_jobs (type, resource_id, status)
		 VALUES ($1, $2, $3)`,
		"ACCOUNT_CREATE", mailboxID, "PENDING",
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(txCtx); err != nil {
		// Check for serialization errors and retry
		if pgconn.SafeToRetry(err) && attempt < 3 {
			slog.Warn("address creation serialization error, retrying", "attempt", attempt+1, "error", err)
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			return r.createAddressTxWithRetry(ctx, domainID, localPart, attempt+1)
		}
		return nil, err
	}

	return &a, nil
}

func (r *Repository) ListByDomainID(ctx context.Context, domainID uuid.UUID) ([]Address, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, domain_id, local_part, status, created_at, updated_at
		 FROM addresses WHERE domain_id = $1`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []Address
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.DomainID, &a.LocalPart, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if addresses == nil {
		addresses = []Address{}
	}
	return addresses, nil
}
