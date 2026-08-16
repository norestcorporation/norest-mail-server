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

// ReserveAddress reserves an address for a user using the database function for race safety.
func (r *Repository) ReserveAddress(ctx context.Context, domainID uuid.UUID, localPart string, userID uuid.UUID, durationHours int) (*Address, error) {
	// Normalize local part to lowercase
	normalized := strings.ToLower(strings.TrimSpace(localPart))
	
	var addressID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT reserve_address($1, $2, $3, $4)`,
		domainID, normalized, userID, durationHours,
	).Scan(&addressID)
	
	if err != nil {
		return nil, ErrAddressExists
	}

	return r.GetByID(ctx, addressID)
}

// ClaimAddress claims a reserved address using the database function.
func (r *Repository) ClaimAddress(ctx context.Context, addressID uuid.UUID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`SELECT claim_address($1, $2)`,
		addressID, userID,
	)
	return err
}

// CheckAddressAvailability checks if an address is available for reservation.
func (r *Repository) CheckAddressAvailability(ctx context.Context, domainID uuid.UUID, localPart string) (bool, error) {
	var available bool
	err := r.pool.QueryRow(ctx,
		`SELECT check_address_available($1, $2)`,
		domainID, localPart,
	).Scan(&available)
	
	return available, err
}

// BlockAddress blocks an address from being reserved.
func (r *Repository) BlockAddress(ctx context.Context, domainID uuid.UUID, localPart, reason string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO blocked_addresses (domain_id, local_part, reason) VALUES ($1, $2, $3)
		 ON CONFLICT (domain_id, lower(local_part)) DO UPDATE SET reason = $3`,
		domainID, localPart, reason,
	)
	return err
}

// UnblockAddress removes an address from the blocked list.
func (r *Repository) UnblockAddress(ctx context.Context, domainID uuid.UUID, localPart string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM blocked_addresses WHERE domain_id = $1 AND lower(local_part) = lower($2)`,
		domainID, localPart,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("address not found in blocked list")
	}
	return nil
}

// CleanExpiredReservations cleans up expired reservations using the database function.
func (r *Repository) CleanExpiredReservations(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT clean_expired_reservations()`).Scan(&count)
	return count, err
}

// GetByID returns an address by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Address, error) {
	var a Address
	err := r.pool.QueryRow(ctx,
		`SELECT id, domain_id, local_part, status, reserved_by, reserved_at, reserved_until, claimed_by, claimed_at, created_at, updated_at
		 FROM addresses WHERE id = $1`, id,
	).Scan(&a.ID, &a.DomainID, &a.LocalPart, &a.Status, &a.ReservedBy, &a.ReservedAt, &a.ReservedUntil, &a.ClaimedBy, &a.ClaimedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("address not found")
		}
		return nil, err
	}
	return &a, nil
}

// GetByDomainAndLocalPart returns an address by domain ID and local part.
func (r *Repository) GetByDomainAndLocalPart(ctx context.Context, domainID uuid.UUID, localPart string) (*Address, error) {
	var a Address
	err := r.pool.QueryRow(ctx,
		`SELECT id, domain_id, local_part, status, reserved_by, reserved_at, reserved_until, claimed_by, claimed_at, created_at, updated_at
		 FROM addresses WHERE domain_id = $1 AND lower(local_part) = lower($2)`, domainID, localPart,
	).Scan(&a.ID, &a.DomainID, &a.LocalPart, &a.Status, &a.ReservedBy, &a.ReservedAt, &a.ReservedUntil, &a.ClaimedBy, &a.ClaimedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("address not found")
		}
		return nil, err
	}
	return &a, nil
}

// CreateAddressTx creates a mailbox record for an existing address.
func (r *Repository) CreateMailboxForAddress(ctx context.Context, addressID uuid.UUID) error {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(txCtx)

	// Create mailbox record
	var mailboxID uuid.UUID
	err = tx.QueryRow(txCtx,
		`INSERT INTO mailboxes (address_id, status)
		 VALUES ($1, $2)
		 RETURNING id`,
		addressID, "provisioning",
	).Scan(&mailboxID)
	if err != nil {
		return err
	}

	// Insert provisioning job
	_, err = tx.Exec(txCtx,
		`INSERT INTO provisioning_jobs (type, resource_id, status)
		 VALUES ($1, $2, $3)`,
		"ACCOUNT_CREATE", mailboxID, "PENDING",
	)
	if err != nil {
		return err
	}

	return tx.Commit(txCtx)
}

func (r *Repository) ListByDomainID(ctx context.Context, domainID uuid.UUID) ([]Address, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, domain_id, local_part, status, reserved_by, reserved_at, reserved_until, claimed_by, claimed_at, created_at, updated_at
		 FROM addresses WHERE domain_id = $1`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []Address
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.DomainID, &a.LocalPart, &a.Status, &a.ReservedBy, &a.ReservedAt, &a.ReservedUntil, &a.ClaimedBy, &a.ClaimedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
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
