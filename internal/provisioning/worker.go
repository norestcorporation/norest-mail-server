package provisioning

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/norest-mail/server/internal/metrics"
	"github.com/norest-mail/server/internal/stalwart"
)

type Worker struct {
	repo            *Repository
	stalwartClient   *stalwart.Client
	pool            *pgxpool.Pool
	workerID        string
	leaseDuration   time.Duration
	heartbeatInterval time.Duration
	maxAttempts     int
	retryPolicy     *RetryPolicy
	interval        time.Duration
	shutdownTimeout time.Duration
	mu              sync.Mutex
	shuttingDown    bool
	jobSemaphore    chan struct{} // Limits concurrent job processing
}

func NewWorker(pool *pgxpool.Pool, stalwartClient *stalwart.Client, workerID string, leaseSeconds, heartbeatSeconds, maxAttempts, maxBackoffSeconds int) *Worker {
	leaseDuration := time.Duration(leaseSeconds) * time.Second
	heartbeatInterval := time.Duration(heartbeatSeconds) * time.Second

	retryPolicy := &RetryPolicy{
		MaxAttempts:      maxAttempts,
		InitialBackoff:   2 * time.Second,
		MaxBackoff:       time.Duration(maxBackoffSeconds) * time.Second,
		JitterFraction:   0.25,
	}

	return &Worker{
		repo:            NewRepository(pool),
		stalwartClient:   stalwartClient,
		pool:            pool,
		workerID:        workerID,
		leaseDuration:   leaseDuration,
		heartbeatInterval: heartbeatInterval,
		maxAttempts:     maxAttempts,
		retryPolicy:     retryPolicy,
		interval:        5 * time.Second,
		shutdownTimeout: 30 * time.Second,
		jobSemaphore:    make(chan struct{}, 5), // Limit to 5 concurrent jobs
	}
}

func (w *Worker) Run(ctx context.Context) error {
	slog.Info("provisioning worker started", "worker_id", w.workerID, "lease_duration", w.leaseDuration, "heartbeat_interval", w.heartbeatInterval)

	// Startup recovery: reclaim stuck jobs
	if err := w.recoverStuckJobs(ctx); err != nil {
		slog.Error("failed to recover stuck jobs on startup", "error", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("provisioning worker stopping", "worker_id", w.workerID)
			return w.gracefulShutdown(ctx)
		case <-ticker.C:
			// Update backlog metrics periodically
			pending, processing, retry, failed, oldest, err := w.repo.GetBacklogStats(ctx)
			if err == nil {
				metrics.SetProvisioningBacklog(pending, processing, retry, failed, oldest)
			}
			// Clean up expired reservations periodically
			w.cleanupExpiredReservations(ctx)
			w.poll(ctx)
		}
	}
}

func (w *Worker) gracefulShutdown(ctx context.Context) error {
	w.mu.Lock()
	w.shuttingDown = true
	w.mu.Unlock()

	slog.Info("worker graceful shutdown initiated", "worker_id", w.workerID)

	// Allow time for current jobs to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), w.shutdownTimeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownCtx.Done():
			slog.Info("worker shutdown timeout reached", "worker_id", w.workerID)
			return nil
		case <-ticker.C:
			// Check if there are still processing jobs owned by this worker
			count, err := w.countProcessingJobs(ctx)
			if err != nil {
				slog.Error("failed to check processing jobs during shutdown", "error", err)
				return err
			}
			if count == 0 {
				slog.Info("all jobs completed, shutting down", "worker_id", w.workerID)
				return nil
			}
			slog.Info("waiting for jobs to complete", "worker_id", w.workerID, "remaining", count)
		}
	}
}

func (w *Worker) countProcessingJobs(ctx context.Context) (int, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var count int
	err := w.pool.QueryRow(queryCtx,
		`SELECT COUNT(*) FROM provisioning_jobs WHERE status = $1 AND worker_id = $2`,
		StatusProcessing, w.workerID,
	).Scan(&count)
	return count, err
}

func (w *Worker) recoverStuckJobs(ctx context.Context) error {
	count, err := w.repo.RecoverStuckJobs(ctx, w.leaseDuration)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("recovered stuck jobs",
			"service", "norest-worker",
			"worker_id", w.workerID,
			"count", count,
		)
		// Record reclaimed jobs metric
		for i := int64(0); i < int64(count); i++ {
			metrics.JobReclaimed()
		}
	}
	return nil
}

