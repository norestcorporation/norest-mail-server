# Norest Mail API Reference

**Frozen API Contract - Chapter 5 Release**

Complete inventory of all public Norest REST API endpoints. This documentation represents the current frozen API state as of Chapter 5 release.

## Health Endpoints

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| GET | /health | None | Basic health check | None | `{"status": "ok"}` |
| GET | /health/live | None | Liveness probe | None | `{"status": "alive"}` |
| GET | /health/ready | None | Readiness probe (DB + Stalwart) | None | `{"status": "ready"}` or error |
| GET | /health/db | None | Database health check | None | `{"status": "ok", "service": "database"}` or error |
| GET | /health/stalwart | None | Stalwart health check | None | `{"status": "ok", "service": "stalwart"}` or error |
| GET | /metrics | None | Application metrics | None | Metrics snapshot |

## Authentication Endpoints

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| POST | /v1/auth/register | None | Register new user | `{"email": string, "password": string}` | User object with tokens |
| POST | /v1/auth/login | None | Login user | `{"email": string, "password": string}` | User object with tokens |
| GET | /v1/me | Bearer token | Get current user info | None | User object |

### POST /v1/auth/register

**Rate Limit:** 5 requests per hour per IP

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Validation:**
- Email must be valid format
- Password must be at least 8 characters

**Success Response (201):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "status": "pending",
  "access_token": "jwt_token",
  "refresh_token": "jwt_token"
}
```

**Error Responses:**
- 400: Invalid email or password too weak
- 409: User already exists

### POST /v1/auth/login

**Rate Limit:** 10 requests per minute per IP

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Success Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "status": "active",
  "access_token": "jwt_token",
  "refresh_token": "jwt_token"
}
```

**Error Responses:**
- 401: Invalid credentials

### GET /v1/me

**Authorization:** Bearer {{norest_access_token}}

