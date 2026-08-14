# Norest Mail Startup Guide

Complete guide for starting the Norest Mail backend from a clean checkout.

## Prerequisites

### Required Software

- **Go**: 1.26 or later
- **Docker**: 20.10 or later
- **Docker Compose**: 2.0 or later

### Verify Installation

```bash
go version
docker --version
docker-compose --version
```

## Quick Start (Development)

### 1. Clone Repository

```bash
git clone <repository-url>
cd Norest\ Mail/server
```

### 2. Setup Environment

```bash
cp .env.example .env
```

The `.env.example` file contains development defaults suitable for local testing.

### 3. Start Services and Run Migrations

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

This starts:
- PostgreSQL (port 5433)
- Stalwart Mail Server (port 8081)
- Norest API (port 8080)
- 3 Norest Workers

Then manually run migrations (see step 5 below).

### 4. Verify Health

```bash
# Check API health
curl http://localhost:8080/health

# Check database health
curl http://localhost:8080/health/db

# Check Stalwart health
curl http://localhost:8080/health/stalwart

# Check readiness
curl http://localhost:8080/health/ready
```

Expected response: `{"status": "ok"}` or `{"status": "ready"}`

### 5. Run Database Migrations

The repository includes a development startup script that automates the complete migration process:

```bash
./scripts/dev-up.sh
```

This script:
- Builds and starts all services
- Waits for PostgreSQL to be healthy
- Applies ALL migrations in the migrations directory in order
- Waits for Stalwart to be ready
- Bootstraps Stalwart in development mode
- Waits for the Norest API to be healthy

**Alternatively**, if you prefer to run migrations manually after starting services:

```bash
# Apply all migrations in order
for migration in migrations/*.sql; do
    echo "Applying $(basename "$migration")..."
    docker-compose exec -T postgres psql -U norest -d norest -f /dev/stdin < "$migration"
done
```

The current migration set includes:
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

### 6. Test Registration

```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePassword123!"
  }'
```

Expected response with access token and refresh token.

## Service URLs

### Development Environment

- **Norest API**: http://localhost:8080
- **Norest API v1**: http://localhost:8080/v1
- **Stalwart Mail**: http://localhost:8081
- **Stalwart JMAP**: http://localhost:8081/jmap
- **Stalwart Discovery**: http://localhost:8081/.well-known/jmap
- **PostgreSQL**: localhost:5433

### Production Environment

In production, these URLs would be configured differently:
- Use HTTPS for all services
- Use domain names instead of localhost
- Configure proper DNS for Stalwart
- Use managed PostgreSQL service

## Development vs Production

### Development

```bash
# Development configuration
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://norest:norest@postgres:5432/norest?sslmode=disable
STALWART_BASE_URL=http://stalwart:8080
STALWART_PUBLIC_URL=http://localhost:8081
ALLOWED_ORIGINS=  # Empty allows wildcard CORS
```

**Development Features:**
- HTTP allowed (no HTTPS required)
- Wildcard CORS allowed
- Development default secrets accepted
- Detailed error messages
- Debug logging enabled

### Production

```bash
# Production configuration
APP_ENV=production
HTTP_ADDR=:8080
DATABASE_URL=postgres://user:pass@production-host:5432/norest?sslmode=require
STALWART_BASE_URL=https://mail.example.com
STALWART_PUBLIC_URL=https://mail.example.com
ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
```

**Production Requirements:**
- HTTPS required
- Specific CORS origins required
- Strong secrets required
- Generic error messages
- Info-level logging
- Security headers enforced

## Manual Service Startup

If you prefer to start services individually:

### 1. Start PostgreSQL

```bash
docker-compose up -d postgres
```

Wait for PostgreSQL to be healthy (check health status).

### 2. Start Stalwart

```bash
docker-compose up -d stalwart
```

Wait for Stalwart to be healthy.

### 3. Start Norest API

```bash
docker-compose up -d norest-api
```

### 4. Start Workers

```bash
docker-compose up -d norest-worker norest-worker-2 norest-worker-3
```

## Stopping Services

### Stop All Services

```bash
docker-compose down
```

### Stop and Remove Volumes

```bash
docker-compose down -v
```

**Warning:** This removes all PostgreSQL and Stalwart data.

### Stop Specific Services

```bash
docker-compose stop norest-api
docker-compose stop norest-worker
```

## Viewing Logs

### All Services

```bash
docker-compose logs -f
```

### Specific Service

```bash
docker-compose logs -f norest-api
docker-compose logs -f norest-worker
docker-compose logs -f postgres
docker-compose logs -f stalwart
```

### Specific Worker

```bash
docker-compose logs -f norest-worker-1
docker-compose logs -f norest-worker-2
docker-compose logs -f norest-worker-3
```

## Database Access

### Connect to PostgreSQL

```bash
docker-compose exec postgres psql -U norest -d norest
```

### Run SQL File

```bash
docker-compose exec -T postgres psql -U norest -d norest -f path/to/file.sql
```

### Backup Database

```bash
docker-compose exec postgres pg_dump -U norest norest > backup.sql
```

