# Norest Mail — Chapter 1

## Real Foundation: Norest Control Plane + Stalwart Mail Engine

You are the implementation engineer for Norest Mail.

Treat this document as the authoritative scope for **Chapter 1**.

Do not continue into later chapters.
Do not invent additional services.
Do not introduce Kafka, Redis, RabbitMQ, Kubernetes, Temporal, Elasticsearch, microservices, service mesh, custom SMTP, custom IMAP, custom JMAP, or a Norest message database.

The purpose of Chapter 1 is to produce a **small, real, runnable foundation** that we can start on a developer machine with one command and prove that Norest and Stalwart can work together.

---

# 1. The architectural rule

Norest Mail has two distinct responsibilities.

## Norest owns the product/control plane

Norest owns:

* Norest users
* Norest authentication
* product accounts
* domains and domain ownership
* email-address ownership/reservation
* subscriptions/plans
* provisioning state
* product policies
* future billing
* future admin operations

## Stalwart owns the mail/data plane

Stalwart owns:

* mailboxes
* folders
* messages
* message bodies
* MIME
* attachments
* threads
* message flags
* search
* JMAP
* IMAP
* POP3
* SMTP
* mail delivery
* mail queue
* actual mail storage

Do not duplicate Stalwart mail data inside PostgreSQL.

Norest PostgreSQL must never become a second mail database.

---

# 2. Chapter 1 goal

At the end of this chapter, the following must work:

```text
docker compose up
        ↓
PostgreSQL starts
        ↓
Stalwart starts
        ↓
Norest API starts
        ↓
Norest worker starts
```

The project must then expose:

```text
GET /health
GET /health/db
GET /health/stalwart
```

The Norest API must successfully communicate with Stalwart.

The code must contain a small Stalwart client abstraction.

The repository must contain migrations and the initial Norest schema.

The system must be runnable from a clean checkout.

---

# 3. Required technology stack

Use:

```text
Go
net/http
chi
pgx
sqlc
PostgreSQL
Docker
Docker Compose
Stalwart
OpenAPI
```

Do not use an ORM.

Do not add a framework larger than necessary.

Go application should be a normal production-style Go project.

Use Go modules.

---

# 4. Required folder structure

Create exactly this structure, with small adjustments only when technically necessary:

```text
norest-mail/
│
├── cmd/
│   ├── api/
│   │   └── main.go
│   │
│   └── worker/
│       └── main.go
│
├── internal/
│   │
│   ├── auth/
│   │   ├── service.go
│   │   ├── handler.go
│   │   ├── repository.go
│   │   └── models.go
│   │
│   ├── users/
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── models.go
│   │
│   ├── domains/
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── models.go
│   │
│   ├── addresses/
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── models.go
│   │
│   ├── provisioning/
│   │   ├── service.go
│   │   ├── worker.go
│   │   └── repository.go
│   │
│   ├── stalwart/
│   │   ├── client.go
│   │   ├── session.go
│   │   ├── management.go
│   │   ├── mailbox.go
│   │   └── email.go
│   │
│   ├── db/
│   │   ├── postgres.go
│   │   └── queries/
│   │
│   ├── http/
│   │   ├── router.go
│   │   ├── middleware.go
│   │   ├── errors.go
│   │   └── response.go
│   │
│   └── config/
│       └── config.go
│
├── migrations/
│   ├── 001_users.sql
│   ├── 002_domains.sql
│   ├── 003_addresses.sql
│   ├── 004_mailboxes.sql
│   └── 005_provisioning_jobs.sql
│
├── api/
│   └── openapi.yaml
│
├── scripts/
│   ├── dev-up.sh
│   ├── dev-down.sh
│   ├── dev-reset.sh
│   ├── wait-for-services.sh
│   └── test-foundation.sh
│
├── docker/
│   └── norest/
│       └── Dockerfile
│
├── docker-compose.yml
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

Do not split these into microservices.

The `api` and `worker` are two binaries from the same Go repository.

---

# 5. Docker architecture

Create one Docker Compose file.

Services:

```text
postgres
stalwart
norest-api
norest-worker
```

Network:

```text
norest
```

All services communicate through the Docker network.

Do not expose PostgreSQL publicly.

Development host ports should be:

```text
Norest API:
8080

