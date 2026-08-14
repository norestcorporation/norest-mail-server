# Norest Mail Quickstart

Get Norest Mail backend running in 5 minutes.

## Prerequisites

- Go 1.26+
- Docker & Docker Compose
- Git

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd Norest\ Mail/server
cp .env.example .env
```

### 2. Start Services and Run Migrations

**Recommended: Use the development startup script**

```bash
./scripts/dev-up.sh
```

This automated script handles the complete startup process including:
- Building and starting all services
- Running all database migrations
- Bootstrapping Stalwart in development mode
- Waiting for all services to be healthy

**Alternative: Manual startup**

```bash
docker-compose up -d
```

This starts PostgreSQL, Stalwart Mail Server, Norest API, and 3 Norest Workers.

Then manually run migrations:

```bash
# Apply all migrations in order
for migration in migrations/*.sql; do
    echo "Applying $(basename "$migration")..."
    docker-compose exec -T postgres psql -U norest -d norest -f /dev/stdin < "$migration"
done
```

### 3. Verify Health

```bash
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}`

### 4. Register User

```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePassword123!"}'
```

Save the `access_token` from the response.

### 5. Create Domain

```bash
curl -X POST http://localhost:8080/v1/domains \
  -H "Authorization: Bearer {{your_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'
```

Save the `id` from the response.

### 6. Create Address

```bash
curl -X POST http://localhost:8080/v1/domains/{{domain_id}}/addresses \
  -H "Authorization: Bearer {{your_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{"local_part":"alice"}'
```

Wait 30 seconds for provisioning.

### 7. Get Mail Session

```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer {{your_access_token}}"
```

Save the `access_token` (this is your JMAP app password) and `account_id`.

### 8. Get Mailboxes

```bash
curl -X POST http://localhost:8081/jmap \
  -u "test@example.com:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "{{account_id}}"}, "0"]]
  }'
```

### 9. Query Emails

```bash
curl -X POST http://localhost:8081/jmap \
  -u "test@example.com:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {"accountId": "{{account_id}}"}, "0"]]
  }'
```

## Service URLs

- **Norest API**: http://localhost:8080/v1
- **Stalwart JMAP**: http://localhost:8081/jmap
- **PostgreSQL**: localhost:5433

## Stop Services

```bash
docker-compose down
```

## Common Issues

**Database connection failed**: Ensure PostgreSQL is healthy: `curl http://localhost:8080/health/db`

**Stalwart connection failed**: Ensure Stalwart is healthy: `curl http://localhost:8080/health/stalwart`

**Provisioning timeout**: Wait 30-60 seconds after creating address, check worker logs: `docker-compose logs norest-worker`

## Next Steps

- [API_REFERENCE.md](docs/API_REFERENCE.md) - Complete API documentation
- [Postman Collection](postman/Norest-Mail.postman_collection.json) - Import for API testing
- [STARTUP.md](docs/STARTUP.md) - Detailed startup guide
- [HOW_TO_GET_MAIL.md](docs/HOW_TO_GET_MAIL.md) - Get mail guide
- [HOW_TO_SEND_MAIL.md](docs/HOW_TO_SEND_MAIL.md) - Send mail guide

## Production Deployment

For production, see [STARTUP.md](docs/STARTUP.md) for production configuration requirements:
- Strong secrets
- HTTPS required
- Specific CORS origins
- Managed PostgreSQL
- Clustered Stalwart deployment
