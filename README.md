# Norest Mail

Norest Mail is a mail service platform built on two distinct systems:

- **Norest** — the product/control plane (users, domains, addresses, subscriptions, provisioning)
- **Stalwart** — the mail/data plane (mailboxes, messages, JMAP, IMAP, SMTP, storage)

Norest does not store mail data. Stalwart does not know about product subscriptions. The boundary is clear and intentional.

## Architecture

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

Norest:
  identity / product / domain / subscription / provisioning

Stalwart:
  mailbox / messages / folders / search / send / receive / storage
```

## Prerequisites

- Docker
- Docker Compose

## Quick Start

```bash
# 1. Copy environment file
cp .env.example .env

# 2. Start the development environment
./scripts/dev-up.sh

# 3. Run foundation tests
./scripts/test-foundation.sh
```

## Services

| Service | URL | Description |
|---------|-----|-------------|
| Norest API | http://localhost:8080 | Product control-plane API |
| Stalwart Admin | http://localhost:8081/admin | Stalwart admin UI |
| Stalwart JMAP | http://localhost:8081/jmap | JMAP mail endpoint |
| PostgreSQL | Internal only | Database (Docker network) |

## Health Endpoints

```bash
# Application health
curl http://localhost:8080/health

# Database health
curl http://localhost:8080/health/db

# Stalwart connectivity
curl http://localhost:8080/health/stalwart
```

## Stalwart Admin

```text
URL:      http://localhost:8081/admin
Username: admin
Password: value from STALWART_RECOVERY_ADMIN in .env
```

The Stalwart setup process may still need to be completed through the admin UI on first initialization.

## First-Run Instructions

1. Copy `.env.example` to `.env`.
2. Run `./scripts/dev-up.sh`.
3. Open http://localhost:8081/admin.
4. Complete the initial Stalwart setup if the installation is still in bootstrap mode.
5. Restart Stalwart if the setup requires it: `docker compose restart stalwart`
6. Run `./scripts/test-foundation.sh`.

## How to Stop

```bash
./scripts/dev-down.sh
```

Volumes are preserved. Your data persists between restarts.

## How to Reset

```bash
./scripts/dev-reset.sh
```

This destroys all development data (database + Stalwart data). You will need to complete Stalwart setup again.

Use `--force` to skip the confirmation prompt:

```bash
./scripts/dev-reset.sh --force
```

## How to Test

```bash
# Foundation tests (services must be running)
./scripts/test-foundation.sh

# Go unit tests
go test ./...
```

## Project Structure

```text
cmd/api/          — Norest API server entry point
cmd/worker/       — Norest provisioning worker entry point
internal/auth/    — Authentication (Chapter 2)
internal/users/   — Norest user management
internal/domains/ — Domain registration and ownership
internal/addresses/ — Email address management
internal/provisioning/ — Async provisioning worker
internal/stalwart/ — Stalwart JMAP client abstraction
internal/db/      — PostgreSQL connection and utilities
internal/http/    — HTTP router, middleware, response helpers
internal/config/  — Configuration loading from environment
migrations/       — PostgreSQL schema migrations
api/              — OpenAPI specification
scripts/          — Development workflow scripts
docker/           — Dockerfiles
```

## Technology Stack

- Go (net/http + chi)
- PostgreSQL 17 (pgx, no ORM)
- Docker + Docker Compose
- Stalwart Mail Server v0.16
- JMAP (RFC 8620 / 8621)