Stalwart admin/JMAP HTTP:
8081 on host -> 8080 inside container

PostgreSQL:
5432
```

Mail protocol ports may be exposed for local testing, but they are not required by Chapter 1.

If you expose them, use the standard Stalwart ports:

```text
25
465
587
110
143
993
995
4190
```

Do not make Norest proxy these ports.

---

# 6. Stalwart Docker image

Use the official image:

```text
stalwartlabs/stalwart:v0.16
```

Do not use `latest`.

The current Stalwart documentation explicitly recommends pinning a minor version instead of relying on a moving `latest` tag.

Use named volumes:

```text
stalwart-etc
stalwart-data
```

Mount:

```text
/etc/stalwart
/var/lib/stalwart
```

Stalwart runs as an unprivileged user in the official image, and named volumes are preferred for the initial Docker deployment.

---

# 7. Stalwart development bootstrap

For development, configure a stable recovery administrator through an environment variable.

Use:

```text
STALWART_RECOVERY_ADMIN=admin:dev-admin-password
```

Do not hardcode the production credential.

Put it in `.env.example` as:

```text
STALWART_RECOVERY_ADMIN=admin:change-me-development-only
```

The actual `.env` file must be gitignored.

The goal is to make first startup deterministic.

Stalwart's current Docker documentation supports `STALWART_RECOVERY_ADMIN` specifically to avoid depending on a random bootstrap password.

For Chapter 1, it is acceptable for the development server to initially use HTTP on localhost.

Do not claim that this is production-ready TLS configuration.

---

# 8. Stalwart startup behavior

The first run must be understandable.

README must state:

```text
Stalwart admin:
http://localhost:8081/admin
```

Development credential:

```text
username: admin
password: value from STALWART_RECOVERY_ADMIN
```

The Stalwart setup process may still need to be completed through the admin UI on first initialization.

Do not build a fake Stalwart configuration system to replace the real server.

Do not invent undocumented configuration keys.

If non-interactive configuration is required for automation, inspect the installed/current Stalwart schema or official CLI before implementing it. Stalwart's current management model is JMAP-based and the CLI is schema-driven.

---

# 9. Environment configuration

Create:

```text
.env.example
```

with approximately:

```env
APP_ENV=development
HTTP_ADDR=:8080

DATABASE_URL=postgres://norest:norest@postgres:5432/norest?sslmode=disable

