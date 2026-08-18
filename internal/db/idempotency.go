package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/mail"
)

type IdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

// StartIdempotentRequest acquires a lock on the idempotency key.
// If the key doesn't exist, it inserts it with IN_PROGRESS and returns (nil, nil).
// If the key exists and matches, it returns the cached state (or ErrIdempotencyInProgress).
// If the key exists but the hash mismatches, it returns ErrIdempotencyMismatch.
func (r *IdempotencyRepository) StartIdempotentRequest(ctx context.Context, userID, idempotencyKey, requestHash string) (*mail.IdempotentState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingHash, status string
	var responseCode sql.NullInt32
	var responseBody []byte

	// SELECT FOR UPDATE locks the row, preventing concurrent identical requests from proceeding
	err = tx.QueryRow(ctx, `
		SELECT request_hash, status, response_code, response_body 
		FROM send_idempotency_keys 
		WHERE user_id = $1 AND idempotency_key = $2 
		FOR UPDATE
	`, userID, idempotencyKey).Scan(&existingHash, &status, &responseCode, &responseBody)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Doesn't exist, insert as IN_PROGRESS
			_, err = tx.Exec(ctx, `
				INSERT INTO send_idempotency_keys (user_id, idempotency_key, request_hash, status, expires_at) 
				VALUES ($1, $2, $3, 'IN_PROGRESS', NOW() + INTERVAL '30 days')
			`, userID, idempotencyKey, requestHash)
			if err != nil {
				return nil, fmt.Errorf("insert idempotency key: %w", err)
			}

			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit idempotency key: %w", err)
			}

			return nil, nil // Caller should proceed with the operation
		}
		return nil, fmt.Errorf("query idempotency key: %w", err)
	}

	// Key exists. Commit the transaction early since we only needed to read the state.
	// Actually, wait, if it's IN_PROGRESS, another request is working on it.
	// We can commit now.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	if existingHash != requestHash {
		return nil, mail.ErrIdempotencyMismatch
	}

	if status == "IN_PROGRESS" {
		return nil, mail.ErrIdempotencyInProgress
	}

	return &mail.IdempotentState{
		Status:       status,
		ResponseCode: int(responseCode.Int32),
		ResponseBody: responseBody,
	}, nil
}

// CompleteIdempotentRequest marks the request as COMPLETED or AMBIGUOUS and saves the response.
func (r *IdempotencyRepository) CompleteIdempotentRequest(ctx context.Context, userID, idempotencyKey, status string, responseCode int, responseBody []byte) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE send_idempotency_keys 
		SET status = $1, response_code = $2, response_body = $3, completed_at = NOW(), updated_at = NOW() 
		WHERE user_id = $4 AND idempotency_key = $5
	`, status, responseCode, responseBody, userID, idempotencyKey)
	return err
}

// ClearIdempotentRequest removes the key, allowing the client to safely retry.
func (r *IdempotencyRepository) ClearIdempotentRequest(ctx context.Context, userID, idempotencyKey string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM send_idempotency_keys 
		WHERE user_id = $1 AND idempotency_key = $2
	`, userID, idempotencyKey)
	return err
}
