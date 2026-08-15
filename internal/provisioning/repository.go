package provisioning

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ClaimJob atomically claims a single job for a worker.
func (r *Repository) ClaimJob(ctx context.Context, workerID string, leaseDuration time.Duration) (*Job, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var j Job
	err := r.pool.QueryRow(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     worker_id = $2,
		     claimed_at = NOW(),
		     heartbeat_at = NOW(),
		     attempts = attempts + 1,
		     updated_at = NOW()
		 WHERE id IN (
			 SELECT id FROM provisioning_jobs
			 WHERE status IN ($3, $4, $5) AND next_attempt_at <= NOW()
			 ORDER BY next_attempt_at ASC
			 LIMIT 1
			 FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, type, resource_id, status, attempts, next_attempt_at, last_error, claimed_at, heartbeat_at, last_checkpoint_at, worker_id, created_at, updated_at`,
		StatusProcessing, workerID, StatusPending, StatusRetryWait, StatusUnknown,
	).Scan(&j.ID, &j.Type, &j.ResourceID, &j.Status, &j.Attempts, &j.NextAttemptAt, &j.LastError, &j.ClaimedAt, &j.HeartbeatAt, &j.LastCheckpointAt, &j.WorkerID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No jobs available to claim
			return nil, nil
		}
		return nil, fmt.Errorf("claiming job: %w", err)
	}
	return &j, nil
}

// UpdateHeartbeat updates the heartbeat timestamp for a job.
func (r *Repository) UpdateHeartbeat(ctx context.Context, id string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET heartbeat_at = NOW(), updated_at = NOW()
		 WHERE id = $1::uuid AND status = $2`,
		id, StatusProcessing,
	)
	return err
}

// RecoverStuckJobs reclaims jobs whose lease has expired.
func (r *Repository) RecoverStuckJobs(ctx context.Context, leaseDuration time.Duration) (int, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	cutoff := time.Now().Add(-leaseDuration)
	result, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     worker_id = NULL,
		     claimed_at = NULL,
		     heartbeat_at = NULL,
		     updated_at = NOW()
		 WHERE status = $2
		   AND heartbeat_at < $3`,
		StatusRetryWait, StatusProcessing, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("recovering stuck jobs: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// MarkSucceeded marks a job as succeeded.
func (r *Repository) MarkSucceeded(ctx context.Context, id uuid.UUID) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     worker_id = NULL,
		     claimed_at = NULL,
		     heartbeat_at = NULL,
		     updated_at = NOW()
		 WHERE id = $2`,
		StatusSucceeded, id,
	)
	return err
}

// MarkRetryWait marks a job for retry with a specific next attempt time.
func (r *Repository) MarkRetryWait(ctx context.Context, id uuid.UUID, nextAttempt time.Time, errMsg string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     next_attempt_at = $2,
		     last_error = $3,
		     worker_id = NULL,
		     claimed_at = NULL,
		     heartbeat_at = NULL,
		     updated_at = NOW()
		 WHERE id = $4`,
		StatusRetryWait, nextAttempt, errMsg, id,
	)
	return err
}

// MarkFailedOrRetry marks a job as failed, or queues a retry if under max attempts.
func (r *Repository) MarkFailedOrRetry(ctx context.Context, id uuid.UUID, maxAttempts int, errMsg string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var attempts int
	err := r.pool.QueryRow(queryCtx, `SELECT attempts FROM provisioning_jobs WHERE id = $1`, id).Scan(&attempts)
	if err != nil {
		return err
	}

	attempts++

	if attempts >= maxAttempts {
		_, err = r.pool.Exec(queryCtx,
			`UPDATE provisioning_jobs
			 SET status = $1,
			     attempts = $2,
			     last_error = $3,
			     worker_id = NULL,
			     claimed_at = NULL,
			     heartbeat_at = NULL,
			     updated_at = NOW()
			 WHERE id = $4`,
			StatusFailed, attempts, errMsg, id,
		)
	} else {
		// Exponential backoff
		backoff := time.Duration(1<<attempts) * time.Minute
		nextAttempt := time.Now().Add(backoff)

		_, err = r.pool.Exec(queryCtx,
			`UPDATE provisioning_jobs
			 SET status = $1,
			     attempts = $2,
			     next_attempt_at = $3,
			     last_error = $4,
			     worker_id = NULL,
			     claimed_at = NULL,
			     heartbeat_at = NULL,
			     updated_at = NOW()
			 WHERE id = $5`,
			StatusRetryWait, attempts, nextAttempt, errMsg, id,
		)
	}

	return err
}

// GetByID retrieves a provisioning job by its UUID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var j Job
	err := r.pool.QueryRow(queryCtx,
		`SELECT id, type, resource_id, status, attempts, next_attempt_at, last_error, claimed_at, heartbeat_at, last_checkpoint_at, worker_id, created_at, updated_at
		 FROM provisioning_jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.Type, &j.ResourceID, &j.Status, &j.Attempts, &j.NextAttemptAt, &j.LastError, &j.ClaimedAt, &j.HeartbeatAt, &j.LastCheckpointAt, &j.WorkerID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting job by id: %w", err)
	}
	return &j, nil
}

