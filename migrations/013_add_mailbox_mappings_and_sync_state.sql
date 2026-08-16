-- 013_add_mailbox_mappings_and_sync_state.sql
-- Add mailbox mappings and JMAP synchronization state tracking
-- This enables proper initial synchronization before account activation

-- Add mailbox mappings table to store role-to-ID mappings from Stalwart
CREATE TABLE IF NOT EXISTS mailbox_mappings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id          UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    role                TEXT NOT NULL, -- JMAP mailbox role (inbox, sent, drafts, trash, junk, etc.)
    stalwart_mailbox_id TEXT NOT NULL, -- Stalwart mailbox ID
    discovered_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(mailbox_id, role) -- One mapping per role per mailbox
);

-- Index for looking up mappings by mailbox
CREATE INDEX idx_mailbox_mappings_mailbox_id ON mailbox_mappings(mailbox_id);

-- Index for looking up mappings by stalwart mailbox ID
CREATE INDEX idx_mailbox_mappings_stalwart_id ON mailbox_mappings(stalwart_mailbox_id);

-- Add JMAP synchronization state to mailboxes table
ALTER TABLE mailboxes 
ADD COLUMN IF NOT EXISTS jmap_state TEXT,
ADD COLUMN IF NOT EXISTS initial_sync_completed_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS initial_sync_checkpoint TEXT;

-- Add status to support full provisioning lifecycle
-- Drop any existing status check constraint (it might have a different auto-generated name)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conrelid = 'mailboxes'::regclass 
        AND conname LIKE '%status%check%'
    ) THEN
        ALTER TABLE mailboxes DROP CONSTRAINT IF EXISTS mailboxes_status_check;
        ALTER TABLE mailboxes DROP CONSTRAINT IF EXISTS mailboxes_status_check1;
        ALTER TABLE mailboxes DROP CONSTRAINT IF EXISTS mailboxes_status_check2;
    END IF;
END $$;

ALTER TABLE mailboxes 
ADD CONSTRAINT mailboxes_status_check 
  CHECK (status IN ('pending', 'provisioning', 'syncing', 'active', 'inactive', 'disabled', 'failed'));

-- Update existing mailboxes to have proper status
UPDATE mailboxes SET status = 'active' WHERE status = 'inactive';

-- Add index for JMAP state lookups
CREATE INDEX idx_mailboxes_jmap_state ON mailboxes(jmap_state);

-- Add index for sync status tracking
CREATE INDEX idx_mailboxes_sync_status ON mailboxes(status, initial_sync_completed_at);
