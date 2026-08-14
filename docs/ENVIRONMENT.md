# Norest Mail Environment Variable Reference

Complete reference of all environment variables used by the Norest Mail backend.

## Application Configuration

### APP_ENV

**Description:** Application environment (development, production)

**Default:** `development`

**Required:** No

**Development Value:** `development`

**Production Requirement:** `production`

**Security Notes:** 
- Production mode enables additional security validations
- Rejects development defaults in production
- Requires specific CORS origins in production

**Example:**
```bash
APP_ENV=production
```

### HTTP_ADDR

**Description:** HTTP server listening address

**Default:** `:8080`

**Required:** No

**Development Value:** `:8080`

**Production Requirement:** `:8080` or configured port

**Security Notes:** None

**Example:**
```bash
HTTP_ADDR=:8080
```

## Database Configuration

### DATABASE_URL

**Description:** PostgreSQL connection string

**Default:** None

**Required:** Yes

**Development Value:** `postgres://norest:norest@postgres:5432/norest?sslmode=disable`

**Production Requirement:** Production PostgreSQL connection with SSL

**Security Notes:**
- Production must use SSL (`sslmode=require` or `sslmode=verify-full`)
- Never commit production credentials to version control
- Use connection pooling in production
- Consider using managed PostgreSQL service

**Example:**
```bash
# Development
DATABASE_URL=postgres://norest:norest@postgres:5432/norest?sslmode=disable

# Production
DATABASE_URL=postgres://user:strongpassword@production-db.example.com:5432/norest?sslmode=require
```

### DB_MAX_CONNS

**Description:** Maximum number of database connections in pool

**Default:** `10`

**Required:** No

**Development Value:** `10`

**Production Requirement:** Adjust based on load (typically 20-100)

**Security Notes:** Higher values increase resource usage

**Example:**
```bash
DB_MAX_CONNS=50
```

### DB_MIN_CONNS

**Description:** Minimum number of database connections in pool

**Default:** `2`

**Required:** No

**Development Value:** `2`

**Production Requirement:** Adjust based on expected baseline load

**Security Notes:** Ensures minimum connections are always available

**Example:**
```bash
DB_MIN_CONNS=5
```

### DB_MAX_CONN_LIFETIME

**Description:** Maximum connection lifetime in seconds

**Default:** `1800` (30 minutes)

**Required:** No

**Development Value:** `1800`

**Production Requirement:** Typically 1800-3600 seconds

**Security Notes:** Prevents long-lived connections from accumulating issues

**Example:**
```bash
DB_MAX_CONN_LIFETIME=3600
```

### DB_MAX_CONN_IDLE_TIME

**Description:** Maximum idle time for connections in seconds

**Default:** `300` (5 minutes)

**Required:** No

**Development Value:** `300`

**Production Requirement:** Typically 300-600 seconds

**Security Notes:** Frees idle connections to save resources

**Example:**
```bash
DB_MAX_CONN_IDLE_TIME=600
```

### DB_OPERATION_TIMEOUT

**Description:** Database operation timeout in seconds

**Default:** `30`

**Required:** No

**Development Value:** `30`

**Production Requirement:** Adjust based on query complexity

**Security Notes:** Prevents long-running queries from blocking

**Example:**
```bash
DB_OPERATION_TIMEOUT=60
```

## Stalwart Configuration

### STALWART_BASE_URL

**Description:** Base URL for Stalwart Mail Server (internal/admin access)

**Default:** None

**Required:** Yes

**Development Value:** `http://stalwart:8080`

**Production Requirement:** Production Stalwart URL

**Security Notes:**
- Used for administrative operations
- Should be internal URL in production
- Should use HTTPS in production

**Example:**
```bash
# Development
STALWART_BASE_URL=http://stalwart:8080

# Production
STALWART_BASE_URL=https://mail.internal.example.com
```

### STALWART_PUBLIC_URL

**Description:** Public URL for Stalwart Mail Server (user access)

**Default:** None

**Required:** No

**Development Value:** `http://localhost:8081`

**Production Requirement:** Public Stalwart URL

**Security Notes:**
- Used for JMAP session URLs
- Should be publicly accessible URL
- Should use HTTPS in production

**Example:**
```bash
# Development
STALWART_PUBLIC_URL=http://localhost:8081

# Production
STALWART_PUBLIC_URL=https://mail.example.com
```

### STALWART_ADMIN_USER

**Description:** Stalwart admin username for management operations

**Default:** None

**Required:** Yes

**Development Value:** `admin`

**Production Requirement:** Production admin username

**Security Notes:**
- Used for server-side provisioning only
- Never expose to frontend or users
- Rotate credentials regularly
- Use strong, unique username

**Example:**
```bash
STALWART_ADMIN_USER=admin
```

### STALWART_ADMIN_PASSWORD

**Description:** Stalwart admin password for management operations

**Default:** None

**Required:** Yes

**Development Value:** `change-me-development-only`

**Production Requirement:** Strong, unique password

**Security Notes:**
- CRITICAL: Never use development defaults in production
- Used for server-side provisioning only
- Never expose to frontend or users
- Rotate credentials regularly
- Use secrets management in production
- Minimum 32 characters recommended

