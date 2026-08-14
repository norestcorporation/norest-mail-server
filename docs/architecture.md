# Norest Mail — Architecture

## Overview

Norest Mail is built on a strict separation between the **product/control plane** (Norest) and the **mail/data plane** (Stalwart).

## The Architectural Rule

### Norest Owns the Control Plane

Norest is responsible for:

- **Norest users** — platform accounts for the product
- **Authentication** — Norest product authentication
- **Product accounts** — subscription and plan management
- **Domains** — domain registration and ownership verification
- **Email-address ownership** — address reservation and allocation
- **Subscriptions/plans** — billing and feature gates
- **Provisioning state** — tracking what has been provisioned in Stalwart
- **Product policies** — rate limits, quotas, restrictions
- **Future billing** — payment processing
- **Future admin operations** — administrative tooling

### Stalwart Owns the Mail Plane

Stalwart is responsible for:

- **Mailboxes** — folder structures
- **Messages** — email storage and retrieval
- **MIME/attachments** — message body parsing
- **Threads** — conversation threading
- **Message flags** — read/unread, flagged, etc.
- **Search** — full-text mail search
- **JMAP** — modern mail protocol
- **IMAP** — legacy mail access
- **POP3** — legacy mail retrieval
- **SMTP** — mail sending and receiving
- **Mail delivery** — MX processing
- **Mail queue** — outbound delivery queue
- **Mail storage** — actual message data

## Data Ownership Rules

1. **Norest PostgreSQL must never become a second mail database.**
2. **Mail data (messages, folders, threads) lives only in Stalwart.**
3. **Product data (users, domains, subscriptions) lives only in Norest PostgreSQL.**
4. **The `mailboxes` table in PostgreSQL is a reference mapping, not message storage.** It records: "This product address is associated with a Stalwart account."

## API Boundary

```text
Norest REST API:
  Product/control operations
  User management
  Domain management
  Address management
  Subscription management
  Provisioning status

Stalwart JMAP:
  Mail operations
  Mailbox listing
  Email reading
  Email sending
  Search
```

Norest does NOT proxy JMAP. Clients that need mail data connect to Stalwart directly.

## JMAP Boundary

Norest interacts with Stalwart through:

1. **Admin JMAP** — for provisioning (creating accounts, domains) using admin credentials
2. **User JMAP** — clients connect directly to Stalwart for mail operations

The admin credential is never exposed to end users or the frontend.

## Docker Architecture

```text
┌──────────────────────────────────────────────┐
│                 Docker Network               │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ postgres │  │ stalwart │  │ norest   │   │
│  │          │  │          │  │ api      │   │
│  │ :5432    │  │ :8080    │  │ :8080    │   │
│  └──────────┘  └──────────┘  └──────────┘   │
│                                              │
│                              ┌──────────┐   │
│                              │ norest   │   │
│                              │ worker   │   │
│                              └──────────┘   │
└──────────────────────────────────────────────┘

Host ports:
  8080 → norest-api:8080
  8081 → stalwart:8080
  (PostgreSQL is not exposed)
```

## Database Ownership

| Table | Owner | Purpose |
|-------|-------|---------|
| `users` | Norest | Platform user accounts |
| `domains` | Norest | Registered mail domains |
| `addresses` | Norest | Reserved email addresses |
| `mailboxes` | Norest | Reference mapping to Stalwart accounts |
| `provisioning_jobs` | Norest | Async provisioning queue |

None of these tables store mail content.

## Future Scaling Direction

- **Norest API** can be horizontally scaled behind a load balancer
- **Norest Worker** can run multiple instances with job locking
- **PostgreSQL** can be replicated or upgraded independently
- **Stalwart** manages its own clustering and storage
- **Each component scales independently** due to the clean separation

The architecture is designed so that Norest and Stalwart can evolve, scale, and be replaced independently.
