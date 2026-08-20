-- 020_make_outbox_user_id_nullable.sql
-- Make user_id nullable in mail_events_outbox to support system-level events

ALTER TABLE mail_events_outbox ALTER COLUMN user_id DROP NOT NULL;