STALWART_BASE_URL=http://stalwart:8080
STALWART_PUBLIC_URL=http://localhost:8081
STALWART_ADMIN_USER=admin
STALWART_ADMIN_PASSWORD=change-me-development-only
STALWART_RECOVERY_ADMIN=admin:change-me-development-only
```

Do not duplicate credentials throughout the repository.

All configuration must be loaded from environment variables.

---

# 10. PostgreSQL

Use the official PostgreSQL image.

Pick a current stable PostgreSQL major supported by the selected Go/Postgres tooling.

Initial database:

```text
database: norest
user: norest
password: norest
```

These values are development-only.

Add a PostgreSQL healthcheck.

Do not make PostgreSQL store Stalwart messages.

---

# 11. Initial Norest schema

Create only these tables.

## users

```sql
id
email
password_hash
status
created_at
updated_at
```

Use UUIDs.

Use a unique constraint on normalized email.

## domains

```sql
id
user_id
name
status
verification_status
created_at
updated_at
```

Domain names must be normalized to lowercase.

Unique domain name.

## addresses

```sql
id
domain_id
local_part
status
created_at
updated_at
```

Store normalized local-part according to the product rule you establish.

Composite uniqueness:

```text
domain_id + normalized_local_part
```

Do not create message tables.

## mailboxes

```sql
id
address_id
stalwart_account_id
status
created_at
updated_at
```

`stalwart_account_id` is a reference/identifier, not a foreign key to PostgreSQL.

The Norest mailbox row means:

```text
"This product address is associated with a Stalwart account."
```

It does NOT store messages.

## provisioning_jobs

```sql
id
type
resource_id
status
attempts
next_attempt_at
last_error
created_at
updated_at
```

Use this for future transactional provisioning.

---

# 12. Database rules

Use:

```text
PostgreSQL
+
pgx
+
sqlc
```

Do not use GORM.

Do not create an event-sourcing system.

Do not create a separate queue service.

For Chapter 1, PostgreSQL is enough.

---

# 13. Stalwart client

Build this first-class abstraction:

```go
type Client struct {
    BaseURL string
    HTTPClient *http.Client
    Username string
    Password string
}
```

Create methods along these lines:

```go
func (c *Client) GetJMAPWellKnown(ctx context.Context) (...)
func (c *Client) GetJMAPSession(ctx context.Context, tokenOrCredential ...) (...)
func (c *Client) ManagementRequest(ctx context.Context, ...)
func (c *Client) MailboxGet(ctx context.Context, ...)
func (c *Client) EmailQuery(ctx context.Context, ...)
func (c *Client) EmailGet(ctx context.Context, ...)
```

The exact signatures are up to good Go design.

The important thing is that application code must not construct arbitrary Stalwart HTTP requests throughout the codebase.

Only:

```text
internal/stalwart/
```

knows Stalwart-specific details.

---

# 14. JMAP requirements

Understand this correctly.

The normal mail JMAP endpoint is:

```text
/jmap
```

and JMAP discovery is available through:

```text
/.well-known/jmap
```

according to the current Stalwart HTTP/JMAP documentation.

Do not invent REST endpoints such as:

```text
GET /emails
GET /folders
GET /messages
```

inside Stalwart.

Use real JMAP methods.

The first JMAP integration proof must cover:

```text
Mailbox/get
Email/query
Email/get
```

and the code should be structured so later chapters can add:

```text
Mailbox/query
Mailbox/set
Email/set
Thread/query
Thread/get
EmailSubmission/set
Email/queryChanges
Email/changes
Mailbox/changes
```

Do not implement every one of these now.

Only implement what Chapter 1 needs to prove connectivity.

---

# 15. What Chapter 1 must prove about mail

The following must be documented even if some require a second bootstrap step after first Stalwart initialization:

```text
1. We can discover the JMAP endpoint.

2. We can authenticate against Stalwart.

3. We can obtain a JMAP session.

4. We can identify the user's mail account.

5. We can request mailboxes/folders.

6. We can query emails.

7. We can fetch emails.

8. We can later extend the same client to sending mail.
```

Do not build a mail proxy.

Do not put JMAP responses into PostgreSQL.

---

# 16. Health endpoints

Implement:

```http
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

Implement:

```http
GET /health/db
```

which executes a cheap PostgreSQL health query.

Implement:

```http
GET /health/stalwart
```

which verifies that the Stalwart service is reachable.

Do not merely check that DNS resolves.

The health check must make an actual HTTP request.

Preferred first check:

```text
GET /.well-known/jmap
```

or another documented lightweight endpoint.

The handler should report a useful failure.

Example:

```json
{
  "status": "degraded",
  "service": "stalwart",
  "error": "connection refused"
}
```

Do not leak passwords or credentials.

---

# 17. API router

Use `chi`.

Create:

```text
internal/http/router.go
```

Routes:

```text
GET /health
GET /health/db
GET /health/stalwart
```

Create a clean place for:

```text
/v1
```

but do not build the complete product API in Chapter 1.

We are preparing the foundation, not implementing billing/auth/domains fully yet.

---

# 18. Worker

Create:

```text
cmd/worker/main.go
```

