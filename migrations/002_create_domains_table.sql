-- 002_domains.sql
-- Mail domains registered in the Norest control plane

CREATE TABLE domains (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    stalwart_domain_id  TEXT,
    status              TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'verifying', 'active', 'suspended', 'disabled')),
    verification_status TEXT NOT NULL DEFAULT 'pending' CHECK (verification_status IN ('pending', 'verified', 'failed')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Domain names must be unique and are stored normalized to lowercase
CREATE UNIQUE INDEX idx_domains_name_unique ON domains (lower(name));

-- Index for looking up domains by user
CREATE INDEX idx_domains_user_id ON domains (user_id);
