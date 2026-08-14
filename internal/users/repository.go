package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles user persistence in PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new user repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetByID retrieves a user by their UUID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, status, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return &u, nil
}

// GetByEmail retrieves a user by their normalized email.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, status, created_at, updated_at
		 FROM users WHERE email = lower($1)`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}
	return &u, nil
}

// UpdateStatus updates a user's status.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("updating user status: %w", err)
	}
	return nil
}
