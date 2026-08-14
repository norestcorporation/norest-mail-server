# Norest Mail API Index

**Frozen API Contract - Chapter 5 Release**

Quick reference for developers working with Norest Mail. See [API_CONTRACT_FROZEN.md](API_CONTRACT_FROZEN.md) for the frozen API contract state.

## Service URLs

### Development
- **Norest API**: http://localhost:8080/v1
- **Stalwart JMAP**: http://localhost:8081/jmap
- **Stalwart Discovery**: http://localhost:8081/.well-known/jmap
- **PostgreSQL**: localhost:5433

### Production
- Configure via environment variables
- Use HTTPS for all services
- Use managed services for PostgreSQL and Stalwart

## Authentication

### Norest API
- **Type**: JWT Bearer Token
- **Header**: `Authorization: Bearer {{token}}`
- **Lifetime**: 15 minutes (access), 7 days (refresh)
- **Docs**: [AUTHENTICATION.md](AUTHENTICATION.md)

### JMAP Mail
- **Type**: HTTP Basic Auth
- **Format**: `username:app_password`
- **Header**: `Authorization: Basic {{credentials}}`
- **Docs**: [HOW_TO_GET_MAIL.md](HOW_TO_GET_MAIL.md)

### Stalwart Management
- **Type**: HTTP Basic Auth
- **Credentials**: Admin username/password (server-side only)
- **Docs**: [STALWART_MANAGEMENT_API.md](STALWART_MANAGEMENT_API.md)

## API Endpoints

### Health (6 endpoints)
- GET /health
- GET /health/live
- GET /health/ready
- GET /health/db
- GET /health/stalwart
- GET /metrics

### Authentication (3 endpoints)
- POST /v1/auth/register
- POST /v1/auth/login
- GET /v1/me

### Domains (6 endpoints)
- POST /v1/domains
- GET /v1/domains
- GET /v1/domains/{id}
- DELETE /v1/domains/{id}
- POST /v1/domains/{id}/verification/start
- GET /v1/domains/{id}/verification

### Addresses (2 endpoints)
- POST /v1/domains/{domainID}/addresses
- GET /v1/domains/{domainID}/addresses

### Mail (2 endpoints)
- POST /v1/mail/session
- GET /v1/mail/account

### Usage (1 endpoint)
- GET /v1/account/usage

### Admin (2 endpoints)
- POST /v1/admin/accounts/{id}/suspend
- POST /v1/admin/accounts/{id}/reactivate

### Billing (1 endpoint)
- POST /v1/billing/webhook

**Total**: 25 public endpoints

## JMAP Methods Used

### Mail Operations (7 methods)
- Mailbox/get
- Identity/get
- Email/set (create, update)
- Email/query
- Email/get
- EmailSubmission/set (create)

### Capabilities
- urn:ietf:params:jmap:core
- urn:ietf:params:jmap:mail
- urn:ietf:params:jmap:submission

**Docs**: [JMAP_MAIL_API.md](JMAP_MAIL_API.md)

## Stalwart Management Methods (12 methods)

### Domain Operations
- x:Domain/set (create)
- x:Domain/get (query, existence, find by name)

### Account Operations
- x:Account/set (create, update quota, enable/disable)
- x:Account/get (query, existence, find by name)

### App Password Operations
- x:AppPassword/set (create)

**Docs**: [STALWART_MANAGEMENT_API.md](STALWART_MANAGEMENT_API.md)

## Environment Variables

### Required
- DATABASE_URL
- STALWART_BASE_URL
- STALWART_ADMIN_USER
- STALWART_ADMIN_PASSWORD
- JWT_SECRET

### Optional
- APP_ENV (default: development)
- HTTP_ADDR (default: :8080)
- ALLOWED_ORIGINS (default: empty/wildcard)
- DB_* pool settings
- Worker configuration
- STALWART_PUBLIC_URL
- STALWART_RECOVERY_ADMIN

**Docs**: [ENVIRONMENT.md](ENVIRONMENT.md)

## Startup Commands

### Development
```bash
cp .env.example .env
./scripts/dev-up.sh
```

### Manual Startup
```bash
cp .env.example .env
docker-compose up -d
for migration in migrations/*.sql; do
    echo "Applying $(basename "$migration")..."
    docker-compose exec -T postgres psql -U norest -d norest -f /dev/stdin < "$migration"
done
```

### Verification
```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/db
curl http://localhost:8080/health/stalwart
```

### Stop
```bash
docker-compose down
```

**Docs**: [STARTUP.md](STARTUP.md)

## Postman Collection

### Collection
- **Path**: postman/Norest-Mail.postman_collection.json
- **Environment**: postman/Norest-Mail.postman_environment.json
- **Folders**: 13 folders covering all API groups
- **Auto-save**: Tokens and IDs automatically saved

### Structure
- 01 Health
- 02 Authentication
- 03 Domains
- 04 Addresses
- 05 Mail Session
- 06 Usage
- 07 JMAP - Session
- 08 JMAP - Mailboxes
- 09 JMAP - Emails
- 10 JMAP - Submission
- 11 Stalwart Management (DEV ONLY)
- 12 Billing
- 13 Admin

**Docs**: [postman/README.md](postman/README.md)

## How-To Guides