func (w *Worker) cleanupExpiredReservations(ctx context.Context) {
	// This function periodically cleans up expired address reservations
	// so they become available again for other users
	count, err := w.repo.CleanExpiredReservations(ctx)
	if err != nil {
		slog.Error("failed to clean expired reservations", "worker_id", w.workerID, "error", err)
		return
	}
	if count > 0 {
		slog.Info("cleaned expired reservations", "worker_id", w.workerID, "count", count)
	}
}

func (w *Worker) poll(ctx context.Context) {
	w.mu.Lock()
	if w.shuttingDown {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	// Apply backpressure: if we're already processing max concurrent jobs, skip polling
	select {
	case w.jobSemaphore <- struct{}{}:
		// Got semaphore slot, proceed
	default:
		// At capacity, skip this poll cycle
		slog.Debug("worker at capacity, skipping poll", "worker_id", w.workerID, "capacity", cap(w.jobSemaphore))
		return
	}

	// Claim one job at a time (we'll spawn goroutines for multiple workers)
	job, err := w.repo.ClaimJob(ctx, w.workerID, w.leaseDuration)
	if err != nil {
		slog.Error("failed to claim job", "worker_id", w.workerID, "error", err)
		<-w.jobSemaphore // Release semaphore
		return
	}
	if job == nil {
		<-w.jobSemaphore // Release semaphore
		return // No jobs available
	}

	slog.Info("processing job",
		"service", "norest-worker",
		"worker_id", w.workerID,
		"job_id", job.ID,
		"job_type", job.Type,
		"resource_id", job.ResourceID,
		"attempt", job.Attempts,
	)

	// Record job started metric and start time
	metrics.JobStarted()
	start := time.Now()

	// Process job with timeout
	jobCtx, cancel := context.WithTimeout(ctx, w.leaseDuration)
	defer cancel()

	// Start heartbeat goroutine
	heartbeatDone := make(chan struct{})
	go w.heartbeat(jobCtx, job.ID.String(), heartbeatDone)

	// Process the job
	err = w.processJob(jobCtx, job)

	// Stop heartbeat
	close(heartbeatDone)

	// Update job status
	if err != nil {
		w.handleJobFailure(ctx, job, err)
	} else {
		w.handleJobSuccess(ctx, job)
	}

	// Record metrics for job duration
	duration := time.Since(start)
	metrics.JobSucceeded(duration)

	// Release semaphore
	<-w.jobSemaphore
}

func (w *Worker) heartbeat(ctx context.Context, jobID string, done <-chan struct{}) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := w.repo.UpdateHeartbeat(ctx, jobID); err != nil {
				slog.Error("failed to update heartbeat",
					"service", "norest-worker",
					"worker_id", w.workerID,
					"job_id", jobID,
					"error", err,
				)
			}
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *Job) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		slog.Info("job processing completed",
			"service", "norest-worker",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"job_type", job.Type,
			"resource_id", job.ResourceID,
			"attempt", job.Attempts,
			"duration", duration.Milliseconds(),
		)
	}()

	switch job.Type {
	case JobTypeDomainCreate:
		return w.processDomainCreate(ctx, job)
	case JobTypeAccountCreate:
		return w.processAccountCreate(ctx, job)
	case "DOMAIN_VERIFY":
		return w.processDomainVerify(ctx, *job)
	case "ACCOUNT_QUOTA_SYNC", "ACCOUNT_SUSPEND", "ACCOUNT_REACTIVATE":
		return w.processAccountQuotaSync(ctx, *job)
	case JobTypeAccountDisable:
		return w.processAccountDisable(ctx, job)
	case JobTypeAccountEnable:
		return w.processAccountEnable(ctx, job)
	default:
		slog.Warn("unknown job type",
			"service", "norest-worker",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"job_type", job.Type,
		)
		return nil
	}
}

func (w *Worker) handleJobSuccess(ctx context.Context, job *Job) {
	if err := w.repo.MarkSucceeded(ctx, job.ID); err != nil {
		slog.Error("failed to mark job as succeeded", "job_id", job.ID, "error", err)
	}
	slog.Info("job succeeded",
		"service", "norest-worker",
		"worker_id", w.workerID,
		"job_id", job.ID,
		"job_type", job.Type,
	)
}

