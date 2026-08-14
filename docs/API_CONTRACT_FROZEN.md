# Norest Mail API Contract - Frozen State

**Frozen Date:** 2026-08-14  
**Release:** Chapter 5  
**Status:** FROZEN - No new endpoints, no breaking changes to existing endpoints

## Overview

This document captures the frozen API contract state as of the Chapter 5 release. The API is now in a stable state suitable for frontend integration. No new endpoints will be added, and existing endpoints will maintain backward compatibility.

## Documentation Package Contents

The following documentation files constitute the complete API contract:

1. **API_REFERENCE.md** - Complete endpoint inventory with request/response formats
2. **AUTHENTICATION.md** - JWT-based authentication flow and token management
3. **STALWART_MANAGEMENT_API.md** - Server-side Stalwart management operations (development only)
4. **STARTUP.md** - Development environment setup and service startup
5. **QUICKSTART.md** - Quick start guide for immediate testing
6. **API_INDEX.md** - Documentation navigation guide
7. **JMAP_MAIL_API.md** - JMAP mail operations reference
8. **API_ERRORS.md** - Standard error response formats

## Postman Collection

The Postman collection (`postman/Norest-Mail.postman_collection.json`) provides:
- Complete request examples for all endpoints
- Automated test scripts for variable extraction
- Development environment configuration (`postman/Norest-Mail.postman_environment.json`)

## Current API State

### Health Endpoints
- GET /health
- GET /health/live
- GET /health/ready
- GET /health/db
- GET /health/stalwart
- GET /metrics

### Authentication Endpoints
- POST /v1/auth/register
- POST /v1/auth/login
- GET /v1/me

### Domain Management
- POST /v1/domains
- GET /v1/domains
- GET /v1/domains/{id}
- POST /v1/domains/{id}/verification/start
- GET /v1/domains/{id}/verification
- DELETE /v1/domains/{id}

### Address Management
- POST /v1/domains/{id}/addresses
- GET /v1/domains/{id}/addresses

### Mail Session
- POST /v1/mail/session
- GET /v1/mail/account

### Usage
- GET /v1/account/usage

### Admin Endpoints
- POST /v1/admin/accounts/{id}/suspend
- POST /v1/admin/accounts/{id}/reactivate

### Billing
- POST /v1/billing/webhook

## Authentication Flow

1. User registers via POST /v1/auth/register
2. User receives JWT access token and refresh token
3. All subsequent requests use Bearer token authentication
4. Mail operations require POST /v1/mail/session to obtain JMAP AppPassword
5. JMAP operations use HTTP Basic Auth with email:app_password

## Security Notes

- All development credentials are clearly marked as "development-only"
- Management APIs are documented as server-side only
- Production deployment requires strong secrets and HTTPS
- JMAP authentication uses standard HTTP Basic Auth
- JWT tokens have 15-minute lifetime for access tokens

## Testing Verified

The following flow has been tested from a clean Docker environment:
1. Fresh Docker environment startup via ./scripts/dev-up.sh
2. All 10 database migrations applied successfully
3. User registration successful
4. Domain creation successful
5. Address creation successful
6. Mail session creation successful
7. JMAP mailbox query successful

## Migration Status

The current migration set includes:
- 001_users.sql
- 002_domains.sql
- 003_addresses.sql
- 004_mailboxes.sql
- 005_provisioning_jobs.sql
- 006_chapter4_product_plane.sql
- 007_domain_verification.sql
- 008_admin_role.sql
- 009_update_verification_check.sql
- 010_chapter5b_job_lease.sql

## Changes from Previous Documentation

### Updated in Chapter 5 Freeze
- Corrected migration process to apply ALL migrations, not just 2 files
- Updated startup documentation to reference ./scripts/dev-up.sh
- Fixed JMAP authentication examples to use HTTP Basic Auth
- Added development-only warnings to management API documentation
- Fixed Postman collection to use proper Basic Auth for JMAP requests
- Added clear security warnings for development credentials

### No Backend Changes
- No API endpoint modifications
- No authentication flow changes
- No database schema changes
- No JMAP interface changes

## Freeze Policy

**Allowed:** 
- Bug fixes to existing endpoints
- Documentation clarifications
- Performance improvements

**Not Allowed:**
- New API endpoints
- Breaking changes to existing endpoints
- Authentication flow changes
- Database schema changes

## Next Steps

This frozen API contract is ready for:
1. Frontend integration (Chapter 6)
2. Production deployment planning
3. Client SDK development
4. Third-party integrations

Any API changes will require a new release cycle and contract unfreeze process.