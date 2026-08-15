# Custom Domain API Guide

This guide explains how to use the Norest Mail API to bring your own domain and obtain an email address (e.g., `ripun@intellaris.in`).

## Overview

Custom/user-owned domains require DNS verification to prove ownership before addresses can be registered. This ensures only the legitimate domain owner can use it for email.

## Complete API Flow

```
Register User
    ↓
Create Custom Domain
    ↓
Start DNS Verification
    ↓
Get DNS Instructions
    ↓
USER CONFIGURES DNS TXT RECORD
    ↓
Check Verification Status (poll)
    ↓
Domain Becomes Verified
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
  "email": "customuser@example.com",
  "password": "SecurePassword123!"
}
```

**RESPONSE** (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "customuser@example.com",
  "status": "active",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**SAVE THESE VALUES**:
- `access_token` - Used for authentication in all subsequent requests
- `refresh_token` - Used to obtain new access tokens when expired
- `id` (user_id) - User identifier

**NEXT API**: Create Custom Domain

**WHY**: You need to register your custom domain with Norest before you can use it for email.

**DATABASE EFFECT**: Creates user record, product account, and subscription in the database.

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #2: Create Custom Domain

### POST /v1/domains

**PURPOSE**: Register your custom domain with Norest. This does NOT mean the domain is verified yet.

**WHEN TO CALL**: After authentication to register your custom domain.

**REQUEST**:
```json
POST /v1/domains
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "intellaris.in"
}
```

**RESPONSE** (201 Created):
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "intellaris.in",
  "status": "pending",
  "verification_status": "pending",
  "ownership_type": "USER",
  "registration_enabled": false,
  "created_at": "2026-08-14T10:00:00Z",
  "updated_at": "2026-08-14T10:00:00Z"
}
```

**SAVE THESE VALUES**:
- `id` (custom_domain_id) - Domain identifier for verification and address operations
- `name` (custom_domain_name) - Domain name for email address construction
- `status` (custom_domain_status) - Track domain status changes
- `verification_status` (custom_domain_verification_status) - Track verification progress

**NEXT API**: Start DNS Verification

**WHY**: The domain is not verified yet. You must prove ownership via DNS verification before you can register addresses.

**DATABASE EFFECT**: Creates domain record with USER ownership_type and PENDING status.

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #3: Start DNS Verification

### POST /v1/domains/{id}/verification/start

**PURPOSE**: Start the DNS verification process to prove domain ownership.

**WHEN TO CALL**: After creating a custom domain to begin ownership verification.

**REQUEST**:
```http
POST /v1/domains/{custom_domain_id}/verification/start
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "intellaris.in",
  "status": "pending",
  "verification_status": "verifying",
  "verification_token": "abc123xyz",
  "ownership_type": "USER",
  "registration_enabled": false,
  "created_at": "2026-08-14T10:00:00Z",
  "updated_at": "2026-08-14T10:01:00Z"
}
```

**SAVE THESE VALUES**:
- `verification_token` - Security token for DNS verification (save this value!)

**NEXT API**: Get DNS Instructions

**WHY**: You need the specific DNS record details to configure in your DNS provider.

**DATABASE EFFECT**: 
- Updates verification_status to "verifying"
- Creates DOMAIN_VERIFY background job

**BACKGROUND EFFECT**: Triggers DOMAIN_VERIFY background job to check DNS

**STALWART EFFECT**: None

---

## API #4: Get DNS Instructions

### GET /v1/domains/{id}/verification

**PURPOSE**: Get the specific DNS TXT record that must be configured to prove domain ownership.

**WHEN TO CALL**: After starting verification to get the DNS configuration instructions.

**REQUEST**:
```http
GET /v1/domains/{custom_domain_id}/verification
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
{
  "type": "TXT",
  "name": "_norest-verification.intellaris.in",
  "value": "norest-verification=abc123xyz",
  "status": "verifying",
  "message": "Configure this TXT record in your DNS"
}
```

**SAVE THESE VALUES**: None (informational only)

**NEXT API**: USER ACTION (Configure DNS)

**WHY**: The user must manually configure this DNS record in their DNS provider before verification can succeed.

**DATABASE EFFECT**: None (read-only)

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## USER ACTION: Configure DNS TXT Record

**WHAT THE USER MUST DO**:
1. Log into their DNS provider (e.g., GoDaddy, Cloudflare, Namecheap)
2. Navigate to DNS management for their domain
3. Add a new TXT record with the following values:
   - **Type**: TXT
   - **Name/Host**: `_norest-verification.intellaris.in`
   - **Value**: `norest-verification=abc123xyz`
4. Save the DNS record

**WHY THIS IS REQUIRED**:
- Proves you own the domain
- Prevents unauthorized use of domains
- Ensures email deliverability
- Required by Norest before address registration

**HOW IT WORKS**:
- Background worker periodically checks DNS
- Verifies the TXT record exists and matches
- Once verified, domain becomes active
- No address registration until verification succeeds

**WHAT HAPPENS NEXT**:
- Background worker checks DNS (every few minutes)
- If DNS record is correct, domain becomes verified
- If DNS record is missing/incorrect, verification fails
- Client must poll verification status to check progress

---

## API #5: Check Verification Status

### GET /v1/domains/{id}

**PURPOSE**: Check if the domain has been verified and is ready for address registration.

**WHEN TO CALL**: After configuring DNS, poll this endpoint until verification_status is "verified".

**REQUEST**:
```http
GET /v1/domains/{custom_domain_id}
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK - Still Verifying):
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "intellaris.in",
  "status": "pending",
  "verification_status": "verifying",
  "ownership_type": "USER",
  "registration_enabled": false,
  "created_at": "2026-08-14T10:00:00Z",
  "updated_at": "2026-08-14T10:01:00Z"
}
```

**RESPONSE** (200 OK - Verified):
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "intellaris.in",
  "status": "active",
  "verification_status": "verified",
  "ownership_type": "USER",
  "registration_enabled": true,
  "created_at": "2026-08-14T10:00:00Z",
  "updated_at": "2026-08-14T10:05:00Z"
}
```