### Restore Database

```bash
docker-compose exec -T postgres psql -U norest norest < backup.sql
```

## Rebuilding Services

### Rebuild After Code Changes

```bash
docker-compose build norest-api norest-worker
docker-compose up -d norest-api norest-worker norest-worker-2 norest-worker-3
```

### Force Rebuild (No Cache)

```bash
docker-compose build --no-cache norest-api norest-worker
docker-compose up -d norest-api norest-worker norest-worker-2 norest-worker-3
```

## Health Check Endpoints

### Basic Health

```bash
curl http://localhost:8080/health
```

Response: `{"status": "ok"}`

### Liveness Probe

```bash
curl http://localhost:8080/health/live
```

Response: `{"status": "alive"}`

### Readiness Probe

```bash
curl http://localhost:8080/health/ready
```

Response: `{"status": "ready"}` or error if dependencies are unhealthy

### Database Health

```bash
curl http://localhost:8080/health/db
```

Response: `{"status": "ok", "service": "database"}`

### Stalwart Health

```bash
curl http://localhost:8080/health/stalwart
```

Response: `{"status": "ok", "service": "stalwart"}`

### Metrics

```bash
curl http://localhost:8080/metrics
```

Response: Application metrics snapshot

## Worker Configuration

The system runs 3 workers by default for development:

- `norest-worker-1`: WORKER_ID=worker-1
- `norest-worker-2`: WORKER_ID=worker-2
- `norest-worker-3`: WORKER_ID=worker-3

Workers process provisioning jobs asynchronously:
- Domain creation and DNS verification
- Account/mailbox provisioning
- Quota synchronization
- Account suspension/reactivation

## Troubleshooting

### Services Won't Start

1. Check if ports are already in use:
   ```bash
   lsof -i :8080
   lsof -i :8081
   lsof -i :5433
   ```

2. Check Docker service status:
   ```bash
   docker-compose ps
   ```

3. Check service logs:
   ```bash
   docker-compose logs
   ```

### Database Connection Issues

1. Verify PostgreSQL is healthy:
   ```bash
   docker-compose exec postgres pg_isready -U norest
   ```

2. Check DATABASE_URL in .env file

3. Verify migrations were run

### Stalwart Connection Issues

1. Verify Stalwart is healthy:
   ```bash
   curl http://localhost:8081/health
   ```

2. Check STALWART_BASE_URL in .env file

3. Verify Stalwart admin credentials

### Health Checks Failing

1. Check individual health endpoints:
   ```bash
   curl http://localhost:8080/health/db
   curl http://localhost:8080/health/stalwart
   ```

2. Wait for services to become healthy (may take 30-60 seconds)

3. Check logs for specific errors

### Permission Issues

1. Ensure .env file exists and is readable
2. Check file permissions: `ls -la .env`
3. Verify environment variables are set correctly

## Configuration Files

### .env

Main configuration file for environment variables.

### docker-compose.yml

Docker Compose configuration for all services.

### docker/norest/Dockerfile

Dockerfile for building Norest API and Worker images.

### migrations/*.sql

Database migration files.

## Network Configuration

The services use a Docker bridge network named `norest`:

- Services can communicate using service names
- PostgreSQL accessible as `postgres:5432`
- Stalwart accessible as `stalwart:8080`
- API accessible as `norest-api:8080`

## Data Persistence

### PostgreSQL Data

Stored in Docker volume: `postgres-data`

### Stalwart Data

Stored in Docker volumes:
- `stalwart-etc`: Configuration files
- `stalwart-data`: Mail data and user accounts

## Cleaning Up

### Remove All Containers and Networks

```bash
docker-compose down
```

### Remove All Containers, Networks, and Volumes

```bash
docker-compose down -v
```

**Warning:** This deletes all data.

### Remove Specific Volume

```bash
docker volume rm norest_postgres-data
docker volume rm norest_stalwart-data
docker volume rm norest_stalwart-etc
```

## Production Deployment

For production deployment, consider:

1. Use managed PostgreSQL service (AWS RDS, Google Cloud SQL, etc.)
2. Use managed or clustered Stalwart deployment
3. Configure proper SSL/TLS certificates
4. Set up monitoring and alerting
5. Configure log aggregation
6. Set up backup and disaster recovery
7. Use environment-specific configuration
8. Implement secrets management (HashiCorp Vault, AWS Secrets Manager)
9. Configure proper firewall rules
10. Set up CDN for static assets

## Next Steps

After starting the backend:

1. Review [API_REFERENCE.md](API_REFERENCE.md) for available endpoints
2. Review [AUTHENTICATION.md](AUTHENTICATION.md) for authentication details
3. Review [ENVIRONMENT.md](ENVIRONMENT.md) for configuration options
4. Import the Postman collection for API testing
5. Run the baseline test suite: `./scripts/test-baseline.sh`

## Support

For issues or questions:

1. Check service logs: `docker-compose logs`
2. Review documentation in `docs/` directory
3. Check health endpoints
4. Review environment configuration
5. Consult the troubleshooting section above