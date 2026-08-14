package domains

import (
	"context"
	"errors"
	"log/slog"
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

// CreateDomainTx creates a domain and its provisioning job atomically.
func (r *Repository) CreateDomainTx(ctx context.Context, userID, productAccountID uuid.UUID, name string) (*Domain, error) {
	return r.createDomainTxWithRetry(ctx, userID, productAccountID, name, 0)
}

func (r *Repository) createDomainTxWithRetry(ctx context.Context, userID, productAccountID uuid.UUID, name string, attempt int) (*Domain, error) {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(txCtx)

	var d Domain
	err = tx.QueryRow(txCtx,
		`INSERT INTO domains (user_id, product_account_id, name, status, verification_status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, product_account_id, name, stalwart_domain_id, status, verification_status, created_at, updated_at`,
		userID, productAccountID, name, StatusPending, VerificationPending,
	).Scan(&d.ID, &d.UserID, &d.ProductAccountID, &d.Name, &d.StalwartDomainID, &d.Status, &d.VerificationStatus, &d.CreatedAt, &d.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || err.Error() == "ERROR: duplicate key value violates unique constraint \"idx_domains_name_unique\" (SQLSTATE 23505)" {
			return nil, ErrDomainExists
		}
		// Try to parse postgres duplicate error generically
		if err != nil && string(err.Error()) != "" {
			return nil, err
		}
		return nil, err
	}

	// Insert provisioning job
	_, err = tx.Exec(txCtx,
		`INSERT INTO provisioning_jobs (type, resource_id, status)
		 VALUES ($1, $2, $3)`,
		"DOMAIN_CREATE", d.ID, "PENDING",
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(txCtx); err != nil {
		// Check for serialization errors and retry
		if pgconn.SafeToRetry(err) && attempt < 3 {
			slog.Warn("domain creation serialization error, retrying", "attempt", attempt+1, "error", err)
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			return r.createDomainTxWithRetry(ctx, userID, productAccountID, name, attempt+1)
		}
		return nil, err
	}

	return &d, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Domain, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, product_account_id, name, stalwart_domain_id, status, verification_status, verification_token_hash, created_at, updated_at
		 FROM domains WHERE user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.UserID, &d.ProductAccountID, &d.Name, &d.StalwartDomainID, &d.Status, &d.VerificationStatus, &d.VerificationTokenHash, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil for JSON marshalling
	if domains == nil {
		domains = []Domain{}
	}
	return domains, nil
}

func (r *Repository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*Domain, error) {
	var d Domain
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, product_account_id, name, stalwart_domain_id, status, verification_status, verification_token_hash, created_at, updated_at
		 FROM domains WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&d.ID, &d.UserID, &d.ProductAccountID, &d.Name, &d.StalwartDomainID, &d.Status, &d.VerificationStatus, &d.VerificationTokenHash, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDomainNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *Repository) DeleteByIDAndUser(ctx context.Context, id, userID uuid.UUID) error {
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(txCtx)

	tag, err := tx.Exec(txCtx, `DELETE FROM domains WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrDomainNotFound
	}

	// Create delete job
	_, err = tx.Exec(txCtx,
		`INSERT INTO provisioning_jobs (type, resource_id, status)
		 VALUES ($1, $2, $3)`,
		"DOMAIN_DELETE", id, "PENDING",
	)
	if err != nil {
		return err
	}

	return tx.Commit(txCtx)
}

func (r *Repository) SetVerificationToken(ctx context.Context, domainID uuid.UUID, tokenHash string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE domains SET verification_token_hash = $1, verification_status = $2 WHERE id = $3`,
		tokenHash, VerificationVerifying, domainID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDomainNotFound
	}
	return nil
}

func (r *Repository) UpdateVerificationStatus(ctx context.Context, domainID uuid.UUID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE domains SET verification_status = $1 WHERE id = $2`,
		status, domainID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDomainNotFound
	}
	return nil
}

func (r *Repository) CreateVerificationJob(ctx context.Context, domainID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO provisioning_jobs (type, resource_id, status)
		 VALUES ($1, $2, $3)`,
		"DOMAIN_VERIFY", domainID, "PENDING",
	)
	return err
}
