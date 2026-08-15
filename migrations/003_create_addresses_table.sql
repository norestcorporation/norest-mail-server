-- 003_create_addresses_table.sql
-- Email addresses registered in the Norest control plane

CREATE TABLE addresses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id  UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    local_part TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'reserved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Composite uniqueness: domain_id + normalized local_part
CREATE UNIQUE INDEX idx_addresses_domain_local_unique ON addresses (domain_id, lower(local_part));

-- Index for looking up addresses by domain
CREATE INDEX idx_addresses_domain_id ON addresses (domain_id);
