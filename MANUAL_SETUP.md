# Manual Mail Server Setup Guide

Complete step-by-step guide to clean and set up a fresh Norest Mail environment manually.

## Prerequisites

- Docker and Docker Compose installed
- Basic command line knowledge
- Access to the server directory

---

## Step 1: Clean Existing Setup

### Stop all services
```bash
cd "/home/ripun/Norest Mail/server"
docker compose down
```

### Remove all volumes (this deletes all data)
```bash
docker compose down -v
```

### Remove old Docker images
```bash
docker images | grep -E "norest|server"
docker rmi server-api:latest server-migrate:latest server-norest-api:latest server-norest-worker-1:latest server-norest-worker-2:latest server-norest-worker-3:latest server-norest-worker:latest server-worker:latest
```

### Verify clean state
```bash
docker ps -a
docker volume ls
```

---

## Step 2: Start Fresh Environment

### Build and start all services
```bash
cd "/home/ripun/Norest Mail/server"
docker compose build
docker compose up -d
```

### Check service status
```bash
docker compose ps
```

Expected output should show all services as "Up":
- postgres
- stalwart  
- norest-api
- norest-worker
- norest-worker-2
- norest-worker-3

---

## Step 3: Run Database Migrations

### Wait for PostgreSQL to be ready
```bash
docker compose exec postgres pg_isready -U norest -d norest
```

### Apply all migrations in order
```bash
cd "/home/ripun/Norest Mail/server"
for migration in migrations/*.sql; do
    echo "Applying $(basename "$migration")..."
    docker compose exec -T postgres psql -U norest -d norest -f /dev/stdin < "$migration"
done
```

Expected migrations:
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

---

## Step 4: Configure Stalwart

### Wait for Stalwart to start
```bash
curl -s http://localhost:8081/
```

### Bootstrap Stalwart (development only)
```bash
curl -s -u admin:change-me-development-only -X POST http://localhost:8081/jmap \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:stalwart:jmap"],
    "methodCalls": [
      ["x:Bootstrap/set", {
        "accountId": "admin",
        "update": {
          "singleton": {
            "serverHostname": "localhost",
            "defaultDomain": "localhost"
          }
        }
      }, "0"]
    ]
  }'
```

### Restart Stalwart
```bash
docker compose restart stalwart
```

### Wait for Stalwart to be ready
```bash
sleep 10
curl -s http://localhost:8080/health/stalwart
```

Expected response: `{"service":"stalwart","status":"ok"}`

---

## Step 5: Verify Services

### Check API health
```bash
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}`

### Check database health
```bash
curl http://localhost:8080/health/db
```

Expected: `{"service":"database","status":"ok"}`

### Check Stalwart health
```bash
curl http://localhost:8080/health/stalwart
```

Expected: `{"service":"stalwart","status":"ok"}`

---

## Step 6: Create Test User

### Register a new user
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePassword123!"
  }'
```

Save the response - you'll need the `access_token` for next steps.

---

## Step 7: Create Domain

### Create a domain (replace TOKEN with your access token)
```bash
curl -X POST http://localhost:8080/v1/domains \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'
```

Save the `id` from the response - this is your domain_id.

---

## Step 8: Create Email Address

### Create an email address (replace TOKEN and DOMAIN_ID)
```bash
curl -X POST http://localhost:8080/v1/domains/YOUR_DOMAIN_ID/addresses \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"local_part":"alice"}'
```

### Wait for provisioning (30-60 seconds)
```bash
sleep 35
```

### Verify address is active
```bash
curl -X GET http://localhost:8080/v1/domains/YOUR_DOMAIN_ID/addresses \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Status should be `"active"`.

---

## Step 9: Get Mail Session

### Request JMAP session
```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Save the response:
- `access_token` - This is your JMAP app password
- `account_id` - Your Stalwart account ID

---

## Step 10: Test Mail Functionality

### Get mailboxes
```bash
curl -X POST http://localhost:8081/jmap \
  -u "alice@example.com:YOUR_JMAP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "YOUR_ACCOUNT_ID"}, "0"]]
  }'