func (w *Worker) handleJobFailure(ctx context.Context, job *Job, err error) {
	errMsg := truncateError(err.Error(), 1000)
	
	// Record job failed metric
	metrics.JobFailed()
	
	// Classify error to determine if retryable
	errorType := w.classifyError(err)
	
	if !w.retryPolicy.IsRetryable(errorType) || job.Attempts >= w.maxAttempts {
		if markErr := w.repo.MarkFailedOrRetry(ctx, job.ID, w.maxAttempts, errMsg); markErr != nil {
			slog.Error("failed to mark job as failed", "job_id", job.ID, "error", markErr)
		}
	slog.Error("job failed permanently",
			"service", "norest-worker",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"job_type", job.Type,
			"error_code", errorType,
			"error", errMsg,
		)
	} else {
		backoff := w.retryPolicy.CalculateBackoff(job.Attempts)
		nextAttempt := time.Now().Add(backoff)
		if markErr := w.repo.MarkRetryWait(ctx, job.ID, nextAttempt, errMsg); markErr != nil {
			slog.Error("failed to mark job for retry", "job_id", job.ID, "error", markErr)
		}
		slog.Warn("job failed, will retry",
			"service", "norest-worker",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"job_type", job.Type,
			"attempt", job.Attempts,
			"next_attempt_at", nextAttempt,
			"error_code", errorType,
			"error", errMsg,
		)
		// Record job retried metric
		metrics.JobRetried()
	}
}

func (w *Worker) classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypePermanent
	}

	errStr := err.Error()
	
	// Check for timeout
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded") {
		return ErrorTypeTimeout
	}

	// Check for rate limiting
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return ErrorTypeRateLimited
	}

	// Check for conflicts
	if strings.Contains(errStr, "409") || strings.Contains(errStr, "conflict") || strings.Contains(errStr, "already exists") {
		return ErrorTypeConflict
	}

	// Check for not found
	if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
		return ErrorTypeNotFound
	}

	// Check for DNS errors
	if strings.Contains(errStr, "DNS") || strings.Contains(errStr, "lookup") {
		return ErrorTypeTemporary
	}

	// Default to permanent for unknown errors
	return ErrorTypePermanent
}

func truncateError(err string, maxLen int) string {
	if len(err) <= maxLen {
		return err
	}
	return err[:maxLen] + "... (truncated)"
}

func (w *Worker) processDomainCreate(ctx context.Context, job *Job) error {
	// 1. Get the domain with current state (with timeout)
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var domainName, status string
	var stalwartDomainID sql.NullString
	err := w.pool.QueryRow(queryCtx,
		`SELECT name, stalwart_domain_id, status FROM domains WHERE id = $1`,
		job.ResourceID,
	).Scan(&domainName, &stalwartDomainID, &status)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	stalwartDomainIDStr := ""
	if stalwartDomainID.Valid {
		stalwartDomainIDStr = stalwartDomainID.String
	}

	// 2. Idempotency check: if already has stalwart_domain_id, verify it exists AND matches expected name
	if stalwartDomainIDStr != "" {
		// Verify domain still exists in Stalwart AND has the correct name
		exists, err := w.stalwartClient.DomainExistsAndMatches(ctx, stalwartDomainIDStr, domainName)
		if err == nil && exists {
			slog.Info("domain already exists in Stalwart with correct name, skipping creation",
				"job_id", job.ID,
				"domain_id", stalwartDomainIDStr,
				"domain_name", domainName,
			)
			return nil
		}
		// If verification fails or name doesn't match, continue to create
		slog.Info("domain ID mismatch or not found, will recreate",
			"job_id", job.ID,
			"existing_id", stalwartDomainIDStr,
			"expected_name", domainName,
		)
	}

	// 2.5. Check if domain already exists in Stalwart by name (prevent duplicate creation)
	existingStalwartID, err := w.stalwartClient.FindDomainByName(ctx, domainName)
	if err == nil && existingStalwartID != "" {
		slog.Info("domain already exists in Stalwart by name, using existing ID",
			"job_id", job.ID,
			"domain_name", domainName,
			"existing_stalwart_id", existingStalwartID,
		)
		stalwartDomainIDStr = existingStalwartID
		// Update the domain record with the existing Stalwart ID
		_, err = w.pool.Exec(ctx, "UPDATE domains SET stalwart_domain_id = $1 WHERE id = $2", stalwartDomainIDStr, job.ResourceID)
		if err != nil {
			return fmt.Errorf("failed to update domain with existing Stalwart ID: %w", err)
		}
		// Mark job as successful
		_, err = w.pool.Exec(ctx, "UPDATE provisioning_jobs SET status = 'SUCCEEDED' WHERE id = $1", job.ID)
		if err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}
		return nil
	} else {
		// 3. Create domain in Stalwart
		stalwartDomainIDStr, err = w.stalwartClient.CreateDomain(ctx, domainName)
		if err != nil {
			return fmt.Errorf("failed to create domain in Stalwart: %w", err)
		}
		if stalwartDomainIDStr == "" {
			stalwartDomainIDStr = domainName // fallback
		}
	}

	// 4. Update domain status
	_, err = w.pool.Exec(ctx,
		`UPDATE domains SET status = $1, stalwart_domain_id = $2 WHERE id = $3`,
		"active", stalwartDomainIDStr, job.ResourceID,
	)
	return err
}

