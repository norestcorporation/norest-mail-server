-- 023_add_bounce_tracking.sql
-- Track bounce notification generation to ensure idempotency

ALTER TABLE message_delivery_status 
ADD COLUMN bounce_generated BOOLEAN DEFAULT false,
ADD COLUMN bounce_generated_at TIMESTAMPTZ,
ADD COLUMN bounce_email_id TEXT;

-- Index for efficient querying of deliveries that need bounces
CREATE INDEX idx_delivery_status_bounce_not_generated 
ON message_delivery_status(submission_id, recipient_email) 
WHERE bounce_generated = false AND status = 'failed' AND is_permanent = true;
