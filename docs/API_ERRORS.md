# API Error Reference

Complete reference of error codes and responses for Norest Mail APIs.

## HTTP Status Codes

### 200 OK
Request succeeded successfully.

### 201 Created
Resource created successfully (used for POST operations that create resources).

### 204 No Content
Request succeeded but no content returned (used for DELETE operations).

### 400 Bad Request
Invalid request parameters or malformed request body.

**Example Response:**
```json
{
  "error": "invalid request body"
}
```

**Common Causes:**
- Invalid JSON
- Missing required fields
- Invalid field format
- Invalid UUID format

### 401 Unauthorized
Authentication failed or missing.

**Example Response:**
```json
{
  "error": "unauthorized"
}
```

**Common Causes:**
- Missing or invalid access token
- Expired access token
- Invalid token signature
- User account suspended

### 403 Forbidden
Request understood but refused due to permissions or limits.

**Example Response:**
```json
{
  "error": "quota exceeded"
}
```

**Common Causes:**
- Quota limit exceeded
- Account suspended
- Insufficient permissions

### 404 Not Found
Resource not found.

**Example Response:**
```json
{
  "error": "domain not found"
}
```

**Common Causes:**
- Invalid resource ID
- Resource deleted
- User doesn't own resource

### 409 Conflict
Resource already exists or conflicts with existing data.

**Example Response:**
```json
{
  "error": "domain already exists"
}
```

**Common Causes:**
- Duplicate resource creation
- Conflicting state

### 429 Too Many Requests
Rate limit exceeded.

**Example Response:**
```json
{
  "error": "rate limit exceeded"
}
```

**Common Causes:**
- Too many requests in time window
- API abuse

### 500 Internal Server Error
Server error processing request.

**Example Response:**
```json
{
  "error": "internal server error"
}
```

**Common Causes:**
- Database connection error
- Stalwart connection error
- Unexpected server error

### 503 Service Unavailable
Service temporarily unavailable.

**Example Response:**
```json
{
  "error": "not_ready",
  "detail": "database unhealthy"
}
```

**Common Causes:**
- Database connection failure
- Stalwart connection failure
- Service startup in progress

## Norest API Error Codes

### Authentication Errors

**Invalid Email**
- Code: `invalid email address`
- Status: 400
- Cause: Email format is invalid

**Password Too Weak**
- Code: `password must be at least 8 characters long`
- Status: 400
- Cause: Password is too short

**Invalid Credentials**
- Code: `invalid email or password`
- Status: 401
- Cause: Wrong email or password

**User Exists**
- Code: (same as message)
- Status: 409
- Cause: Email already registered

**Invalid Token**
- Code: `invalid or expired token`
- Status: 401
- Cause: Token is invalid or expired

### Domain Errors

**Invalid Domain**
- Code: `invalid domain`
- Status: 400
- Cause: Domain format is invalid

**Domain Exists**
- Code: `domain already exists`
- Status: 409
- Cause: Domain name already exists for user

**Domain Not Found**
- Code: `domain not found`
- Status: 404
- Cause: Domain ID doesn't exist or user doesn't own it

**Invalid Domain ID**
- Code: `invalid domain id`
- Status: 400
- Cause: Domain ID is not a valid UUID

### Address Errors

**Invalid Local Part**
- Code: `invalid local part`
- Status: 400
- Cause: Local part format is invalid

**Address Exists**
- Code: `address already exists`
- Status: 409
- Cause: Address already exists in domain

**Quota Exceeded**
- Code: `quota exceeded`
- Status: 403
- Cause: Plan limit reached

**Account Suspended**
- Code: `account suspended`
- Status: 403
- Cause: User account is suspended

### Mail Errors

**Mailbox Not Active**
- Code: `mailbox is not active (status: {status})`
- Status: 500
- Cause: Mailbox not fully provisioned

**Mailbox Not Provisioned**
- Code: `mailbox not fully provisioned in stalwart`
- Status: 500
- Cause: Stalwart account ID not set

### Admin Errors

**Invalid Account ID**
- Code: `invalid account id`
- Status: 400
- Cause: Account ID is not a valid UUID

