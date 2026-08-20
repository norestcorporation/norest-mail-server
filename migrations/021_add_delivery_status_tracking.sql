-- 021_add_delivery_status_tracking.sql
-- Track delivery status for sent messages

CREATE TABLE IF NOT EXISTS message_delivery_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id TEXT NOT NULL, -- Stalwart email ID
    submission_id TEXT NOT NULL, -- Stalwart EmailSubmission ID
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mailbox_id UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    
    -- Status lifecycle: submitted -> sending/queued -> delivered/failed/bounced
    status TEXT NOT NULL DEFAULT 'submitted' CHECK (status IN (
        'submitted',      -- Successfully submitted to Stalwart
        'queued',         -- Queued for delivery
        'sending',        -- Currently being sent
        'delivered',      -- Successfully delivered
        'failed',         -- Delivery failed (permanent)
        'bounced',        -- Bounced by recipient server
        'temporary_failure', -- Temporary failure, will retry
        'unknown'         -- Status could not be determined
    )),
    
    -- Recipient-specific delivery info
    recipient_email TEXT NOT NULL,
    smtp_reply TEXT, -- Raw SMTP reply from recipient server
    delivered TEXT, -- Stalwart delivery status: "queued", "yes", "no", "unknown"
    
    -- Error information
    error_message TEXT,
    error_type TEXT, -- e.g., "mailbox_not_found", "domain_not_found", "quota_exceeded"
    is_permanent BOOLEAN DEFAULT false,
    
    -- Timestamps
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    
    -- Metadata
    subject TEXT,
    retry_count INTEGER DEFAULT 0,
    next_retry_at TIMESTAMPTZ
);

-- Indexes for efficient querying
CREATE INDEX idx_delivery_status_message_id ON message_delivery_status(message_id);
CREATE INDEX idx_delivery_status_submission_id ON message_delivery_status(submission_id);
CREATE INDEX idx_delivery_status_user_id ON message_delivery_status(user_id);
CREATE INDEX idx_delivery_status_status ON message_delivery_status(status);
CREATE INDEX idx_delivery_status_recipient ON message_delivery_status(recipient_email);
CREATE INDEX idx_delivery_status_next_retry ON message_delivery_status(next_retry_at) WHERE next_retry_at IS NOT NULL;

-- Unique constraint to prevent duplicate tracking for same submission
CREATE UNIQUE INDEX idx_delivery_status_unique ON message_delivery_status(submission_id, recipient_email);
