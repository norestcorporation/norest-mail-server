# JMAP Authentication Flow and Requirements

## Authentication Method

**Method**: HTTP Basic Authentication

**Standard**: RFC 7617 (HTTP Basic Authentication)

## Norest → Stalwart/JMAP Adapter Authentication

### Server-Side Authentication (Norest Backend)

The Norest backend uses admin credentials for Stalwart management operations:

- **Credentials**: `STALWART_ADMIN_USER` and `STALwart_ADMIN_PASSWORD` environment variables
- **Purpose**: Server-side provisioning operations (domain/account creation, quota management)
- **Authentication**: HTTP Basic Auth with admin credentials
- **Used by**: Provisioning workers only

### Client-Side Authentication (Frontend → Stalwart)

The frontend authenticates directly with Stalwart using user credentials:

- **Credentials**: Email address + AppPassword
- **Purpose**: JMAP mail operations (mailbox access, email operations)
- **Authentication**: HTTP Basic Auth with user email and AppPassword
- **Used by**: Frontend JMAP client

## JMAP Authentication Flow

### 1. User Requests Mail Session

**Request**:
```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer {{norest_access_token}}"
```

**Authentication**: Norest JWT Bearer token

**Response**:
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "c"
}
```

### 2. Frontend Uses JMAP with AppPassword

**Request**:
```bash
curl -X POST http://localhost:8081/jmap \
  -u "user@example.com:app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "c"}, "0"]]
  }'
```

**Authentication**: HTTP Basic Auth
- **Username**: Email address (e.g., `user@example.com`)
- **Password**: AppPassword from mail session

## Credentials Required

### For Mail Session Creation
- **Norest JWT Access Token**: Obtained from `/v1/auth/register` or `/v1/auth/login`
- **Scope**: Must be a valid, non-expired token

### For JMAP Operations
- **Email Address**: Full email address (e.g., `user@example.com`)
- **AppPassword**: Short-lived token from mail session endpoint
- **Stalwart Account ID**: Provided in mail session response

## Stalwart Account/Identity Used

The JMAP operations use the **user's provisioned Stalwart account**:

- **Account**: Created during address provisioning
- **Account ID**: Provided in mail session response
- **Permissions**: Limited to the user's own mailboxes and data
- **Isolation**: Each user has a separate Stalwart account

## Frontend Security

### Frontend Never Needs Direct Stalwart Credentials

The frontend **never needs**:
- Stalwart admin credentials (`STALWART_ADMIN_USER`, `STALWART_ADMIN_PASSWORD`)
- Direct database access
- Direct Stalwart management API access

The frontend only needs:
- Norest JWT token for Norest API
- Email address + AppPassword for JMAP operations

## Exact cURL Example

### Complete Flow for New Account

```bash
# 1. Register user
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"newuser@example.com","password":"SecurePassword123!"}'

# Save the access_token from response

# 2. Create domain
export TOKEN="your_access_token_here"
curl -X POST http://localhost:8080/v1/domains \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'

# Save the domain_id from response

# 3. Create address
export DOMAIN_ID="your_domain_id_here"
curl -X POST http://localhost:8080/v1/domains/$DOMAIN_ID/addresses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"local_part":"alice"}'

# Wait 30 seconds for provisioning

# 4. Get mail session
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer $TOKEN"

# Save the access_token (AppPassword) and account_id from response

# 5. Get mailboxes
export EMAIL="alice@example.com"
export APP_PASSWORD="your_app_password_here"
export ACCOUNT_ID="your_account_id_here"
curl -X POST http://localhost:8081/jmap \
  -u "$EMAIL:$APP_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "$ACCOUNT_ID"}, "0"]]
  }'
```

## Security Summary

| Layer | Authentication | Credentials | Purpose |
|-------|-----------------|-------------|---------|
| Norest API | JWT Bearer Token | User JWT token | API authentication |
| Norest → Stalwart (Management) | HTTP Basic Auth | Admin credentials | Server-side provisioning |
| Frontend → Stalwart (JMAP) | HTTP Basic Auth | Email + AppPassword | Mail operations |

## Verification Against Implementation

The above flow has been verified against the actual implementation:

1. **Mail Service** (`internal/mail/service.go`): 
   - Creates AppPassword via `stalwart.CreateAppPassword()`
   - Returns Stalwart Account ID in session response

2. **Stalwart Client** (`internal/stalwart/management.go`):
   - Uses admin credentials for management operations
   - Supports both Basic Auth for management

3. **JMAP Standard**:
   - Uses HTTP Basic Auth for authentication
   - Username: email address
   - Password: AppPassword or account password

## No Production Secrets in Example

The example uses placeholder values:
- `your_access_token_here`
- `your_domain_id_here`
- `your_app_password_here`
- `your_account_id_here`

No real passwords or secrets are exposed.