**Example:**
```bash
# Development
STALWART_ADMIN_PASSWORD=change-me-development-only

# Production (use secrets manager)
STALWART_ADMIN_PASSWORD={{from_secrets_manager}}
```

### STALWART_RECOVERY_ADMIN

**Description:** Stalwart recovery admin credentials (used by Stalwart container)

**Default:** None

**Required:** No

**Development Value:** `admin:change-me-development-only`

**Production Requirement:** Production recovery admin credentials

**Security Notes:**
- Used directly by Stalwart container
- Format: `username:password`
- CRITICAL: Never use development defaults in production
- Used for recovery operations only

**Example:**
```bash
# Development
STALWART_RECOVERY_ADMIN=admin:change-me-development-only

# Production
STALWART_RECOVERY_ADMIN=recovery:{{strong_password}}
```

## Authentication Configuration

### JWT_SECRET

**Description:** Secret key for JWT token signing

**Default:** None

**Required:** Yes

**Development Value:** `development-only-jwt-secret-do-not-use-in-production`

**Production Requirement:** Strong, random secret (minimum 32 characters)

**Security Notes:**
- CRITICAL: Never use development defaults in production
- Used to sign all authentication tokens
- Compromise allows token forgery
- Use cryptographically random secret
- Rotate periodically
- Use secrets management in production
- Generate with: `openssl rand -base64 32`

**Example:**
```bash
# Development
JWT_SECRET=development-only-jwt-secret-do-not-use-in-production

# Production (generate secure secret)
JWT_SECRET={{from_secrets_manager}}
```

## CORS Configuration

### ALLOWED_ORIGINS

**Description:** Comma-separated list of allowed CORS origins

**Default:** Empty string (wildcard in development)

**Required:** No

**Development Value:** Empty (allows wildcard)

**Production Requirement:** Specific origins required

**Security Notes:**
- Development: Empty allows wildcard CORS
- Production: Must specify exact origins
- Wildcard (*) rejected in production
- Include protocol and port
- Used for cross-origin API access

**Example:**
```bash
# Development (wildcard)
ALLOWED_ORIGINS=

# Production (specific origins)
ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com

# Multiple environments
ALLOWED_ORIGINS=https://app.example.com,https://staging.example.com
```

## Worker Configuration

### PROVISIONING_WORKERS

**Description:** Number of provisioning worker instances

**Default:** `4`

**Required:** No

**Development Value:** `4`

**Production Requirement:** Based on load (typically 4-16)

**Security Notes:** Higher values increase resource usage

**Example:**
```bash
PROVISIONING_WORKERS=8
```

### WORKER_ID

**Description:** Unique identifier for worker instance

**Default:** Empty string

**Required:** No

**Development Value:** `worker-1`, `worker-2`, `worker-3` (via docker-compose)

**Production Requirement:** Unique ID per worker instance

**Security Notes:** Used for logging and coordination

**Example:**
```bash
WORKER_ID=worker-1
```

### JOB_LEASE_SECONDS

**Description:** Job lease duration in seconds

**Default:** `60`

**Required:** No

**Development Value:** `60`

**Production Requirement:** Typically 60-120 seconds

**Security Notes:** Higher values reduce coordination overhead but increase failure detection time

**Example:**
```bash
JOB_LEASE_SECONDS=90
```

### JOB_HEARTBEAT_SECONDS

**Description:** Job heartbeat interval in seconds

**Default:** `20`

**Required:** No

**Development Value:** `20`

**Production Requirement:** Typically 15-30 seconds

**Security Notes:** Should be less than JOB_LEASE_SECONDS

**Example:**
```bash
JOB_HEARTBEAT_SECONDS=15
```

### JOB_MAX_ATTEMPTS

**Description:** Maximum number of job retry attempts

**Default:** `10`

**Required:** No

**Development Value:** `10`

**Production Requirement:** Typically 5-15

**Security Notes:** Higher values increase retry duration but improve success rate

**Example:**
```bash
JOB_MAX_ATTEMPTS=8
```

### JOB_MAX_BACKOFF_SECONDS

**Description:** Maximum backoff time between retries in seconds

**Default:** `300` (5 minutes)

**Required:** No

**Development Value:** `300`

**Production Requirement:** Typically 300-600 seconds

**Security Notes:** Exponential backoff capped at this value

**Example:**
```bash
JOB_MAX_BACKOFF_SECONDS=600
```

### MAX_CONCURRENT_JOBS

**Description:** Maximum concurrent jobs per worker

**Default:** `5`

**Required:** No

**Development Value:** `5`

**Production Requirement:** Based on worker capacity (typically 5-20)

**Security Notes:** Higher values increase resource usage

**Example:**
```bash
MAX_CONCURRENT_JOBS=10
```

## Test Configuration

### TEST_ACCOUNT_PASSWORD

**Description:** Password for end-to-end test accounts

**Default:** None

**Required:** No

**Development Value:** `change-me-development-only`

**Production Requirement:** Not used in production