It should:

1. start,
2. connect to PostgreSQL,
3. verify connectivity,
4. run a basic worker loop,
5. shut down cleanly on SIGINT/SIGTERM.

For Chapter 1, the worker does not need a sophisticated provisioning engine.

Create the plumbing for:

```text
provisioning_jobs
```

but don't build the complete state machine yet.

A simple polling loop is enough.

Example:

```text
every 5 seconds
    query pending jobs
    log that worker is alive
```

Do not add Redis or RabbitMQ.

---

# 19. Graceful shutdown

Both:

```text
norest-api
norest-worker
```

must handle:

```text
SIGINT
SIGTERM
```

and close:

```text
HTTP server
database connections
worker loops
```

cleanly.

---

# 20. Dockerfile for Norest

Create:

```text
docker/norest/Dockerfile
```

Use a multi-stage build.

Builder:

```text
golang
```

Runtime:

Use a minimal runtime image.

Do not run Norest as root.

The final image should contain only the compiled binary and required CA certificates.

Build two binaries:

```text
norest-api
norest-worker
```

Either:

1. build two images, or
2. use one image with two commands.

Prefer one image with two commands for Chapter 1.

Example conceptual pattern:

```dockerfile
FROM golang:... AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/norest-api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/norest-worker ./cmd/worker

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/norest-api /norest-api
COPY --from=builder /out/norest-worker /norest-worker

USER nonroot:nonroot
```

Use versions that are current and compatible when implementing.

---

# 21. Docker Compose requirements

Compose must provide:

```text
postgres
stalwart
norest-api
norest-worker
```

Use service healthchecks.

Norest API must not start before PostgreSQL is healthy.

Norest worker must not start before PostgreSQL is healthy.

Stalwart health should be checked before declaring the whole development environment ready.

Use a shared Docker network.

Do not use host networking.

---

# 22. Run script

Create:

```text
scripts/dev-up.sh
```

Requirements:

```text
1. Verify Docker is installed.
2. Verify Docker Compose is available.
3. Create .env from .env.example if missing.
4. Start all services.
5. Wait for PostgreSQL.
6. Wait for Stalwart HTTP.
7. Wait for Norest API.
8. Run migrations.
9. Print service URLs.
10. Exit successfully only when Norest health endpoint responds.
```

Output should look approximately like:

```text
Norest Mail development environment

PostgreSQL:
  localhost:5432

Stalwart Admin:
  http://localhost:8081/admin

Stalwart JMAP:
  http://localhost:8081/jmap

Norest API:
  http://localhost:8080

Health:
  http://localhost:8080/health

Starting services...
Waiting for PostgreSQL... OK
Waiting for Stalwart... OK
Waiting for Norest API... OK

Norest Mail is running.
```

Do not fake these checks.

Actually test them.

---

# 23. Run-down script

Create:

```text
scripts/dev-down.sh
```

It should run:

```text
docker compose down
```

Do not remove volumes by default.

---

# 24. Reset script

Create:

```text
scripts/dev-reset.sh
```

This is explicitly destructive.

It must ask for confirmation unless a force flag is supplied.

Example:

```text
./scripts/dev-reset.sh
```

asks:

```text
This deletes development database and Stalwart data.
Type RESET to continue:
```

Then:

```text
docker compose down -v
```

and restart.

Do not make normal `dev-down.sh` destructive.

---

# 25. Foundation test script

Create:

```text
scripts/test-foundation.sh
```

It must test:

```text
1. /health
2. /health/db
3. /health/stalwart
4. Stalwart JMAP discovery
5. Stalwart JMAP endpoint reachability
```

Exit with non-zero status if any test fails.

Use curl or a small Go test utility.

Keep it simple.

---

# 26. Migrations

Do not add a heavyweight migration system unless necessary.

If using a migration library, keep it simple.

The application must automatically apply development migrations at startup OR the dev script must explicitly run migrations.

I prefer:

```text
scripts/dev-up.sh
    ↓
migration command
    ↓
start/verify
```

Production behavior should later become explicit rather than hidden inside application startup.

---

# 27. OpenAPI

Create:

```text
api/openapi.yaml
```

Document only:

```text
GET /health
GET /health/db
GET /health/stalwart
```

Do not invent all future endpoints now.

This document should be valid OpenAPI.

---

# 28. Logging

Use structured logging.

Prefer Go's standard:

```text
log/slog
```

Log:

```text
startup
shutdown
database connection
Stalwart connection
HTTP requests
worker startup
worker shutdown
migration startup/result
```

Never log:

```text
passwords
access tokens
API keys
mail credentials
```

---

# 29. Configuration package

Create:

```text
internal/config/config.go
```

It should load:

```text
APP_ENV
HTTP_ADDR
DATABASE_URL
STALWART_BASE_URL
STALWART_PUBLIC_URL
STALWART_ADMIN_USER
STALWART_ADMIN_PASSWORD
```

Validate required values during startup.

Fail fast if required configuration is missing.

---

# 30. Stalwart integration design

Do not hard-code assumptions about undocumented API responses.

The current Stalwart management API is intentionally small and most configuration/mailbox data is exposed through JMAP. The management API supports authentication via bearer or basic auth, while API keys are intended for management automation and do not authenticate normal mail protocols.

Therefore:

```text
Norest administrative Stalwart client
        ↓
Management JMAP / documented API
```

is separate conceptually from:

```text
User mail access
        ↓
JMAP mail endpoint
```

Do not confuse admin credentials with end-user mail credentials.

---

# 31. Do NOT use the Stalwart admin password as the user's mail password

The Stalwart admin credential is only for development administration/provisioning.

Never expose:

```text
STALWART_ADMIN_PASSWORD
```

to the frontend.

Never return it from a Norest endpoint.

Never store it in Norest user records.

Never use it as a user's mail authentication credential.

---

# 32. First-run manual bootstrap

The repository README must clearly say:

```text
1. Copy .env.example to .env.
2. Run ./scripts/dev-up.sh.
3. Open http://localhost:8081/admin.
4. Complete the initial Stalwart setup if the installation is still in bootstrap mode.
5. Restart Stalwart if the setup requires it.
6. Run ./scripts/test-foundation.sh.
```

Do not hide this behavior.

The first version should be understandable to a human developer.

---

# 33. Important: do not fake mailbox APIs

Do NOT implement:

```text
GET /v1/mailboxes
GET /v1/messages
GET /v1/folders
```

in Chapter 1.

Those belong to the eventual webmail/mail client integration with Stalwart JMAP.

Norest APIs are for Norest product data.

---

# 34. Important: do not create mail tables

This is a hard rule.

The migration directory must NOT contain:

```text
messages
emails
email_bodies
attachments
folders
threads
mail_queue
smtp_queue
imap_state
jmap_state
```

Any implementation that creates these tables is incorrect.

---

# 35. Documentation required

Create:

```text
README.md
docs/architecture.md
```

`README.md` must contain:

```text
What Norest is
What Stalwart is
How to run
How to stop
How to reset
How to access Stalwart admin
How to access Norest API
How to test
```

`docs/architecture.md` must contain:

```text
Norest control plane
Stalwart mail plane
Data ownership rules
API boundary
JMAP boundary
Docker architecture
Database ownership
Future scaling direction
```

---

# 36. Required README architecture diagram

Include:

```text
                  Browser / Client
                        |
             +----------+----------+
             |                     |
             v                     v
       Norest REST API        Stalwart JMAP
             |                     |
             v                     v
         PostgreSQL             Mail Data
```

Also document:

```text
Norest:
identity/product/domain/subscription/provisioning

Stalwart:
mailbox/messages/folders/search/send/receive/storage
```

---

# 37. Required code quality

The implementation must:

