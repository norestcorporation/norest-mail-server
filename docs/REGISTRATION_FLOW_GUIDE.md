# Norest Mail Registration Flow Guide

## Overview

The Norest Mail registration system has been designed with automatic domain type detection and separated flows for platform-owned domains and custom domains. This guide explains the complete end-to-end flow for both types of users.

## Key Features

### Domain Type Detection

The registration API automatically detects the domain type from the user's email address:
- **Platform-owned domains**: Pre-verified domains managed by Norest (e.g., `norestmail.com`)
- **Custom domains**: User-owned domains requiring DNS verification (e.g., `example.com`)

### Automatic Flow Routing

- **Platform domains**: Auto-provisioning with immediate address reservation and Stalwart account creation
- **Custom domains**: Manual domain verification before address registration and Stalwart provisioning

### Separated Registration Stages

Each registration stage has distinct endpoints and responses with clear status communication.

## Platform Domain Flow

### Stage 1: Registration with Domain Detection

**Endpoint**: `POST /v1/auth/register`

**Request**:
```json
{
  "email": "user@norestmail.com",
  "password": "SecurePassword123!"
}
```

**Response**:
```json
{
  "id": "uuid",
  "email": "user@norestmail.com",
  "status": "pending",
  "access_token": "jwt_token",
  "refresh_token": "refresh_token",
  "registration_flow": {
    "id": "uuid",
    "email": "user@norestmail.com",
    "domain_type": "platform_owned",
    "status": "provisioning",
    "requires_action": null,
    "domain_id": "domain_uuid",
    "domain_name": "norestmail.com",
    "domain_verified": true,
    "address_id": "address_uuid",
    "mailbox_provisioned": true,
    "ready_for_mail": false
  }
}
```

**Key Points**:
- `domain_type` is `platform_owned`
- `domain_verified` is `true` (pre-verified)
- `requires_action` is `null` (no user action needed)
- Auto-provisioning is triggered automatically

### Stage 2: Check Provisioning Status

**Endpoint**: `GET /v1/mail/provisioning-status`

**Response**:
```json
{
  "mailbox_id": "mailbox_uuid",
  "address_id": "address_uuid",
  "status": "active",
  "stalwart_account_id": "stalwart_account_id",
  "ready_for_session": true
}
```

### Stage 3: Create Mail Session

**Endpoint**: `POST /v1/mail/session`

**Response**:
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_password",
  "account_id": "stalwart_account_id"
}
```

## Custom Domain Flow

### Stage 1: Registration with Domain Detection

**Endpoint**: `POST /v1/auth/register`

**Request**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response**:
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "status": "pending",
  "access_token": "jwt_token",
  "refresh_token": "refresh_token",
  "registration_flow": {
    "id": "uuid",
    "email": "user@example.com",
    "domain_type": "custom",
    "status": "pending",
    "requires_action": "add_domain",
    "domain_name": "example.com",
    "domain_verified": false,
    "mailbox_provisioned": false,
    "ready_for_mail": false
  }
}
```

**Key Points**:
- `domain_type` is `custom`
- `domain_verified` is `false` (verification required)
- `requires_action` is `add_domain` (user must add domain)

### Stage 2: Add Custom Domain

**Endpoint**: `POST /v1/domains`

**Request**:
```json
{
  "name": "example.com"
}
```

**Response**:
```json
{
  "id": "domain_uuid",
  "name": "example.com",
  "status": "pending",
  "verification_status": "pending",
  "ownership_type": "USER",
  "registration_enabled": false
}
```

### Stage 3: Domain Verification

**Endpoint**: `POST /v1/registration/domains/{domainID}/verify`

**Response**:
```json
{
  "domain_id": "domain_uuid",
  "domain_name": "example.com",
  "status": "verifying",
  "verification_token": "token_hash",
  "dns_record": {
    "type": "TXT",
    "name": "_norest-verification.example.com",
    "value": "norest-verification=token_hash"
  },
  "message": "Configure this TXT record in your DNS provider. The background worker will verify automatically."
}
```

**User Action**: Configure DNS TXT record

### Stage 4: Check Verification Status

**Endpoint**: `GET /v1/registration/domains/{domainID}/verify`

**Response**:
```json
{
  "domain_id": "domain_uuid",
  "domain_name": "example.com",
  "verification_status": "verifying",
  "registration_enabled": false,
  "dns_check": {
    "txt_record_verified": true,
    "mx_record_verified": true,
    "mx_records": ["mx1.example.com", "mx2.example.com"],
    "checked_at": "2024-01-01T00:00:00Z"
  },
  "next_action": "wait_for_activation"
}
```

**Background Process**: Worker verifies DNS and activates domain

### Stage 5: Domain Activation & Stalwart Provisioning

**Automatic Process**:
1. Background worker verifies TXT record
2. Worker checks MX records (informational)
3. Domain status changes to `verified` then `active`
4. `registration_enabled` set to `true`
5. DOMAIN_CREATE job created for Stalwart provisioning
6. Stalwart domain created asynchronously

**Check Status**: `GET /v1/domains/{domainID}`

**Response**:
```json
{
  "id": "domain_uuid",
  "name": "example.com",
  "status": "active",
  "verification_status": "verified",
  "registration_enabled": true,
  "stalwart_domain_id": "stalwart_domain_id"
}
```

### Stage 6: Address Registration

