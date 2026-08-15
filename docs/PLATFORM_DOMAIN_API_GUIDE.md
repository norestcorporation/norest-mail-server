# Platform Domain API Guide

This guide explains how to use the Norest Mail API to obtain and use an email address on a platform-owned domain (e.g., `ripun@norestmail.com`).

## Overview

Platform-owned domains are managed by Norest and do not require DNS verification. Users can simply choose an available address and register it.

## Complete API Flow

```
Register User
    ↓
List Platform Domains
    ↓
Check Address Availability
    ↓
Reserve Address
    ↓
Claim Address
    ↓
Check Provisioning Status
    ↓
Create Mail Session
    ↓
Use JMAP for Mail Operations
```

---

## API #1: Register User

### POST /v1/auth/register

**PURPOSE**: Create a new Norest user account. This is the first step for all users.

**WHEN TO CALL**: At the beginning of onboarding when the user doesn't have an account yet.

**REQUEST**:
```json
{
  "email": "platformuser@example.com",
  "password": "SecurePassword123!"
}
```

**RESPONSE** (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "platformuser@example.com",
  "status": "active",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**SAVE THESE VALUES**:
- `access_token` - Used for authentication in all subsequent requests
- `refresh_token` - Used to obtain new access tokens when expired
- `id` (user_id) - User identifier

**NEXT API**: List Platform Domains

**WHY**: You need to discover which platform domains are available for registration.

**DATABASE EFFECT**: Creates user record, product account, and subscription in the database.

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #2: List Platform Domains

### GET /v1/domains/platform

**PURPOSE**: Discover which platform-owned domains are available for address registration.

**WHEN TO CALL**: After authentication to see available domains.

**REQUEST**:
```http
GET /v1/domains/platform
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "norestmail.com",
    "status": "active",
    "verification_status": "verified",
    "ownership_type": "PLATFORM",
    "registration_enabled": true,
    "created_at": "2026-08-14T10:00:00Z",
    "updated_at": "2026-08-14T10:00:00Z"
  }
]
```

**SAVE THESE VALUES**:
- `id` (platform_domain_id) - Domain identifier for address operations
- `name` (platform_domain_name) - Domain name for email address construction

**NEXT API**: Check Address Availability

**WHY**: Before reserving an address, you need to verify it's available and not already taken.

**DATABASE EFFECT**: None (read-only)

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #3: Check Address Availability

### GET /v1/domains/{domainID}/addresses/check/{localPart}

**PURPOSE**: Verify if a specific address (local part) is available for reservation on a domain.

**WHEN TO CALL**: Before attempting to reserve an address to ensure it's available.

**REQUEST**:
```http
GET /v1/domains/{platform_domain_id}/addresses/check/testuser
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
{
  "available": true,
  "local_part": "testuser",
  "domain_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**SAVE THESE VALUES**: None (informational only)

**NEXT API**: Reserve Address

**WHY**: If the address is available, proceed to reserve it for your exclusive use.

**DATABASE EFFECT**: None (read-only)

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #4: Reserve Address

### POST /v1/domains/{domainID}/addresses/reserve

**PURPOSE**: Reserve an address for your exclusive use for 2 hours. This prevents others from taking it while you complete the registration process.

**WHEN TO CALL**: After confirming address availability and before claiming it.

**REQUEST**:
```json
POST /v1/domains/{platform_domain_id}/addresses/reserve
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "local_part": "testuser"
}
```

**RESPONSE** (201 Created):
```json
{
  "id": "abc12345-def67-8901-abcd-ef1234567890",
  "domain_id": "123e4567-e89b-12d3-a456-426614174000",
  "local_part": "testuser",
  "status": "RESERVED",
  "reserved_by": "550e8400-e29b-41d4-a716-446655440000",
  "reserved_at": "2026-08-14T10:00:00Z",
  "reserved_until": "2026-08-14T12:00:00Z",
  "created_at": "2026-08-14T10:00:00Z",
  "updated_at": "2026-08-14T10:00:00Z"
}
```

**SAVE THESE VALUES**:
- `id` (address_id) - Address identifier required for claiming

**NEXT API**: Claim Address

**WHY**: Complete the registration by claiming the reserved address, which triggers mailbox provisioning.

**DATABASE EFFECT**: Creates address record with RESERVED status in the database.

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #5: Claim Address

### POST /v1/addresses/{addressID}/claim

**PURPOSE**: Claim the reserved address as your own and trigger mailbox provisioning. This is the final step of address registration.

**WHEN TO CALL**: After reserving an address to complete the registration process.

**REQUEST**:
```http
POST /v1/addresses/{address_id}/claim
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
{
  "status": "claimed"
}
```

**SAVE THESE VALUES**: None (informational confirmation)

**NEXT API**: Check Provisioning Status

**WHY**: Mailbox provisioning happens asynchronously. You need to check when it's complete before using mail.

**DATABASE EFFECT**: 
- Updates address status to CLAIMED
- Creates mailbox record with provisioning status
- Creates ACCOUNT_CREATE provisioning job

**BACKGROUND EFFECT**: Triggers ACCOUNT_CREATE background job to provision mailbox in Stalwart

**STALWART EFFECT**: Indirect - background job will create Stalwart account

---

## API #6: Check Provisioning Status

### GET /v1/mail/provisioning-status

**PURPOSE**: Check if your mailbox has been fully provisioned and is ready for mail operations.

**WHEN TO CALL**: After claiming an address, poll this endpoint until `ready_for_session` is true.

**REQUEST**:
```http
GET /v1/mail/provisioning-status
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK - Not Ready):
```json
{
  "mailbox_id": "abc12345-def67-8901-abcd-ef1234567890",
  "address_id": "abc12345-def67-8901-abcd-ef1234567890",
  "status": "provisioning",
  "stalwart_account_id": null,
  "ready_for_session": false
}
```

