-- 009_update_domain_verification_constraint.sql
-- Update domains verification_status check constraint to include 'verifying'

ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_verification_status_check;
ALTER TABLE domains ADD CONSTRAINT domains_verification_status_check 
    CHECK (verification_status IN ('pending', 'verifying', 'verified', 'failed'));