**Check Availability**: `GET /v1/domains/{domainID}/addresses/check/{localPart}`

**Reserve Address**: `POST /v1/domains/{domainID}/addresses/reserve`

**Request**:
```json
{
  "local_part": "user"
}
```

**Response**:
```json
{
  "id": "address_uuid",
  "local_part": "user",
  "domain_id": "domain_uuid",
  "status": "RESERVED",
  "reserved_until": "2024-01-01T02:00:00Z"
}
```

### Stage 7: Address Claim & Stalwart Account Creation

**Endpoint**: `POST /v1/addresses/{addressID}/claim`

**Response**:
```json
{
  "status": "claimed"
}
```

**Background Process**:
1. Address status changes to `CLAIMED`
2. Mailbox record created
3. ACCOUNT_CREATE job created
4. Stalwart account created asynchronously
5. Initial JMAP synchronization performed
6. Mailbox status changes to `active`

### Stage 8: Check Provisioning Status

**Endpoint**: `GET /v1/mail/provisioning-status`

**Response**:
```json
{
  "mailbox_id": "mailbox_uuid",
  "address_id": "address_uuid",
  "status": "active",
  "stalwart_account_id": "stalwart_account_id",
  "ready_for_session": true
}
```

### Stage 9: Create Mail Session

**Endpoint**: `POST /v1/mail/session`

**Response**:
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_password",
  "account_id": "stalwart_account_id"
}
```

## API Endpoints Summary

### Registration Flow Endpoints

- `POST /v1/auth/register` - User registration with automatic domain type detection
- `GET /v1/registration/status` - Get current registration status
- `POST /v1/registration/domains/{domainID}/verify` - Start domain verification
- `GET /v1/registration/domains/{domainID}/verify` - Check verification status with real-time DNS checks

### Domain Management Endpoints

- `GET /v1/domains/platform` - List platform domains
- `GET /v1/domains/check/{name}` - Check domain availability
- `POST /v1/domains` - Create custom domain
- `GET /v1/domains` - List user domains
- `GET /v1/domains/{id}` - Get domain details
- `DELETE /v1/domains/{id}` - Delete domain
- `POST /v1/domains/{id}/verification/start` - Start verification (legacy)
- `GET /v1/domains/{id}/verification` - Get verification instructions (legacy)

### Address Management Endpoints

- `GET /v1/domains/{domainID}/addresses/check/{localPart}` - Check address availability
- `POST /v1/domains/{domainID}/addresses/reserve` - Reserve address
- `GET /v1/domains/{domainID}/addresses` - List domain addresses
- `POST /v1/addresses/{addressID}/claim` - Claim address

### Mail Provisioning Endpoints

- `GET /v1/mail/provisioning-status` - Check provisioning status
- `GET /v1/mail/account` - Get mail account details
- `POST /v1/mail/session` - Create mail session

## Flow Comparison

### Platform Domain Flow
```
Register → Domain Detection → Platform Domain Found → Address Availability 
→ Address Reservation → Address Claim → Stalwart Account → Mailbox Sync 
→ Active → Mail Session
```

### Custom Domain Flow
```
Register → Domain Detection → Custom Domain Found → Add Domain 
→ DNS Verification → Domain Activation → Stalwart Domain → Address Reservation 
→ Address Claim → Stalwart Account → Mailbox Sync → Active → Mail Session
```

## Key Differences

| Aspect | Platform Domain | Custom Domain |
|--------|----------------|---------------|
| Domain Type | `platform_owned` | `custom` |
| Domain Verification | Pre-verified | Manual DNS verification required |
| Stalwart Provisioning | Automatic after registration | After domain verification |
| User Actions | None required | DNS configuration required |
| `requires_action` | `null` | `add_domain`, `verify_domain`, etc. |
| `domain_verified` | `true` | `false` initially |

## Acceptance Conditions

### Platform Domain Scenario
**Email**: `user@norestmail.com`

**Expected Response**:
```json
{
  "domain_type": "platform_owned",
  "domain_verified": true,
  "requires_action": null,
  "status": "provisioning"
}
```

**Expected Behavior**: System automatically proceeds toward Stalwart provisioning without user domain verification.

### Custom Domain Scenario
**Email**: `user@example.com`

**Expected Response**:
```json
{
  "domain_type": "custom",
  "domain_verified": false,
  "requires_action": "add_domain",
  "status": "pending"
}
```

**Expected Behavior**: Stalwart provisioning waits until domain verification and MX/DNS checks are satisfied.

## Testing with Postman Collections

Two updated Postman collections are provided:

1. **Norest-Mail-Platform-Domain.postman_collection.json**: Platform domain flow with auto-provisioning
2. **Norest-Mail-Custom-Domain.postman_collection.json**: Custom domain flow with manual verification

Both collections include:
- Health check endpoints
- Complete registration flows
- Automated test scripts for status validation
- Domain-specific response expectations

## Production Considerations

### Security

- Domain type detection prevents unauthorized access
- DNS verification prevents domain hijacking for custom domains
- Platform domains are pre-verified and controlled by Norest
- Secure password generation for Stalwart accounts

### Scalability

- Background workers handle provisioning asynchronously
- Platform domains have faster time-to-active
- Custom domains require user action but scale similarly
- Job queue system with retry logic

### Monitoring

- Separate metrics for platform vs custom domain flows
- Domain verification success/failure tracking
- Provisioning time comparison between flows
- User funnel analytics by domain type