## JMAP Error Codes

### Core JMAP Errors

**invalidArguments**
- Type: `invalidArguments`
- Description: Request arguments are invalid
- Cause: Invalid method parameters, missing required fields

**notFound**
- Type: `notFound`
- Description: Requested resource not found
- Cause: Invalid ID, resource deleted

**permissionDenied**
- Type: `permissionDenied`
- Description: Insufficient permissions
- Cause: User lacks permission for operation

**serverFail**
- Type: `serverFail`
- Description: Internal server error
- Cause: Server-side error

**rateLimit**
- Type: `rateLimit`
- Description: Rate limit exceeded
- Cause: Too many requests

### Mail JMAP Errors

**alreadyExists**
- Type: `alreadyExists`
- Description: Resource already exists
- Cause: Duplicate creation attempt

**noPrivateKey**
- Type: `noPrivateKey`
- Description: No private key available
- Cause: Encryption/decryption error

### Stalwart Management Errors

**invalidArguments**
- Invalid management operation parameters

**alreadyExists**
- Domain or account already exists

**notFound**
- Domain or account not found

**notCreated**
- Failed to create resource
- Details include specific reason

**notUpdated**
- Failed to update resource
- Details include specific reason

## Error Response Format

### Norest API Errors

**Standard Format:**
```json
{
  "error": "error message"
}
```

**With Details (for health checks):**
```json
{
  "error": "not_ready",
  "detail": "database unhealthy"
}
```

### JMAP Errors

**Standard Format:**
```json
{
  "methodResponses": [
    [
      "error",
      {
        "type": "error_type",
        "description": "Error description"
      },
      "client_request_id"
    ]
  ]
}
```

**With notCreated/notUpdated:**
```json
{
  "methodResponses": [
    [
      "x:Domain/set",
      {
        "notCreated": {
          "create_key": {
            "type": "alreadyExists"
          }
        }
      },
      "0"
    ]
  ]
}
```

## Common Error Scenarios

### Registration Fails

**400 Invalid Email:**
```json
{
  "error": "invalid email address"
}
```

**409 User Exists:**
```json
{
  "error": "user already exists"
}
```

### Login Fails

**401 Invalid Credentials:**
```json
{
  "error": "invalid email or password"
}
```

### Domain Creation Fails

**400 Invalid Domain:**
```json
{
  "error": "invalid domain"
}
```

**409 Domain Exists:**
```json
{
  "error": "domain already exists"
}
```

**403 Quota Exceeded:**
```json
{
  "error": "quota exceeded"
}
```

### Address Creation Fails

**400 Invalid Local Part:**
```json
{
  "error": "invalid local part"
}
```

**409 Address Exists:**
```json
{
  "error": "address already exists"
}
```

**404 Domain Not Found:**
```json
{
  "error": "domain not found"
}
```

### Mail Session Fails

**500 Mailbox Not Active:**
```json
{
  "error": "failed to create mail session"
}
```

**Cause:** Mailbox not fully provisioned or account suspended

### JMAP Operation Fails

**Invalid Arguments:**
```json
{
  "methodResponses": [
    [
      "error",
      {
        "type": "invalidArguments",
        "description": "Invalid email ID"
      },
      "0"
    ]
  ]
}
```

**Not Found:**
```json
{
  "methodResponses": [
    [
      "error",
      {
        "type": "notFound",
        "description": "Email not found"
      },
      "0"
    ]
  ]
}
```

## Error Handling Best Practices

### Client-Side

1. **Check HTTP status first** - Handle by status code category
2. **Parse error message** - Display user-friendly messages
3. **Implement retry logic** - For 5xx errors and rate limits
4. **Validate input** - Prevent common errors before sending
5. **Log errors** - For debugging and monitoring

### Server-Side

1. **Use appropriate status codes** - Match HTTP semantics
2. **Provide clear error messages** - Help users understand issues
3. **Log detailed errors** - For debugging without exposing internals
4. **Validate input early** - Fail fast with clear errors
5. **Handle rate limits** - Return clear retry information

## Rate Limiting Errors

