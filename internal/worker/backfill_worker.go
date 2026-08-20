package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/db"
	"github.com/norest-mail/server/internal/stalwart"
)

type BackfillWorker struct {
	pool       *pgxpool.Pool
	stalwart   *stalwart.Client
	repository *db.MailRepository
}

func NewBackfillWorker(pool *pgxpool.Pool, stalwart *stalwart.Client, repo *db.MailRepository) *BackfillWorker {
	return &BackfillWorker{
		pool:       pool,
		stalwart:   stalwart,
		repository: repo,
	}
}

// RunBackfillBatch processes a single batch of emails for a given account.
// It is safely resumable and idempotent.
func (w *BackfillWorker) RunBackfillBatch(ctx context.Context, accountID string, stalwartAccountID string, limit int) (bool, error) {
	// 1. Fetch current backfill position
	var position *string
	err := w.pool.QueryRow(ctx, "SELECT backfill_position FROM sync_checkpoints WHERE account_id = $1", accountID).Scan(&position)
	if err != nil && err != pgx.ErrNoRows {
		return false, fmt.Errorf("getting backfill position: %w", err)
	}

	// 2. Fetch Email/query from Stalwart
	posInt := 0
	if position != nil && *position != "" {
		fmt.Sscanf(*position, "%d", &posInt)
	}

	queryResp, err := w.stalwart.EmailQueryWithPosition(ctx, stalwartAccountID, nil, nil, posInt, limit)
	if err != nil {
		return false, fmt.Errorf("email query: %w", err)
	}

	if len(queryResp.IDs) == 0 {
		// Complete
		return true, nil
	}

	// 3. Fetch full Email details
	emails, err := w.stalwart.EmailGet(ctx, stalwartAccountID, queryResp.IDs, nil)
	if err != nil {
		return false, fmt.Errorf("email get: %w", err)
	}

	// Process emails in a transaction
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	repo := w.repository
	for _, e := range emails.List {
		err = ProcessEmailSync(ctx, tx, repo, accountID, &e)
		if err != nil {
			return false, fmt.Errorf("process email %s: %w", e.ID, err)
		}
	}

	// Update checkpoint
	newPos := posInt + len(queryResp.IDs)
	newPosStr := fmt.Sprintf("%d", newPos)
	_, err = tx.Exec(ctx, `
		INSERT INTO sync_checkpoints (account_id, backfill_position, status, updated_at)
		VALUES ($1, $2, 'backfilling', NOW())
		ON CONFLICT (account_id) DO UPDATE SET 
			backfill_position = EXCLUDED.backfill_position,
			status = 'backfilling',
			updated_at = NOW()
	`, accountID, newPosStr)
	if err != nil {
		return false, fmt.Errorf("update checkpoint: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}

	slog.Info("Processed backfill batch", "account_id", accountID, "count", len(queryResp.IDs), "new_position", newPos)

	return false, nil // returning false means there might be more
}