### Mail Operations
- [HOW_TO_GET_MAIL.md](HOW_TO_GET_MAIL.md) - Retrieve emails
- [HOW_TO_SEND_MAIL.md](HOW_TO_SEND_MAIL.md) - Send emails
- [HOW_TO_READ_MAIL.md](HOW_TO_READ_MAIL.md) - Read email content
- [HOW_TO_MARK_READ.md](HOW_TO_MARK_READ.md) - Mark read/unread
- [HOW_TO_STAR.md](HOW_TO_STAR.md) - Star/flag emails
- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move between folders
- [HOW_TO_DELETE_MAIL.md](HOW_TO_DELETE_MAIL.md) - Delete emails
- [SEARCH.md](SEARCH.md) - Search emails
- [DRAFTS.md](DRAFTS.md) - Draft management

## Error Handling

### HTTP Status Codes
- 200 OK - Success
- 201 Created - Resource created
- 204 No Content - Success with no content
- 400 Bad Request - Invalid request
- 401 Unauthorized - Authentication failed
- 403 Forbidden - Permission denied
- 404 Not Found - Resource not found
- 409 Conflict - Resource exists
- 429 Too Many Requests - Rate limit
- 500 Internal Server Error - Server error
- 503 Service Unavailable - Service unavailable

**Docs**: [API_ERRORS.md](API_ERRORS.md)

## Rate Limiting

- Register: 5 requests/hour/IP
- Login: 10 requests/minute/IP
- Mail Session: 10 requests/minute/user
- Admin: 5 requests/minute/user

## Worker Configuration

- **Default Workers**: 3 (worker-1, worker-2, worker-3)
- **Job Lease**: 60 seconds
- **Job Heartbeat**: 20 seconds
- **Max Attempts**: 10
- **Max Backoff**: 300 seconds
- **Max Concurrent Jobs**: 5 per worker

## Database Schema

### Tables
- users
- product_accounts
- user_product_accounts
- plans
- subscriptions
- domains
- addresses
- mailboxes
- provisioning_jobs
- billing_events

## Testing

### Baseline Test
```bash
./scripts/test-baseline.sh
```

### Chapter Tests
```bash
./scripts/test-chapter2.sh
./scripts/test-chapter3.sh
./scripts/test-chapter4-full.sh
```

### Go Tests
```bash
go test ./...
go build ./...
```

## Security Notes

### Development
- HTTP allowed
- Wildcard CORS allowed
- Development default secrets accepted
- Detailed error messages

### Production
- HTTPS required
- Specific CORS origins required
- Strong secrets required
- Generic error messages
- Security headers enforced

### Credentials
- Never commit secrets to version control
- Use secrets management in production
- Rotate credentials regularly
- Different secrets per environment

## Documentation Files

### Core Documentation
- [API_CONTRACT_FROZEN.md](API_CONTRACT_FROZEN.md) - Frozen API contract state
- [API_REFERENCE.md](API_REFERENCE.md) - Complete API reference
- [AUTHENTICATION.md](AUTHENTICATION.md) - Authentication documentation
- [STARTUP.md](STARTUP.md) - Startup guide
- [ENVIRONMENT.md](ENVIRONMENT.md) - Environment variables
- [QUICKSTART.md](QUICKSTART.md) - Quick start guide

### API Documentation
- [STALWART_MANAGEMENT_API.md](STALWART_MANAGEMENT_API.md) - Stalwart management APIs
- [JMAP_MAIL_API.md](JMAP_MAIL_API.md) - JMAP mail APIs
- [API_ERRORS.md](API_ERRORS.md) - Error reference

### How-To Guides
- [HOW_TO_GET_MAIL.md](HOW_TO_GET_MAIL.md) - Get mail
- [HOW_TO_SEND_MAIL.md](HOW_TO_SEND_MAIL.md) - Send mail
- [HOW_TO_READ_MAIL.md](HOW_TO_READ_MAIL.md) - Read mail
- [HOW_TO_MARK_READ.md](HOW_TO_MARK_READ.md) - Mark read/unread
- [HOW_TO_STAR.md](HOW_TO_STAR.md) - Star emails
- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move emails
- [HOW_TO_DELETE_MAIL.md](HOW_TO_DELETE_MAIL.md) - Delete emails
- [SEARCH.md](SEARCH.md) - Search emails
- [DRAFTS.md](DRAFTS.md) - Drafts

### Testing
- [postman/Norest-Mail.postman_collection.json](postman/Norest-Mail.postman_collection.json) - Postman collection
- [postman/Norest-Mail.postman_environment.json](postman/Norest-Mail.postman_environment.json) - Postman environment
- [postman/README.md](postman/README.md) - Postman guide

## Key Concepts

### Provisioning Flow
1. User creates domain in Norest
2. Domain record created with status "pending"
3. Provisioning job created
4. Worker provisions domain in Stalwart
5. Domain status changes to "active"
6. stalwart_domain_id populated

### Mailbox Flow
1. User creates address in Norest
2. Address and mailbox records created
3. Provisioning job created
4. Worker provisions account in Stalwart
5. Mailbox status changes to "active"
6. stalwart_account_id populated

### Mail Session Flow
1. User requests mail session from Norest
2. Norest verifies mailbox is active
3. Norest creates AppPassword in Stalwart
4. Norest returns AppPassword and account_id
5. Client uses AppPassword for JMAP authentication

### Multi-Tenant Isolation
- Each Norest domain maps to unique Stalwart domain ID
- Each Norest mailbox maps to unique Stalwart account ID
- Users can only access their own mailboxes
- Admin credentials are server-side only

## Support

For issues or questions:
1. Check service logs: `docker-compose logs`
2. Review documentation in `docs/` directory
3. Check health endpoints
4. Verify environment configuration
