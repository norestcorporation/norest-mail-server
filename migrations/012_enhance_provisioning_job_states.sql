-- 012_enhance_provisioning_job_states.sql
-- Enhanced provisioning job states for better error handling and reconciliation

-- Update provisioning jobs table to support enhanced states
ALTER TABLE provisioning_jobs 
ALTER COLUMN status DROP DEFAULT,
ALTER COLUMN status SET NOT NULL;

-- Drop the old check constraint
ALTER TABLE provisioning_jobs DROP CONSTRAINT IF EXISTS provisioning_jobs_status_check;

-- Add new enhanced check constraint with additional states
ALTER TABLE provisioning_jobs 
ADD CONSTRAINT provisioning_jobs_status_check 
CHECK (status IN ('PENDING', 'PROCESSING', 'UNKNOWN', 'RECONCILING', 'REPAIRING', 'SUCCEEDED', 'FAILED', 'RETRY_WAIT'));

-- Add new columns for better job tracking
ALTER TABLE provisioning_jobs 
ADD COLUMN IF NOT EXISTS worker_id TEXT,
ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS last_checkpoint_at TIMESTAMPTZ;

-- Create indexes for better job management
CREATE INDEX IF NOT EXISTS idx_provisioning_jobs_worker 
ON provisioning_jobs (worker_id, status) 
WHERE status = 'PROCESSING';

CREATE INDEX IF NOT EXISTS idx_provisioning_jobs_heartbeat 
ON provisioning_jobs (heartbeat_at) 
WHERE status = 'PROCESSING';

CREATE INDEX IF NOT EXISTS idx_provisioning_jobs_unknown 
ON provisioning_jobs (status, next_attempt_at) 
WHERE status = 'UNKNOWN';

CREATE INDEX IF NOT EXISTS idx_provisioning_jobs_reconciling 
ON provisioning_jobs (status, updated_at) 
WHERE status = 'RECONCILING';