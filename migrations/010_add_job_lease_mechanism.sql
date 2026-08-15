-- 010_add_job_lease_mechanism.sql
-- Add lease and heartbeat fields for worker reliability and stuck job recovery

ALTER TABLE provisioning_jobs
ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS worker_id TEXT;

-- Update index to include lease expiry consideration
DROP INDEX IF EXISTS idx_provisioning_jobs_pending;
CREATE INDEX idx_provisioning_jobs_claimable ON provisioning_jobs (status, next_attempt_at)
    WHERE status IN ('PENDING', 'RETRY_WAIT');

-- Index for stuck job recovery
CREATE INDEX idx_provisioning_jobs_processing ON provisioning_jobs (status, heartbeat_at)
    WHERE status = 'PROCESSING';
