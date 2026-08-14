package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/users"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user with this email already exists")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (*users.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user users.User

	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(txCtx)

	err = tx.QueryRow(txCtx, `
		INSERT INTO users (email, password_hash, status)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, status, role, created_at, updated_at
	`, email, passwordHash, users.StatusPending).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "idx_users_email_unique") {
			return nil, ErrUserExists
		}
		return nil, err
	}

	// Create Product Account
	var productAccountID uuid.UUID
	err = tx.QueryRow(txCtx, `
		INSERT INTO product_accounts (status)
		VALUES ('ACTIVE')
		RETURNING id
	`).Scan(&productAccountID)
	if err != nil {
		return nil, err
	}

	// Link User and Product Account
	_, err = tx.Exec(txCtx, `
		INSERT INTO user_product_accounts (user_id, product_account_id, role)
		VALUES ($1, $2, 'owner')
	`, user.ID, productAccountID)
	if err != nil {
		return nil, err
	}

	// Create FREE Subscription
	// First get the FREE plan ID
	var freePlanID uuid.UUID
	err = tx.QueryRow(txCtx, `
		SELECT id FROM plans WHERE code = 'FREE'
	`).Scan(&freePlanID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(txCtx, `
		INSERT INTO subscriptions (product_account_id, plan_id, status)
		VALUES ($1, $2, 'ACTIVE')
	`, productAccountID, freePlanID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(txCtx); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*users.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user users.User

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, status, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*users.User, error) {
	var user users.User

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, status, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
