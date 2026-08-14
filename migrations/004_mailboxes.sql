-- 004_mailboxes.sql
-- Norest mailbox records linking product addresses to Stalwart accounts.
-- This does NOT store messages. It only records:
-- "This product address is associated with a Stalwart account."

CREATE TABLE mailboxes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    address_id          UUID NOT NULL REFERENCES addresses(id) ON DELETE CASCADE,
    stalwart_account_id TEXT,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'provisioning')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for looking up mailboxes by address
CREATE INDEX idx_mailboxes_address_id ON mailboxes (address_id);

-- Index for looking up mailboxes by Stalwart account
CREATE INDEX idx_mailboxes_stalwart_account ON mailboxes (stalwart_account_id);
