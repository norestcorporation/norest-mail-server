-- 019_add_sync_checkpoints.sql
-- Checkpoint table for durable historical backfill and synchronization.

CREATE TABLE sync_checkpoints (
    account_id UUID PRIMARY KEY REFERENCES mailboxes(id) ON DELETE CASCADE,
    last_jmap_state TEXT,
    backfill_position TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'backfilling', 'active', 'failed')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