func (w *Worker) processDomainVerify(ctx context.Context, job Job) error {
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var domainName, tokenHash string
	err := w.pool.QueryRow(queryCtx, `SELECT name, verification_token_hash FROM domains WHERE id = $1`, job.ResourceID).Scan(&domainName, &tokenHash)
	if err != nil {
		return err
	}

	recordName := "_norest-verification." + domainName

	// Use Go's net package to lookup TXT records
	txts, err := net.LookupTXT(recordName)
	if err != nil {
		return fmt.Errorf("DNS lookup failed: %w", err)
	}

	expectedPrefix := "norest-verification="
	found := false
	for _, txt := range txts {
		if strings.HasPrefix(txt, expectedPrefix) {
			val := strings.TrimPrefix(txt, expectedPrefix)
			if val == tokenHash {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("verification record not found or token mismatch")
	}

	// Update domain to verified status first
	_, err = w.pool.Exec(ctx,
		`UPDATE domains SET verification_status = $1 WHERE id = $2`,
		"verified", job.ResourceID,
	)
	if err != nil {
		return err
	}

	// Check MX records for additional validation
	mxRecords, err := net.LookupMX(domainName)
	if err != nil {
		slog.Warn("MX record lookup failed during domain verification", "domain", domainName, "error", err)
		// MX record failure is not critical for ownership verification
		// Continue with activation
	} else if len(mxRecords) == 0 {
		slog.Warn("No MX records found for domain", "domain", domainName)
		// No MX records is acceptable for verification, but warn
	}

	// Now activate the domain and enable registration
	_, err = w.pool.Exec(ctx,
		`UPDATE domains SET status = $1, registration_enabled = $2 WHERE id = $3`,
		"active", true, job.ResourceID,
	)
	if err != nil {
		return err
	}

	// Now that it is verified and active, provision it in Stalwart!
	// Create DOMAIN_CREATE job
	_, err = w.pool.Exec(ctx,
		`INSERT INTO provisioning_jobs (type, resource_id, status) VALUES ($1, $2, $3)`,
		"DOMAIN_CREATE", job.ResourceID, "PENDING",
	)
	if err != nil {
		return fmt.Errorf("failed to create DOMAIN_CREATE job: %w", err)
	}

	slog.Info("domain verified and activated, Stalwart provisioning job created",
		"domain_id", job.ResourceID,
		"domain_name", domainName,
	)

	return nil
}

func (w *Worker) processAccountCreate(ctx context.Context, job *Job) error {
	// 1. Get the mailbox and address with current state (with timeout)
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var localPart, domainName, mailboxStatus string
	var stalwartDomainID, stalwartAccountID sql.NullString

	err := w.pool.QueryRow(queryCtx, `
		SELECT a.local_part, d.name, d.stalwart_domain_id, m.stalwart_account_id, m.status
		FROM mailboxes m
		JOIN addresses a ON m.address_id = a.id
		JOIN domains d ON a.domain_id = d.id
		WHERE m.id = $1
	`, job.ResourceID).Scan(&localPart, &domainName, &stalwartDomainID, &stalwartAccountID, &mailboxStatus)
	if err != nil {
		return fmt.Errorf("failed to get mailbox: %w", err)
	}

	accountName := localPart + "@" + domainName

	stalwartDomainIDStr := ""
	if stalwartDomainID.Valid {
		stalwartDomainIDStr = stalwartDomainID.String
	}

	stalwartAccountIDStr := ""
	if stalwartAccountID.Valid {
		stalwartAccountIDStr = stalwartAccountID.String
	}

	// 2. Ensure Stalwart domain exists before creating account
	if stalwartDomainIDStr == "" {
		slog.Info("Stalwart domain ID missing, attempting to ensure domain exists",
			"job_id", job.ID,
			"domain_name", domainName,
		)
		
		// Try to find existing domain by name
		existingStalwartID, err := w.stalwartClient.FindDomainByName(ctx, domainName)
		if err == nil && existingStalwartID != "" {
			stalwartDomainIDStr = existingStalwartID
			// Update domain record
			_, err = w.pool.Exec(ctx, "UPDATE domains SET stalwart_domain_id = $1 WHERE name = $2", stalwartDomainIDStr, domainName)
			if err != nil {
				return fmt.Errorf("failed to update domain with existing Stalwart ID: %w", err)
			}
			slog.Info("Found existing Stalwart domain", "domain_name", domainName, "stalwart_id", stalwartDomainIDStr)
		} else {
			// Create domain
			stalwartDomainIDStr, err = w.stalwartClient.CreateDomain(ctx, domainName)
			if err != nil {
				return fmt.Errorf("failed to create Stalwart domain: %w", err)
			}
			// Update domain record
			_, err = w.pool.Exec(ctx, "UPDATE domains SET stalwart_domain_id = $1 WHERE name = $2", stalwartDomainIDStr, domainName)
			if err != nil {
				return fmt.Errorf("failed to update domain with new Stalwart ID: %w", err)
			}
			slog.Info("Created new Stalwart domain", "domain_name", domainName, "stalwart_id", stalwartDomainIDStr)
		}
	}

	// 3. Idempotency check: if already has stalwart_account_id and is fully active, skip
	if stalwartAccountIDStr != "" && mailboxStatus == "active" {
		// Verify account still exists in Stalwart AND has the correct name/domain
		exists, err := w.stalwartClient.AccountExistsAndMatches(ctx, stalwartAccountIDStr, localPart, stalwartDomainIDStr)
		if err == nil && exists {
			slog.Info("account already exists in Stalwart with correct name/domain, skipping creation",
				"job_id", job.ID,
				"account_id", stalwartAccountIDStr,
				"local_part", localPart,
				"domain_id", stalwartDomainIDStr,
			)
			return nil
		}
		// If verification fails or info doesn't match, continue to create
		slog.Info("account ID mismatch or not found, will recreate",
			"job_id", job.ID,
			"existing_id", stalwartAccountIDStr,
			"expected_local_part", localPart,
			"expected_domain_id", stalwartDomainIDStr,
		)
	}

	// 4. Check if account already exists in Stalwart by name (prevent duplicate creation)
	existingStalwartAccountID, err := w.stalwartClient.FindAccountByName(ctx, accountName)
	if err == nil && existingStalwartAccountID != "" {
		slog.Info("account already exists in Stalwart by name, using existing ID",
			"job_id", job.ID,
			"account_name", accountName,
			"existing_stalwart_account_id", existingStalwartAccountID,
		)
		stalwartAccountIDStr = existingStalwartAccountID
	} else {
		// 5. Generate secure random password
		b := make([]byte, 32)
		rand.Read(b)
		password := base64.RawURLEncoding.EncodeToString(b)
		
		// 6. Create account in Stalwart
		stalwartAccountIDStr, err = w.stalwartClient.CreateAccount(ctx, localPart, stalwartDomainIDStr, password, accountName)
		if err != nil {
			return fmt.Errorf("failed to create account in Stalwart: %w", err)
		}
		if stalwartAccountIDStr == "" {
			stalwartAccountIDStr = accountName // fallback
		}
	}

	// 7. Update mailbox to provisioning state with Stalwart account ID
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err := w.pool.Begin(txCtx)
	if err != nil {
		return err
	}
	defer tx.Rollback(txCtx)

	_, err = tx.Exec(txCtx,
		`UPDATE mailboxes SET status = $1, stalwart_account_id = $2 WHERE id = $3`,
		"provisioning", stalwartAccountIDStr, job.ResourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update mailbox to provisioning: %w", err)
	}

	// Update address to CLAIMED (preserve the claimed_by from reservation)
	_, err = tx.Exec(txCtx,
		`UPDATE addresses SET status = $1, claimed_at = NOW(), claimed_by = COALESCE(claimed_by, reserved_by), reserved_by = NULL, reserved_at = NULL, reserved_until = NULL WHERE id = (SELECT address_id FROM mailboxes WHERE id = $2)`,
		"CLAIMED", job.ResourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update address to claimed: %w", err)
	}

	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("failed to commit provisioning state: %w", err)
	}

	// 8. Update job checkpoint for progress tracking
	if err := w.repo.UpdateCheckpoint(ctx, job.ID); err != nil {
		slog.Warn("failed to update job checkpoint", "error", err, "job_id", job.ID)
	}

	// 9. Discover and persist mailbox mappings
	slog.Info("discovering mailbox mappings", "job_id", job.ID, "account_id", stalwartAccountIDStr)
	mailboxMappings, err := w.stalwartClient.DiscoverMailboxes(ctx, stalwartAccountIDStr)
	if err != nil {
		return fmt.Errorf("failed to discover mailboxes: %w", err)
	}

	// Persist mailbox mappings
	txCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err = w.pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for mailbox mappings: %w", err)
	}
	defer tx.Rollback(txCtx)

	// Clear existing mappings for this mailbox
	_, err = tx.Exec(txCtx, "DELETE FROM mailbox_mappings WHERE mailbox_id = $1", job.ResourceID)
	if err != nil {
		return fmt.Errorf("failed to clear existing mailbox mappings: %w", err)
	}

	// Insert new mappings
	for role, stalwartMailboxID := range mailboxMappings {
		_, err = tx.Exec(txCtx,
			`INSERT INTO mailbox_mappings (mailbox_id, role, stalwart_mailbox_id) VALUES ($1, $2, $3)`,
			job.ResourceID, role, stalwartMailboxID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert mailbox mapping for role %s: %w", role, err)
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("failed to commit mailbox mappings: %w", err)
	}

	slog.Info("mailbox mappings persisted", "job_id", job.ID, "mappings_count", len(mailboxMappings))

	// 10. Update job checkpoint
	if err := w.repo.UpdateCheckpoint(ctx, job.ID); err != nil {
		slog.Warn("failed to update job checkpoint", "error", err, "job_id", job.ID)
	}

	// 11. Perform initial JMAP synchronization
	slog.Info("performing initial JMAP synchronization", "job_id", job.ID, "account_id", stalwartAccountIDStr)
	
	// Update mailbox status to syncing
	_, err = w.pool.Exec(ctx,
		`UPDATE mailboxes SET status = $1 WHERE id = $2`,
		"syncing", job.ResourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update mailbox to syncing: %w", err)
	}

	// Get JMAP session to retrieve initial state
	jmapSession, err := w.stalwartClient.GetJMAPSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to get JMAP session for initial sync: %w", err)
	}

	// The JMAP state represents the initial synchronization checkpoint
	initialState := jmapSession.State

	// 12. Persist initial synchronization checkpoint
	txCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	tx, err = w.pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for checkpoint: %w", err)
	}
	defer tx.Rollback(txCtx)

	_, err = tx.Exec(txCtx,
		`UPDATE mailboxes SET jmap_state = $1, initial_sync_checkpoint = $2, initial_sync_completed_at = NOW() WHERE id = $3`,
		initialState, initialState, job.ResourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to persist initial sync checkpoint: %w", err)
	}

	// 13. NOW set mailbox to active - only after complete initialization
	_, err = tx.Exec(txCtx,
		`UPDATE mailboxes SET status = $1 WHERE id = $2`,
		"active", job.ResourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update mailbox to active: %w", err)
	}

	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("failed to commit activation: %w", err)
	}

	slog.Info("mailbox activated after complete initialization", 
		"job_id", job.ID, 
		"mailbox_id", job.ResourceID,
		"jmap_state", initialState,
	)

	// 14. Update user status to active when first mailbox is successfully provisioned
	var userID sql.NullString
	err = w.pool.QueryRow(ctx,
		`SELECT a.claimed_by FROM addresses a 
		 JOIN mailboxes m ON a.id = m.address_id 
		 WHERE m.id = $1`,
		job.ResourceID,
	).Scan(&userID)
	if err != nil {
		slog.Warn("failed to get user_id for status update", "error", err, "mailbox_id", job.ResourceID)
		return nil // User status update is not critical for activation
	}

	if userID.Valid {
		result, err := w.pool.Exec(ctx,
			`UPDATE users SET status = 'active', updated_at = NOW() 
			 WHERE id = $1 AND status = 'pending'`,
			userID.String,
		)
		if err != nil {
			slog.Warn("failed to update user status to active", "error", err, "user_id", userID.String)
		} else {
			rowsAffected := result.RowsAffected()
			if rowsAffected > 0 {
				slog.Info("user status updated to active", "user_id", userID.String, "mailbox_id", job.ResourceID)
			}
		}
	}

	return nil
}

