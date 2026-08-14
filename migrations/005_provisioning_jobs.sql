-- 005_provisioning_jobs.sql
-- Asynchronous provisioning job queue backed by PostgreSQL.
-- No external message broker required.

CREATE TABLE provisioning_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            TEXT NOT NULL CHECK (type IN ('DOMAIN_CREATE', 'DOMAIN_DELETE', 'ACCOUNT_CREATE', 'ACCOUNT_DISABLE', 'ACCOUNT_ENABLE', 'ACCOUNT_DELETE')),
    resource_id     UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED', 'RETRY_WAIT')),
    attempts        INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for the worker polling query
CREATE INDEX idx_provisioning_jobs_pending ON provisioning_jobs (status, next_attempt_at)
    WHERE status = 'PENDING';
