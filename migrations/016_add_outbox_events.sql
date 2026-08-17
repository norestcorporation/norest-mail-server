-- 016_add_outbox_events.sql

CREATE TABLE IF NOT EXISTS mail_events_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mailbox_id UUID REFERENCES mailboxes(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL, -- e.g., 'message.created', 'message.updated'
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- 'pending', 'published', 'failed'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX idx_mail_events_outbox_user_id_status ON mail_events_outbox(user_id, status);
CREATE INDEX idx_mail_events_outbox_status_created_at ON mail_events_outbox(status, created_at);
