-- 006_chapter4_product_plane.sql
-- Chapter 4: Product Control Plane, Billing, and Quotas

-- 1. Product Accounts
CREATE TABLE product_accounts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status     TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DISABLED', 'PENDING')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Link users to product accounts (for now, 1:1 mapped via a join table or just user_id on product_accounts)
-- Since a user can have multiple accounts later, let's create a join table
CREATE TABLE user_product_accounts (
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_account_id UUID NOT NULL REFERENCES product_accounts(id) ON DELETE CASCADE,
    role               TEXT NOT NULL DEFAULT 'owner',
    PRIMARY KEY (user_id, product_account_id)
);

-- Alter domains to belong to product_accounts instead of users
-- First add the column
ALTER TABLE domains ADD COLUMN product_account_id UUID REFERENCES product_accounts(id) ON DELETE CASCADE;
-- We'll backfill it if there are any existing domains, but let's assume we can just drop existing data for tests if needed, or we write a simple backfill.
-- For simplicity, since it's a test db, we can just TRUNCATE users CASCADE (which truncates domains) before applying, OR allow NULL temporarily.
-- Actually, let's just make it NULLable for a moment, we will handle it in Go if needed, but it should be NOT NULL eventually.

-- 2. Plans
CREATE TABLE plans (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code              TEXT NOT NULL UNIQUE,
    name              TEXT NOT NULL,
    description       TEXT,
    status            TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    max_domains       INTEGER NOT NULL,
    max_mailboxes     INTEGER NOT NULL,
    max_addresses     INTEGER NOT NULL,
    max_storage_bytes BIGINT NOT NULL,
    features          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert some default plans
INSERT INTO plans (code, name, max_domains, max_mailboxes, max_addresses, max_storage_bytes) VALUES
('FREE', 'Free Plan', 1, 5, 10, 1073741824),          -- 1GB
('STARTER', 'Starter Plan', 10, 25, 50, 5368709120),   -- 5GB
('PRO', 'Pro Plan', 50, 250, 500, 53687091200);       -- 50GB

-- 3. Subscriptions
CREATE TABLE subscriptions (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_account_id       UUID NOT NULL REFERENCES product_accounts(id) ON DELETE CASCADE,
    plan_id                  UUID NOT NULL REFERENCES plans(id),
    status                   TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('TRIALING', 'ACTIVE', 'PAST_DUE', 'CANCELED', 'EXPIRED', 'PAUSED')),
    provider                 TEXT NOT NULL DEFAULT 'internal',
    provider_customer_id     TEXT,
    provider_subscription_id TEXT,
    current_period_start     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end       TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    cancel_at_period_end     BOOLEAN NOT NULL DEFAULT false,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Billing Events
CREATE TABLE billing_events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider          TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_type        TEXT NOT NULL,
    payload_hash      TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED')),
    processed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);

-- 5. Extend Provisioning Jobs
-- We need to update the CHECK constraint for provisioning_jobs.type
ALTER TABLE provisioning_jobs DROP CONSTRAINT IF EXISTS provisioning_jobs_type_check;
ALTER TABLE provisioning_jobs ADD CONSTRAINT provisioning_jobs_type_check 
CHECK (type IN (
    'DOMAIN_CREATE', 'DOMAIN_DELETE', 'ACCOUNT_CREATE', 'ACCOUNT_DISABLE', 'ACCOUNT_ENABLE', 'ACCOUNT_DELETE',
    'DOMAIN_VERIFY', 'DOMAIN_ACTIVATE', 'DOMAIN_SUSPEND', 'DOMAIN_DISABLE',
    'ACCOUNT_QUOTA_SYNC', 'ACCOUNT_SUSPEND', 'ACCOUNT_REACTIVATE', 'ACCOUNT_PLAN_SYNC'
));