```

### Get sender identity
```bash
curl -X POST http://localhost:8081/jmap \
  -u "alice@example.com:YOUR_JMAP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:submission"],
    "methodCalls": [["Identity/get", {"accountId": "YOUR_ACCOUNT_ID"}, "0"]]
  }'
```

### Create and send test email
```bash
curl -X POST http://localhost:8081/jmap \
  -u "alice@example.com:YOUR_JMAP_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:mail",
      "urn:ietf:params:jmap:submission"
    ],
    "methodCalls": [
      [
        "Email/set",
        {
          "accountId": "YOUR_ACCOUNT_ID",
          "create": {
            "msg1": {
              "mailboxIds": {"d": true},
              "from": [{"email": "alice@example.com"}],
              "to": [{"email": "test@example.com"}],
              "subject": "Test Email from Fresh Setup",
              "bodyValues": {
                "b1": {"value": "This is a test email from our fresh Norest Mail setup!"}
              },
              "textBody": [{"partId": "b1", "type": "text/plain"}]
            }
          }
        },
        "0"
      ],
      [
        "EmailSubmission/set",
        {
          "accountId": "YOUR_ACCOUNT_ID",
          "create": {
            "sub1": {"emailId": "#msg1", "identityId": "YOUR_IDENTITY_ID"}
          }
        },
        "1"
      ]
    ]
  }'
```

---

## Service URLs

After setup, these are your service endpoints:

- **Norest API**: http://localhost:8080
- **Norest API v1**: http://localhost:8080/v1
- **Stalwart Admin**: http://localhost:8081/admin
- **Stalwart JMAP**: http://localhost:8081/jmap
- **PostgreSQL**: localhost:5433

---

## Troubleshooting

### Services won't start
```bash
# Check port conflicts
lsof -i :8080
lsof -i :8081
lsof -i :5433

# Check service logs
docker compose logs
docker compose logs postgres
docker compose logs stalwart
docker compose logs norest-api
```

### Database connection issues
```bash
# Verify PostgreSQL is ready
docker compose exec postgres pg_isready -U norest

# Check DATABASE_URL in .env file
cat .env | grep DATABASE_URL
```

### Stalwart connection issues
```bash
# Verify Stalwart is running
curl http://localhost:8081/

# Check Stalwart logs
docker compose logs stalwart

# Restart Stalwart
docker compose restart stalwart
```

### Migration errors
```bash
# Check PostgreSQL logs
docker compose logs postgres

# Re-run specific migration
docker compose exec -T postgres psql -U norest -d norest -f /dev/stdin < migrations/001_users.sql
```

---

## Quick Reference Commands

### Stop all services
```bash
docker compose down
```

### Start all services
```bash
docker compose up -d
```

### View logs
```bash
docker compose logs -f
```

### Restart specific service
```bash
docker compose restart stalwart
docker compose restart norest-api
```

### Check service status
```bash
docker compose ps
```

### Health checks
```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/db
curl http://localhost:8080/health/stalwart
```

---

## Cleanup Commands

### Complete reset (deletes all data)
```bash
docker compose down -v
docker volume rm server_postgres-data server_stalwart-data server_stalwart-etc
docker images | grep -E "norest|server" | awk '{print $3}' | xargs docker rmi
```

### Remove only containers (keep data)
```bash
docker compose down
```

---

## Notes

- The `.env` file contains development defaults - do not use in production
- Stalwart admin credentials: `admin:change-me-development-only`
- All data is stored in Docker volumes and persists between restarts
- Use `./scripts/dev-reset.sh --force` for automated cleanup (equivalent to Step 1)
- Use `./scripts/dev-up.sh` for automated setup (equivalent to Steps 2-4)

---

## Additional Resources

- API Documentation: `docs/API_REFERENCE.md`
- Startup Guide: `docs/STARTUP.md`
- Quickstart Guide: `docs/QUICKSTART.md`
- Authentication: `docs/AUTHENTICATION.md`
- Mail Operations: `docs/HOW_TO_*.md`