* compile cleanly
* pass `go test ./...`
* use context-aware database operations
* use timeouts on outgoing HTTP
* handle errors explicitly
* close response bodies
* avoid global mutable state
* support graceful shutdown
* avoid credential leakage
* use parameterized SQL
* use transactions where appropriate
* keep package responsibilities clear

Do not over-abstract.

Do not create interfaces for every single struct just for the sake of testability.

---

# 38. Testing requirements

At minimum include:

```text
unit test:
configuration loading

unit test:
domain normalization

integration test:
PostgreSQL connectivity

integration test:
Stalwart HTTP/JMAP connectivity

integration test:
Norest health endpoint
```

Tests must be deterministic.

---

# 39. Acceptance criteria

Chapter 1 is COMPLETE only when all of these are true:

```text
[ ] Fresh checkout works.

[ ] cp .env.example .env works.

[ ] ./scripts/dev-up.sh starts the stack.

[ ] PostgreSQL becomes healthy.

[ ] Stalwart becomes reachable.

[ ] Norest API becomes reachable.

[ ] Norest worker starts.

[ ] Database migrations apply successfully.

[ ] GET /health returns success.

[ ] GET /health/db returns success.

[ ] GET /health/stalwart returns success.

[ ] Stalwart admin page is reachable.

[ ] Stalwart JMAP discovery is reachable.

[ ] Stalwart JMAP endpoint is reachable.

[ ] internal/stalwart package exists.

[ ] The code contains JMAP client plumbing.

[ ] No mail/message tables exist in Norest PostgreSQL.

[ ] No custom SMTP implementation exists.

[ ] No custom IMAP implementation exists.

[ ] No custom mail storage exists.

[ ] No Redis/Kafka/RabbitMQ exists.

[ ] go test ./... passes.

[ ] docker compose config is valid.

[ ] test-foundation.sh passes.

[ ] README contains exact startup instructions.

[ ] docs/architecture.md explains ownership boundaries.
```

---

# 40. Definition of success

The final developer experience should be approximately:

```bash
git clone ...
cd norest-mail

cp .env.example .env

./scripts/dev-up.sh
```

Then:

```text
Norest Mail is running.

Norest API:
http://localhost:8080

Stalwart Admin:
http://localhost:8081/admin

Stalwart JMAP:
http://localhost:8081/jmap
```

Then:

```bash
./scripts/test-foundation.sh
```

should report all checks successful.

---

# 41. Do not proceed beyond Chapter 1

Do not implement:

```text
full authentication
domain verification
billing
subscriptions
webmail
mail session API
production OAuth
multi-tenant policies
admin portal
advanced provisioning
cluster deployment
multi-region
observability stack
```

Those belong to later chapters.

Chapter 1 is infrastructure and boundary validation.

The purpose is to prove:

```text
Norest exists.
PostgreSQL exists.
Stalwart exists.
Norest can communicate with Stalwart.
The architecture is clean.
The development environment is reproducible.
The foundation actually runs.
```

---

# 42. Final instruction to the implementation agent

Work directly in the repository.

Do not merely describe what should be done.

Create the files.

Create the code.

Create the Dockerfile.

Create docker-compose.yml.

Create migrations.

Create environment examples.

Create the scripts.

Create the OpenAPI file.

Create the tests.

Run:

```bash
go test ./...
docker compose config
./scripts/dev-up.sh
./scripts/test-foundation.sh
```

Fix every issue you encounter.

Do not stop at "implemented".

At the end, verify the actual system.

Then report:

```text
1. Files created/changed
2. Exact startup command
3. Exact test command
4. Services running
5. Norest health result
6. PostgreSQL health result
7. Stalwart health result
8. JMAP discovery result
9. Any manual Stalwart first-run step still required
10. Known limitations
```

Most importantly:

**Build the smallest real system that proves the Norest + Stalwart architecture.**

Do not overbuild.
Do not create a second mail server.
Do not create a second mail database.
Do not introduce infrastructure that Chapter 1 does not need.
