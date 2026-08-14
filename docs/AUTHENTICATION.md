# Norest Mail Authentication Documentation

## Overview

Norest Mail uses JWT (JSON Web Token) based authentication with Bearer token authorization. The system implements access tokens and refresh tokens with different lifetimes.

## Authentication Flow

### 1. User Registration

**Endpoint:** `POST /v1/auth/register`

**Rate Limit:** 5 requests per hour per IP

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'
```

**Response (201 Created):**
```json
{
  "id": "3b699a46-9e65-45b1-b455-d9122e5604b0",
  "email": "user@example.com",
  "status": "pending",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Validation Rules:**
- Email must be a valid RFC 5322 address
- Password must be at least 8 characters long
- Email must be unique (no duplicate registrations)

**Error Responses:**
- 400: Invalid email format or password too weak
- 409: User already exists

### 2. User Login

**Endpoint:** `POST /v1/auth/login`

**Rate Limit:** 10 requests per minute per IP

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'
```

**Response (200 OK):**
```json
{
  "id": "3b699a46-9e65-45b1-b455-d9122e5604b0",
  "email": "user@example.com",
  "status": "active",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Error Responses:**
- 401: Invalid email or password

### 3. Authenticated Request

All protected endpoints require the access token in the Authorization header:

```bash
curl -X GET http://localhost:8080/v1/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 4. Get Current User Info

**Endpoint:** `GET /v1/me`

**Request:**
```bash
curl -X GET http://localhost:8080/v1/me \
  -H "Authorization: Bearer {{norest_access_token}}"
```

**Response (200 OK):**
```json
{
  "id": "3b699a46-9e65-45b1-b455-d9122e5604b0",
  "email": "user@example.com",
  "status": "active"
}
```

## Token Details

### Access Token

- **Purpose:** Short-lived token for API authentication
- **Lifetime:** 15 minutes
- **Algorithm:** HS256 (HMAC-SHA256)
- **Claims:**
  - `user_id`: UUID of the user
  - `exp`: Expiration timestamp
  - `iat`: Issued at timestamp
  - `nbf`: Not before timestamp

### Refresh Token

- **Purpose:** Long-lived token for obtaining new access tokens
- **Lifetime:** 7 days (168 hours)
- **Algorithm:** HS256 (HMAC-SHA256)
- **Claims:** Same as access token

### Token Structure

Both tokens use the same JWT structure:

```json
{
  "user_id": "3b699a46-9e65-45b1-b455-d9122e5604b0",
  "exp": 1786685036,
  "nbf": 1786684136,
  "iat": 1786684136
}
```

## Token Validation

The authentication middleware validates tokens by:

1. Checking for empty tokens
2. Verifying the signing method (HS256 only)
3. Validating the signature using the JWT secret
4. Checking expiration and not-before timestamps
5. Ensuring user_id is not nil

## Error Handling

### Invalid Token

When a token is invalid or expired:

```json
{
  "error": "unauthorized"
}
```

**HTTP Status:** 401 Unauthorized

### Missing Token

When no token is provided:

```json
{
  "error": "unauthorized"
}
```

**HTTP Status:** 401 Unauthorized

## Token Refresh

**Note:** The current implementation does not include a dedicated refresh endpoint. Users must re-login to obtain new tokens when access tokens expire.

## Security Considerations

### JWT Secret

- The JWT secret is configured via the `JWT_SECRET` environment variable
- In production, this must be a strong, random string
- Development default: `development-only-jwt-secret-do-not-use-in-production`
- Production validation rejects development defaults

### Token Storage

- Tokens should be stored securely on the client side
- Recommended: Use httpOnly cookies or secure storage mechanisms
- Never store tokens in localStorage in production

### HTTPS

- In production, all authentication requests must use HTTPS
- Development mode allows HTTP for local testing

### Token Lifetime

- Access tokens are short-lived (15 minutes) to limit exposure
- Refresh tokens are longer-lived (7 days) for user convenience
- Token lifetimes are currently not configurable

## Admin Authentication

Admin endpoints require:

1. Valid JWT authentication (same as regular users)
2. Admin role verification (via middleware)

**Example Admin Request:**
```bash
curl -X POST http://localhost:8080/v1/admin/accounts/{id}/suspend \
  -H "Authorization: Bearer {{admin_access_token}}"
```

## Mail Session Authentication

Mail operations use a different authentication mechanism:

1. User authenticates with Norest JWT
2. User requests mail session via `POST /v1/mail/session`
3. Norest provisions a Stalwart AppPassword
4. Client uses AppPassword for JMAP authentication

**Mail Session Request:**
```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer {{norest_access_token}}"
```

**Mail Session Response:**
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "stalwart_account_id"
}
```

**JMAP Authentication:**
```bash
curl -X POST http://localhost:8081/jmap \
  -H "Content-Type: application/json" \
  -u "user@example.com:app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [...]
  }'
```

**Important**: JMAP authentication uses HTTP Basic Auth with the email address as username and the AppPassword as password. This is the standard JMAP authentication method.

## Password Security

- Passwords are hashed using bcrypt before storage
- Minimum password length: 8 characters
- Passwords are never returned in API responses
- Passwords are never logged

## User Status

Users can have the following statuses:

- `pending`: Newly registered, not yet activated
- `active`: Fully functional account
- `suspended`: Suspended by admin (cannot authenticate)

Suspended users cannot authenticate even with valid credentials.

## Rate Limiting

Authentication endpoints have rate limits:

- Register: 5 requests per hour per IP
- Login: 10 requests per minute per IP

Rate limits help prevent brute force attacks and abuse.

## Development vs Production

### Development
- Allows weak JWT secrets
- Allows wildcard CORS
- HTTP allowed for local testing
- Default credentials for testing

### Production
- Requires strong JWT secrets
- Requires specific CORS origins
- HTTPS required
- Rejects development defaults
- Rejects wildcard CORS

## Common Issues

### Token Expiration

Access tokens expire after 15 minutes. If you receive 401 errors:

1. Check if your access token is expired
2. Re-login to obtain a new access token
3. Use the new token for subsequent requests

### Invalid Credentials

If login fails with 401:

1. Verify email and password are correct
2. Check if user account exists
3. Check if user account is not suspended

### Rate Limiting

If you receive 429 errors:

1. Wait for the rate limit window to expire
2. Reduce request frequency
3. Contact support if limits are too restrictive

## Configuration

Authentication behavior is configured via environment variables:

- `JWT_SECRET`: Secret key for token signing (required)
- `APP_ENV`: Environment (development/production)
- `ALLOWED_ORIGINS`: CORS allowed origins (comma-separated)

See [ENVIRONMENT.md](ENVIRONMENT.md) for complete configuration reference.