**Security Notes:**
- Only used for testing
- Never use in production
- Can be weak for test environments

**Example:**
```bash
TEST_ACCOUNT_PASSWORD=change-me-development-only
```

## Security Validation

### Production Safety Checks

The application performs the following validations in production mode:

1. **Rejects Development Defaults:**
   - `STALWART_ADMIN_PASSWORD` cannot start with `change-me-development-only`
   - `JWT_SECRET` cannot start with `change-me-development-only`
   - `DATABASE_URL` cannot start with `postgres://norest:norest@`

2. **Requires Specific CORS Origins:**
   - `ALLOWED_ORIGINS` must be set
   - Wildcard (*) is rejected

3. **Rejects Development Bootstrap:**
   - `STALWART_RECOVERY_ADMIN` cannot contain `change-me-development-only`

### Security Best Practices

1. **Use Secrets Management:**
   - AWS Secrets Manager
   - HashiCorp Vault
   - Google Secret Manager
   - Azure Key Vault

2. **Generate Strong Secrets:**
   ```bash
   # Generate 32-byte random secret
   openssl rand -base64 32
   
   # Generate password
   openssl rand -base64 24
   ```

3. **Rotate Credentials Regularly:**
   - JWT secrets: Monthly
   - Database passwords: Quarterly
   - Stalwart admin credentials: Quarterly

4. **Use Different Secrets per Environment:**
   - Development secrets
   - Staging secrets
   - Production secrets

5. **Never Commit Secrets:**
   - Add `.env` to `.gitignore`
   - Use environment-specific configuration
   - Audit repository for accidental commits

## Environment-Specific Examples

### Development (.env)

```bash
APP_ENV=development
HTTP_ADDR=:8080

JWT_SECRET=development-only-jwt-secret-do-not-use-in-production

DATABASE_URL=postgres://norest:norest@postgres:5432/norest?sslmode=disable

STALWART_BASE_URL=http://stalwart:8080
STALWART_PUBLIC_URL=http://localhost:8081
STALWART_ADMIN_USER=admin
STALWART_ADMIN_PASSWORD=change-me-development-only
STALWART_RECOVERY_ADMIN=admin:change-me-development-only

ALLOWED_ORIGINS=

PROVISIONING_WORKERS=4
WORKER_ID=worker-1
JOB_LEASE_SECONDS=60
JOB_HEARTBEAT_SECONDS=20
JOB_MAX_ATTEMPTS=10
JOB_MAX_BACKOFF_SECONDS=300
MAX_CONCURRENT_JOBS=5

DB_MAX_CONNS=10
DB_MIN_CONNS=2
DB_MAX_CONN_LIFETIME=1800
DB_MAX_CONN_IDLE_TIME=300
DB_OPERATION_TIMEOUT=30

TEST_ACCOUNT_PASSWORD=change-me-development-only
```

### Production

```bash
APP_ENV=production
HTTP_ADDR=:8080

JWT_SECRET={{from_secrets_manager}}

DATABASE_URL=postgres://{{user}}:{{password}}@{{host}}:5432/norest?sslmode=require

STALWART_BASE_URL=https://mail.internal.example.com
STALWART_PUBLIC_URL=https://mail.example.com
STALWART_ADMIN_USER={{admin_user}}
STALWART_ADMIN_PASSWORD={{from_secrets_manager}}
STALWART_RECOVERY_ADMIN={{recovery_user}}:{{from_secrets_manager}}

ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com

PROVISIONING_WORKERS=8
JOB_LEASE_SECONDS=90
JOB_HEARTBEAT_SECONDS=15
JOB_MAX_ATTEMPTS=8
JOB_MAX_BACKOFF_SECONDS=600
MAX_CONCURRENT_JOBS=10

DB_MAX_CONNS=50
DB_MIN_CONNS=5
DB_MAX_CONN_LIFETIME=3600
DB_MAX_CONN_IDLE_TIME=600
DB_OPERATION_TIMEOUT=60
```

## Configuration Loading

Configuration is loaded at application startup from environment variables. The application:

1. Reads all environment variables
2. Applies defaults where optional
3. Validates required variables
4. Performs production safety checks
5. Fails fast if configuration is invalid

## Troubleshooting

### Missing Required Variables

If you see "required environment variable X is not set":

1. Check if the variable is in your `.env` file
2. Ensure the `.env` file is being loaded
3. Verify variable name is correct (case-sensitive)
4. Check for typos in variable name

### Production Validation Failures

If you see production validation errors:

1. Check you're not using development defaults
2. Ensure `ALLOWED_ORIGINS` is set
3. Verify secrets are strong enough
4. Check for wildcard CORS in production

### Connection Issues

If you see database or Stalwart connection errors:

1. Verify connection strings are correct
2. Check if services are accessible
3. Ensure SSL mode is correct
4. Verify credentials are valid
5. Check network connectivity

## Additional Resources

- [STARTUP.md](STARTUP.md) - Service startup guide
- [API_REFERENCE.md](API_REFERENCE.md) - API documentation
- [AUTHENTICATION.md](AUTHENTICATION.md) - Authentication details