### Endpoints with Rate Limits

- `/v1/auth/register`: 5 requests/hour/IP
- `/v1/auth/login`: 10 requests/minute/IP
- `/v1/mail/session`: 10 requests/minute/user
- `/v1/admin/*`: 5 requests/minute/user

### Rate Limit Response

**Status:** 429 Too Many Requests

**Headers:**
```
Retry-After: 60
```

**Response:**
```json
{
  "error": "rate limit exceeded"
}
```

## Database Errors

### Connection Errors

**Status:** 503 Service Unavailable

**Response:**
```json
{
  "error": "not_ready",
  "detail": "database unhealthy"
}
```

**Handling:**
- Wait and retry
- Check database health endpoint
- Verify connection string

### Query Errors

**Status:** 500 Internal Server Error

**Response:**
```json
{
  "error": "internal server error"
}
```

**Handling:**
- Log detailed error server-side
- Return generic error to client
- Investigate logs

## Stalwart Errors

### Connection Errors

**Status:** 503 Service Unavailable

**Response:**
```json
{
  "error": "not_ready",
  "detail": "stalwart unhealthy"
}
```

**Handling:**
- Wait and retry
- Check Stalwart health endpoint
- Verify Stalwart URL and credentials

### Management Operation Errors

**Status:** 500 Internal Server Error

**Response:**
```json
{
  "error": "failed to create domain in Stalwart"
}
```

**Handling:**
- Check Stalwart logs
- Verify domain/account doesn't already exist
- Check for quota limits in Stalwart

## Validation Errors

### UUID Validation

**Invalid UUID:**
```json
{
  "error": "invalid domain id"
}
```

**Format:** UUID v4 string (e.g., `123e4567-e89b-12d3-a456-426614174000`)

### Email Validation

**Invalid Email:**
```json
{
  "error": "invalid email address"
}
```

**Format:** RFC 5322 email address

### Domain Validation

**Invalid Domain:**
```json
{
  "error": "invalid domain"
}
```

**Format:** Valid domain name (RFC 1035)

## Security Errors

### Suspended Account

**Status:** 403 Forbidden

**Response:**
```json
{
  "error": "account suspended"
}
```

**Handling:**
- Display suspension message
- Provide contact information
- No retry allowed

### Invalid Admin Token

**Status:** 401 Unauthorized

**Response:**
```json
{
  "error": "unauthorized"
}
```

**Handling:**
- Check user has admin role
- Verify token is valid
- Re-authenticate if needed

## Debugging Errors

### Check Health Endpoints

```bash
curl http://localhost:8080/health/db
curl http://localhost:8080/health/stalwart
curl http://localhost:8080/health/ready
```

### Check Service Logs

```bash
docker-compose logs norest-api
docker-compose logs norest-worker
docker-compose logs postgres
docker-compose logs stalwart
```

### Check Database State

```bash
docker-compose exec postgres psql -U norest -d norest
```

### Check Stalwart State

```bash
curl http://localhost:8081/health
```

## Error Prevention

### Input Validation

Validate before sending:
- Email format
- UUID format
- Domain format
- Password strength
- Required fields

### Idempotency

Design operations to be idempotent:
- Use unique create keys
- Check for existing resources
- Handle conflicts gracefully

### Timeout Handling

Set appropriate timeouts:
- API requests: 15-30 seconds
- Database operations: 5-10 seconds
- Stalwart operations: 10-30 seconds

### Circuit Breaking

Implement circuit breakers for:
- Database connections
- Stalwart connections
- External services

## Monitoring

### Key Metrics

- Error rate by endpoint
- Error rate by status code
- Error rate by error type
- Latency by error type
- Success rate

### Alerting

Alert on:
- High error rate (> 5%)
- High 5xx error rate (> 1%)
- Database connection failures
- Stalwart connection failures
- Authentication failures

## Additional Resources

- [API_REFERENCE.md](API_REFERENCE.md) - API documentation
- [AUTHENTICATION.md](AUTHENTICATION.md) - Authentication details
- [JMAP_MAIL_API.md](JMAP_MAIL_API.md) - JMAP error handling