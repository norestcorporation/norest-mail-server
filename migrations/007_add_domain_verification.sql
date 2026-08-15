-- 007_add_domain_verification.sql
-- Add verification token hash for DNS domain ownership proof
ALTER TABLE domains ADD COLUMN verification_token_hash TEXT;
