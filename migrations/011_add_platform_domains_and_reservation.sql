-- 011_add_platform_domains_and_reservation.sql
-- Platform-owned domains and proper address reservation system

-- 1. Add platform domain support to domains table
ALTER TABLE domains ADD COLUMN IF NOT EXISTS ownership_type TEXT NOT NULL DEFAULT 'USER' CHECK (ownership_type IN ('PLATFORM', 'USER'));
ALTER TABLE domains ADD COLUMN IF NOT EXISTS registration_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE domains ALTER COLUMN user_id DROP NOT NULL; -- Allow NULL for platform domains

-- 2. Update domains status check to include PLATFORM status if needed
-- Drop any existing status check constraint (it might have a different auto-generated name)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conrelid = 'domains'::regclass 
        AND conname LIKE '%status%check%'
    ) THEN
        ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_status_check;
        ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_status_check1;
        ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_status_check2;
    END IF;
END $$;

ALTER TABLE domains ADD CONSTRAINT domains_status_check CHECK (status IN ('pending', 'verifying', 'active', 'suspended', 'disabled'));

-- 3. Add proper address reservation system
ALTER TABLE addresses ADD COLUMN IF NOT EXISTS reserved_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE addresses ADD COLUMN IF NOT EXISTS reserved_at TIMESTAMPTZ;
ALTER TABLE addresses ADD COLUMN IF NOT EXISTS reserved_until TIMESTAMPTZ;
ALTER TABLE addresses ADD COLUMN IF NOT EXISTS claimed_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE addresses ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;

-- Update address status check to support proper reservation flow
-- Drop any existing status check constraint (it might have a different auto-generated name)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conrelid = 'addresses'::regclass 
        AND conname LIKE '%status%check%'
    ) THEN
        ALTER TABLE addresses DROP CONSTRAINT IF EXISTS addresses_status_check;
        ALTER TABLE addresses DROP CONSTRAINT IF EXISTS addresses_status_check1;
        ALTER TABLE addresses DROP CONSTRAINT IF EXISTS addresses_status_check2;
    END IF;
END $$;

ALTER TABLE addresses ADD CONSTRAINT addresses_status_check CHECK (status IN ('AVAILABLE', 'RESERVED', 'CLAIMED', 'BLOCKED'));
ALTER TABLE addresses ALTER COLUMN status SET DEFAULT 'AVAILABLE';

-- 4. Create blocked addresses table
CREATE TABLE IF NOT EXISTS blocked_addresses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id  UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    local_part TEXT NOT NULL,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add unique constraint for blocked addresses
-- Note: PostgreSQL doesn't support UNIQUE with function calls inside DO blocks
-- Using a simpler approach with standard UNIQUE constraint
ALTER TABLE blocked_addresses ADD CONSTRAINT blocked_addresses_unique UNIQUE (domain_id, local_part);

-- 5. Create indexes for reservation queries
CREATE INDEX IF NOT EXISTS idx_addresses_reservation ON addresses (domain_id, local_part, status) WHERE status = 'RESERVED';
CREATE INDEX IF NOT EXISTS idx_addresses_reservation_expiry ON addresses (reserved_until) WHERE status = 'RESERVED';
CREATE INDEX IF NOT EXISTS idx_addresses_claimed_by ON addresses (claimed_by);
CREATE INDEX IF NOT EXISTS idx_blocked_addresses_domain ON blocked_addresses (domain_id);