**Success Response (200):**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "status": "active"
}
```

**Error Responses:**
- 401: Unauthorized
- 404: User not found

## Domain Endpoints

All domain endpoints require authentication: `Authorization: Bearer {{norest_access_token}}`

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| POST | /v1/domains | Bearer token | Create domain | `{"name": string}` | Domain object |
| GET | /v1/domains | Bearer token | List user domains | None | Array of domains |
| GET | /v1/domains/{id} | Bearer token | Get domain details | None | Domain object |
| DELETE | /v1/domains/{id} | Bearer token | Delete domain | None | 204 No Content |
| POST | /v1/domains/{id}/verification/start | Bearer token | Start DNS verification | None | Domain object |
| GET | /v1/domains/{id}/verification | Bearer token | Get verification status | None | Verification instructions |

### POST /v1/domains

**Request:**
```json
{
  "name": "example.com"
}
```

**Success Response (201):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "product_account_id": "uuid",
  "name": "example.com",
  "status": "pending",
  "verification_status": "pending",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Error Responses:**
- 400: Invalid domain format
- 409: Domain already exists
- 403: Quota exceeded or account suspended

### GET /v1/domains

**Success Response (200):**
```json
[
  {
    "id": "uuid",
    "name": "example.com",
    "status": "active",
    "verification_status": "verified",
    "stalwart_domain_id": "stalwart_id"
  }
]
```

### GET /v1/domains/{id}

**Path Parameters:**
- `id`: Domain UUID

**Success Response (200):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "product_account_id": "uuid",
  "name": "example.com",
  "status": "active",
  "verification_status": "verified",
  "stalwart_domain_id": "stalwart_id",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Error Responses:**
- 400: Invalid domain ID
- 404: Domain not found

### DELETE /v1/domains/{id}

**Path Parameters:**
- `id`: Domain UUID

**Success Response:** 204 No Content

**Error Responses:**
- 400: Invalid domain ID
- 404: Domain not found

### POST /v1/domains/{id}/verification/start

**Path Parameters:**
- `id`: Domain UUID

**Success Response (200):**
```json
{
  "id": "uuid",
  "name": "example.com",
  "status": "verifying",
  "verification_status": "verifying"
}
```

### GET /v1/domains/{id}/verification

**Path Parameters:**
- `id`: Domain UUID

**Success Response (200):**
```json
{
  "type": "TXT",
  "name": "_norest-verification.example.com",
  "value": "norest-verification=..."
}
```

## Address Endpoints

All address endpoints require authentication: `Authorization: Bearer {{norest_access_token}}`

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| POST | /v1/domains/{domainID}/addresses | Bearer token | Create address | `{"local_part": string}` | Address object |
| GET | /v1/domains/{domainID}/addresses | Bearer token | List domain addresses | None | Array of addresses |

### POST /v1/domains/{domainID}/addresses

**Path Parameters:**
- `domainID`: Domain UUID

**Request:**
```json
{
  "local_part": "alice"
}
```

**Success Response (201):**
```json
{
  "id": "uuid",
  "domain_id": "uuid",
  "local_part": "alice",
  "status": "reserved",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Error Responses:**
- 400: Invalid local part or domain ID
- 404: Domain not found
- 409: Address already exists
- 403: Quota exceeded or account suspended

### GET /v1/domains/{domainID}/addresses

**Path Parameters:**
- `domainID`: Domain UUID

**Success Response (200):**
```json
[
  {
    "id": "uuid",
    "domain_id": "uuid",
    "local_part": "alice",
    "status": "active",
    "created_at": "timestamp"
  }
]
```

## Mail Session Endpoints

Mail endpoints require authentication: `Authorization: Bearer {{norest_access_token}}`

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| POST | /v1/mail/session | Bearer token | Create mail session | None | Session object |
| GET | /v1/mail/account | Bearer token | Get mailbox info | None | Account object |

### POST /v1/mail/session

**Rate Limit:** 10 requests per minute per user

**Success Response (200):**
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_password_token",
  "account_id": "stalwart_account_id"
}
```

**Error Responses:**
- 401: Unauthorized
- 500: Failed to create mail session

### GET /v1/mail/account

**Success Response (200):**
```json
{
  "address": "alice",
  "status": "active"
}
```

**Error Responses:**
- 401: Unauthorized
- 500: Failed to get mailbox

## Usage/Entitlements Endpoints

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| GET | /v1/account/usage | Bearer token | Get usage statistics | None | Usage object |

### GET /v1/account/usage

**Authorization:** Bearer {{norest_access_token}}

**Success Response (200):**
```json
{
  "domains": {
    "used": 2,
    "limit": 10
  },
  "mailboxes": {
    "used": 2,
    "limit": 50
  },
  "addresses": {
    "used": 2,
    "limit": 50
  }
}
```

## Admin Endpoints

Admin endpoints require both authentication and admin role: `Authorization: Bearer {{norest_access_token}}`

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| POST | /v1/admin/accounts/{id}/suspend | Bearer + Admin | Suspend account | None | 204 No Content |
| POST | /v1/admin/accounts/{id}/reactivate | Bearer + Admin | Reactivate account | None | 204 No Content |

### POST /v1/admin/accounts/{id}/suspend

**Rate Limit:** 5 requests per minute per user

**Path Parameters:**
- `id`: Account UUID

**Success Response:** 204 No Content

**Error Responses:**
- 400: Invalid account ID
- 401: Unauthorized (not admin)
- 500: Internal server error

### POST /v1/admin/accounts/{id}/reactivate

**Rate Limit:** 5 requests per minute per user

**Path Parameters:**
- `id`: Account UUID

**Success Response:** 204 No Content

**Error Responses:**
- 400: Invalid account ID
- 401: Unauthorized (not admin)
- 500: Internal server error

## Billing Webhook Endpoints

| Method | Path | Auth | Purpose | Request | Response |
|--------|------|------|---------|---------|----------|
| POST | /v1/billing/webhook | None | Handle billing webhooks | Webhook payload | `{"processed": boolean}` |

### POST /v1/billing/webhook

**Request Size Limit:** 100KB

**Request:**
```json
{
  "provider": "stripe",
  "event_id": "evt_123",
  "type": "subscription.created",
  "account_id": "uuid",
  "plan_code": "pro",
  "payload_hash": "sha256_hash"
}
```

**Success Response (200):**
```json
{
  "processed": true
}
```

**Error Responses:**
- 400: Invalid request body
- 500: Internal server error

## Rate Limiting

The following endpoints have rate limits:

- `/v1/auth/register`: 5 requests per hour per IP
- `/v1/auth/login`: 10 requests per minute per IP
- `/v1/mail/session`: 10 requests per minute per user
- `/v1/admin/*`: 5 requests per minute per user

## Request Size Limits

- General API endpoints: 1MB max request body
- Billing webhook: 100KB max request body

## Common Error Responses

### 400 Bad Request
```json
{
  "error": "invalid request body"
}
```

### 401 Unauthorized
```json
{
  "error": "unauthorized"
}
```

### 404 Not Found
```json
{
  "error": "resource not found"
}
```

### 409 Conflict
```json
{
  "error": "resource already exists"
}
```

### 500 Internal Server Error
```json
{
  "error": "internal server error"
}
```

### 503 Service Unavailable
```json
{
  "error": "not_ready",
  "detail": "database unhealthy"
}
```

## Total Endpoint Count

**25 public endpoints** across 8 functional groups:

- Health: 6 endpoints
- Authentication: 3 endpoints
- Domains: 6 endpoints
- Addresses: 2 endpoints
- Mail: 2 endpoints
- Usage: 1 endpoint
- Admin: 2 endpoints
- Billing: 1 endpoint

## Security Notes

1. All protected endpoints require JWT Bearer token authentication
2. Admin endpoints require additional admin role verification
3. Rate limiting is applied to sensitive endpoints
4. Request size limits prevent abuse
5. CORS is configurable via ALLOWED_ORIGINS environment variable
6. Development mode allows wildcard CORS; production requires specific origins