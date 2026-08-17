-- 017_add_reconciliation.sql

CREATE TABLE IF NOT EXISTS mail_reconciliation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(255),
    mutation_type VARCHAR(64) NOT NULL, -- e.g. 'message.mark_read', 'message.send'
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING', -- PENDING, EXECUTING, SUCCESS, FAILED, UNKNOWN
    stalwart_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reconciliation_user_id ON mail_reconciliation_logs(user_id);
CREATE INDEX idx_reconciliation_status ON mail_reconciliation_logs(status) WHERE status IN ('PENDING', 'EXECUTING', 'UNKNOWN');