// MarkUnknown marks a job as UNKNOWN (uncertain state due to timeout/ambiguity).
func (r *Repository) MarkUnknown(ctx context.Context, id uuid.UUID, errMsg string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	nextAttempt := time.Now().Add(5 * time.Minute) // Reconcile after 5 minutes
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     next_attempt_at = $2,
		     last_error = $3,
		     worker_id = NULL,
		     claimed_at = NULL,
		     heartbeat_at = NULL,
		     updated_at = NOW()
		 WHERE id = $4`,
		StatusUnknown, nextAttempt, errMsg, id,
	)
	return err
}

// MarkReconciling marks a job as RECONCILING for state reconciliation.
func (r *Repository) MarkReconciling(ctx context.Context, id uuid.UUID) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     worker_id = NULL,
		     claimed_at = NULL,
		     heartbeat_at = NULL,
		     updated_at = NOW()
		 WHERE id = $2`,
		StatusReconciling, id,
	)
	return err
}

// MarkRepairing marks a job as REPAIRING for corrective action.
func (r *Repository) MarkRepairing(ctx context.Context, id uuid.UUID, errMsg string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	nextAttempt := time.Now().Add(10 * time.Minute) // Repair after 10 minutes
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET status = $1,
		     next_attempt_at = $2,
		     last_error = $3,
		     worker_id = NULL,
		     claimed_at = NULL,
		     heartbeat_at = NULL,
		     updated_at = NOW()
		 WHERE id = $4`,
		StatusRepairing, nextAttempt, errMsg, id,
	)
	return err
}

// UpdateCheckpoint updates the last checkpoint timestamp for a job.
func (r *Repository) UpdateCheckpoint(ctx context.Context, id uuid.UUID) error {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(queryCtx,
		`UPDATE provisioning_jobs
		 SET last_checkpoint_at = NOW(), updated_at = NOW()
		 WHERE id = $1::uuid AND status = $2`,
		id, StatusProcessing,
	)
	return err
}

// GetBacklogStats returns statistics about the provisioning job backlog.
func (r *Repository) GetBacklogStats(ctx context.Context) (pending, processing, retry, failed int64, oldestPending time.Time, err error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	// Get counts by status
	rows, err := r.pool.Query(queryCtx, `
		SELECT status, COUNT(*), MIN(created_at)
		FROM provisioning_jobs
		GROUP BY status
	`)
	if err != nil {
		return 0, 0, 0, 0, time.Time{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		var oldest sql.NullTime
		if err := rows.Scan(&status, &count, &oldest); err != nil {
			return 0, 0, 0, 0, time.Time{}, err
		}

		switch status {
		case "PENDING":
			pending = count
			if oldest.Valid && (oldestPending.IsZero() || oldest.Time.Before(oldestPending)) {
				oldestPending = oldest.Time
			}
		case "PROCESSING":
			processing = count
		case "RETRY_WAIT":
			retry = count
		case "FAILED":
			failed = count
		}
	}

	return pending, processing, retry, failed, oldestPending, nil
}