**SAVE THESE VALUES**:
- `status` (custom_domain_status) - Update when domain becomes active
- `verification_status` (custom_domain_verification_status) - Update when verified

**NEXT API**: Check Address Availability

**WHY**: Once verified, you can check if your desired address is available for registration.

**DATABASE EFFECT**: None (read-only)

**BACKGROUND EFFECT**: None

**STALWART EFFECT**: None

---

## API #6: Check Address Availability

### GET /v1/domains/{domainID}/addresses/check/{localPart}

**PURPOSE**: Verify if a specific address (local part) is available for reservation on your verified domain.

**WHEN TO CALL**: After domain verification and before attempting to reserve an address.

**REQUEST**:
```http
GET /v1/domains/{custom_domain_id}/addresses/check/ripun
Authorization: Bearer {access_token}
```

**RESPONSE** (200 OK):
```json
{
  "available": true,
  "local_part": "ripun",
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

## API #7: Reserve Address

### POST /v1/domains/{domainID}/addresses/reserve

**PURPOSE**: Reserve an address for your exclusive use for 2 hours. This prevents others from taking it while you complete the registration process.

**WHEN TO CALL**: After confirming address availability and before claiming it.

**REQUEST**:
```json
POST /v1/domains/{custom_domain_id}/addresses/reserve
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "local_part": "ripun"
}
```

**RESPONSE** (201 Created):
```json
{
  "id": "abc12345-def67-8901-abcd-ef1234567890",
  "domain_id": "123e4567-e89b-12d3-a456-426614174000",
  "local_part": "ripun",
  "status": "RESERVED",
  "reserved_by": "550e8400-e29b-41d4-a716-446655440000",
  "reserved_at": "2026-08-14T10:06:00Z",
  "reserved_until": "2026-08-14T12:06:00Z",
  "created_at": "2026-08-14T10:06:00Z",
  "updated_at": "2026-08-14T10:06:00Z"
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

## API #8: Claim Address

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

## API #9: Check Provisioning Status

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
  "stalwart_account_id": "ripun",
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

## API #10: Create Mail Session

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
  "account_id": "ripun"
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

### 403 Forbidden - Unverified Domain
- Attempting to reserve address on unverified custom domain
- Response: "domain must be verified before registering addresses"
- Solution: Complete DNS verification first

### 403 Forbidden - Not Your Domain
- Attempting to use another user's custom domain
- Response: "domain does not belong to the user"
- Solution: Use your own verified domain

### 409 Conflict
- Address already reserved or claimed
- Solution: Choose a different address

### 503 Service Unavailable
- Mailbox not ready for mail session
- Solution: Wait for provisioning to complete

---

## Key Concepts

### Custom Domain Characteristics
- `ownership_type = USER`
- Requires DNS verification
- Only the domain owner can use it
- Domain must be verified before address registration

### DNS Verification Process
1. Create domain (status: pending)
2. Start verification (status: verifying)
3. Configure DNS TXT record
4. Background worker checks DNS
5. Domain becomes verified (status: active)
6. Address registration enabled

### Why DNS Verification is Required
- Proves domain ownership
- Prevents unauthorized domain use
- Ensures email deliverability
- Required for spam prevention
- Industry standard practice

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
- DNS verification is asynchronous
- Stalwart provisioning is asynchronous
- Background workers handle heavy lifting
- Client must check status periodically
- Prevents blocking the API during long operations

---

## Testing the Flow

1. Execute all requests in sequence
2. Configure DNS TXT record when instructed
3. Poll verification status until domain is verified
4. Verify each response has expected status codes
5. Check that variables are automatically saved
6. Monitor console logs for step-by-step progress
7. Test negative scenarios to ensure proper error handling

The Norest Mail Custom Domain collection is designed to be self-contained and executable from a fresh environment without manual ID copying.