-- 6. Add function to check address availability
CREATE OR REPLACE FUNCTION check_address_available(p_domain_id UUID, p_local_part TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- Check if address is blocked
    IF EXISTS (SELECT 1 FROM blocked_addresses WHERE domain_id = p_domain_id AND local_part = p_local_part) THEN
        RETURN FALSE;
    END IF;
    
    -- Check if address is already claimed
    IF EXISTS (SELECT 1 FROM addresses WHERE domain_id = p_domain_id AND local_part = p_local_part AND status = 'CLAIMED') THEN
        RETURN FALSE;
    END IF;
    
    -- Check if address is reserved and not expired
    IF EXISTS (
        SELECT 1 FROM addresses 
        WHERE domain_id = p_domain_id 
        AND local_part = p_local_part 
        AND status = 'RESERVED' 
        AND reserved_until > NOW()
    ) THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- 7. Add function to reserve address with race safety
CREATE OR REPLACE FUNCTION reserve_address(
    p_domain_id UUID, 
    p_local_part TEXT, 
    p_user_id UUID, 
    p_duration_hours INTEGER DEFAULT 2
) RETURNS UUID AS $$
DECLARE
    v_address_id UUID;
    v_expires_at TIMESTAMPTZ;
BEGIN
    v_expires_at := NOW() + (p_duration_hours || ' hours')::INTERVAL;
    
    -- Try to insert new reservation
    INSERT INTO addresses (domain_id, local_part, status, reserved_by, reserved_at, reserved_until)
    VALUES (p_domain_id, p_local_part, 'RESERVED', p_user_id, NOW(), v_expires_at)
    ON CONFLICT (domain_id, local_part) 
    DO UPDATE SET
        status = 'RESERVED',
        reserved_by = p_user_id,
        reserved_at = NOW(),
        reserved_until = v_expires_at
    WHERE 
        addresses.status = 'AVAILABLE' 
        OR (addresses.status = 'RESERVED' AND addresses.reserved_until <= NOW())
    RETURNING id INTO v_address_id;
    
    IF v_address_id IS NULL THEN
        -- Address is not available (claimed or valid reservation exists)
        RAISE EXCEPTION 'Address not available';
    END IF;
    
    RETURN v_address_id;
END;
$$ LANGUAGE plpgsql;

-- 8. Add function to claim reserved address
CREATE OR REPLACE FUNCTION claim_address(p_address_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_status TEXT;
    v_reserved_by UUID;
BEGIN
    -- Lock the row for update
    SELECT status, reserved_by INTO v_status, v_reserved_by
    FROM addresses 
    WHERE id = p_address_id
    FOR UPDATE;
    
    IF v_status IS NULL THEN
        RAISE EXCEPTION 'Address not found';
    END IF;
    
    IF v_status = 'CLAIMED' THEN
        RAISE EXCEPTION 'Address already claimed';
    END IF;
    
    IF v_status = 'BLOCKED' THEN
        RAISE EXCEPTION 'Address is blocked';
    END IF;
    
    IF v_status = 'RESERVED' THEN
        -- Verify the user claiming is the one who reserved (or admin)
        IF v_reserved_by IS NOT NULL AND v_reserved_by != p_user_id THEN
            -- Check if user is admin (simplified - in real system, check admin role)
            -- For now, allow any user to claim if reservation expired
            IF NOT EXISTS (
                SELECT 1 FROM addresses 
                WHERE id = p_address_id AND reserved_until <= NOW()
            ) THEN
                RAISE EXCEPTION 'Address reserved by another user';
            END IF;
        END IF;
    END IF;
    
    -- Claim the address
    UPDATE addresses
    SET 
        status = 'CLAIMED',
        claimed_by = p_user_id,
        claimed_at = NOW(),
        reserved_by = NULL,
        reserved_at = NULL,
        reserved_until = NULL
    WHERE id = p_address_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- 9. Add function to clean expired reservations
CREATE OR REPLACE FUNCTION clean_expired_reservations() RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    WITH expired_reservations AS (
        SELECT id FROM addresses 
        WHERE status = 'RESERVED' AND reserved_until <= NOW()
        FOR UPDATE SKIP LOCKED
    )
    UPDATE addresses
    SET 
        status = 'AVAILABLE',
        reserved_by = NULL,
        reserved_at = NULL,
        reserved_until = NULL
    WHERE id IN (SELECT id FROM expired_reservations);
    
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- 10. Insert default platform domain (with correct status)
-- Temporarily disable constraint to insert, then re-enable
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conrelid = 'domains'::regclass 
        AND conname LIKE '%status%check%'
    ) THEN
        ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_status_check;
        ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_status_check1;
        ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_status_check2;
    END IF;
END $$;

INSERT INTO domains (name, ownership_type, registration_enabled, status, verification_status, user_id, product_account_id)
VALUES ('norestmail.com', 'PLATFORM', true, 'active', 'verified', NULL, NULL)
ON CONFLICT (name) DO UPDATE SET
    ownership_type = EXCLUDED.ownership_type,
    registration_enabled = EXCLUDED.registration_enabled,
    status = EXCLUDED.status,
    verification_status = EXCLUDED.verification_status;

-- Re-enable the constraint
ALTER TABLE domains ADD CONSTRAINT domains_status_check CHECK (status IN ('pending', 'verifying', 'active', 'suspended', 'disabled'));