**RESPONSE** (200 OK - Ready):
```json
{
  "mailbox_id": "abc12345-def67-8901-abcd-ef1234567890",
  "address_id": "abc12345-def67-8901-abcd-ef1234567890",
  "status": "active",
  "stalwart_account_id": "c",
  "ready_for_session": true
}
```

**SAVE THESE VALUES**:
- `mailbox_id` (mail_account_id) - Mailbox identifier
- `stalwart_account_id` - Stalwart account identifier (when ready)

**NEXT API**: Create Mail Session

**WHY**: Once provisioning is complete, you can create a mail session to access JMAP mail operations.

**DATABASE EFFECT**: None (read-only)

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #7: Create Mail Session

### POST /v1/mail/session

**PURPOSE**: Create a secure mail session that provides JMAP credentials for accessing your mailbox.

**WHEN TO CALL**: After provisioning is complete (ready_for_session = true).

**REQUEST**:
```http
POST /v1/mail/session
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "c"
}
```

**SAVE THESE VALUES**:
- `jmap_session_url` - JMAP endpoint URL
- `access_token` - JMAP AppPassword for authentication
- `account_id` - Stalwart account ID for JMAP operations

**NEXT API**: Use JMAP directly

**WHY**: With these credentials, you can now access JMAP mail operations directly from Stalwart.

**DATABASE EFFECT**: None

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: Creates an AppPassword in Stalwart for session authentication

---

## Mail Operations (JMAP)

After creating a mail session, all mail operations are performed directly via JMAP calls to Stalwart using the credentials provided.

### JMAP Endpoint
```
http://localhost:8081/.well-known/jmap
```

### Authentication
Use the `access_token` from the mail session response as basic auth username (no password).

### Common JMAP Operations

**Get JMAP Session**:
```http
GET http://localhost:8081/.well-known/jmap
Authorization: Basic {mail_session_access_token}
```

**List Mailboxes**:
```json
POST http://localhost:8081/jmap
Authorization: Basic {mail_session_access_token}
Content-Type: application/json

{
  "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
  "methodCalls": [
    ["Mailbox/get", {
      "accountId": "{mail_session_account_id}"
    }, "0"]
  ]
}
```

**Query Messages**:
```json
POST http://localhost:8081/jmap
Authorization: Basic {mail_session_access_token}
Content-Type: application/json

{
  "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
  "methodCalls": [
    ["Email/query", {
      "accountId": "{mail_session_account_id}",
      "filter": {
        "inMailbox": "INBOX"
      }
    }, "0"]
  ]
}
```

---

## Error Responses

### 401 Unauthorized
- Invalid or expired access token
- Solution: Re-authenticate with login endpoint

### 403 Forbidden
- Attempting to reserve on unverified custom domain
- Attempting to use another user's domain
- Account suspended or quota exceeded
- Solution: Fix the specific permission issue

### 409 Conflict
- Address already reserved or claimed
- Solution: Choose a different address

### 503 Service Unavailable
- Mailbox not ready for mail session
- Solution: Wait for provisioning to complete

---

## Key Concepts

### Platform Domain Characteristics
- `ownership_type = PLATFORM`
- No DNS verification required
- Any authenticated user can use
- Pre-configured by Norest

### Address States
- `AVAILABLE` - Free for reservation
- `RESERVED` - Reserved for 2 hours by a user
- `CLAIMED` - Permanently owned by a user
- `BLOCKED` - Cannot be used

### Provisioning Process
1. Claim address → Creates mailbox record
2. Background job → Creates Stalwart account
3. Poll provisioning status → Wait for completion
4. Create mail session → Get JMAP credentials

### Why Polling is Required
- Stalwart provisioning is asynchronous
- Background worker handles the heavy lifting
- Client must check status periodically
- Prevents blocking the API during long operations

---

## Testing the Flow

1. Execute all requests in sequence
2. Verify each response has expected status codes
3. Check that variables are automatically saved
4. Monitor console logs for step-by-step progress
5. Test negative scenarios to ensure proper error handling

The Norest Mail Platform Domain collection is designed to be self-contained and executable from a fresh environment without manual ID copying.