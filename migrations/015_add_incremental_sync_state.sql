-- 015_add_incremental_sync_state.sql

CREATE TABLE IF NOT EXISTS mail_sync_state (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id      UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    state           TEXT, -- The current JMAP state/checkpoint
    last_synced_at  TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'idle', -- 'idle', 'syncing', 'error'
    error_message   TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(mailbox_id)
);

CREATE INDEX idx_mail_sync_state_mailbox_id ON mail_sync_state(mailbox_id);