func (w *Worker) processAccountQuotaSync(ctx context.Context, job Job) error {
	// ResourceID is product_account_id
	// We need to fetch the plan for this product account and its status (with timeout)
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var maxStorageBytes int64
	var status string
	err := w.pool.QueryRow(queryCtx, `
		SELECT p.max_storage_bytes, a.status
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		JOIN product_accounts a ON s.product_account_id = a.id
		WHERE s.product_account_id = $1 AND s.status IN ('ACTIVE', 'TRIALING')
		ORDER BY s.created_at DESC LIMIT 1
	`, job.ResourceID).Scan(&maxStorageBytes, &status)
	if err != nil {
		return err
	}

	if status == "SUSPENDED" {
		maxStorageBytes = 1 // Restrict to 1 byte
	}

	// Fetch all Stalwart Account IDs for this product account
	rows, err := w.pool.Query(ctx, `
		SELECT m.stalwart_account_id
		FROM mailboxes m
		JOIN addresses a ON m.address_id = a.id
		JOIN domains d ON a.domain_id = d.id
		WHERE d.product_account_id = $1 AND m.stalwart_account_id IS NOT NULL
	`, job.ResourceID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var accountIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		accountIDs = append(accountIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Update Quota for each Stalwart Account
	for _, accountID := range accountIDs {
		if err := w.stalwartClient.UpdateAccountQuota(ctx, accountID, maxStorageBytes); err != nil {
			slog.Error("failed to update quota", "account_id", accountID, "error", err)
			return err
		}
	}

	return nil
}

func (w *Worker) processAccountDisable(ctx context.Context, job *Job) error {
	// 1. Get the mailbox with current state (with timeout)
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var stalwartAccountID sql.NullString
	var status string
	err := w.pool.QueryRow(queryCtx,
		`SELECT stalwart_account_id, status FROM mailboxes WHERE id = $1`,
		job.ResourceID,
	).Scan(&stalwartAccountID, &status)
	if err != nil {
		return fmt.Errorf("failed to get mailbox: %w", err)
	}

	// 2. Idempotency check: if already disabled, return success
	if status == "disabled" {
		slog.Info("account already disabled, skipping",
			"job_id", job.ID,
			"mailbox_id", job.ResourceID,
		)
		return nil
	}

	// 3. Disable account in Stalwart
	if stalwartAccountID.Valid && stalwartAccountID.String != "" {
		err = w.stalwartClient.DisableAccount(ctx, stalwartAccountID.String)
		if err != nil {
			return fmt.Errorf("failed to disable account in Stalwart: %w", err)
		}
	}

	// 4. Update mailbox status
	_, err = w.pool.Exec(ctx,
		`UPDATE mailboxes SET status = $1 WHERE id = $2`,
		"disabled", job.ResourceID,
	)
	return err
}

func (w *Worker) processAccountEnable(ctx context.Context, job *Job) error {
	// 1. Get the mailbox with current state (with timeout)
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var stalwartAccountID sql.NullString
	var status string
	err := w.pool.QueryRow(queryCtx,
		`SELECT stalwart_account_id, status FROM mailboxes WHERE id = $1`,
		job.ResourceID,
	).Scan(&stalwartAccountID, &status)
	if err != nil {
		return fmt.Errorf("failed to get mailbox: %w", err)
	}

	// 2. Idempotency check: if already enabled, return success
	if status == "active" {
		slog.Info("account already enabled, skipping",
			"job_id", job.ID,
			"mailbox_id", job.ResourceID,
		)
		return nil
	}

	// 3. Enable account in Stalwart
	if stalwartAccountID.Valid && stalwartAccountID.String != "" {
		err = w.stalwartClient.EnableAccount(ctx, stalwartAccountID.String)
		if err != nil {
			return fmt.Errorf("failed to enable account in Stalwart: %w", err)
		}
	}

	// 4. Update mailbox status
	_, err = w.pool.Exec(ctx,
		`UPDATE mailboxes SET status = $1 WHERE id = $2`,
		"active", job.ResourceID,
	)
	return err
}
