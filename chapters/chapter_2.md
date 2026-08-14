# Norest Mail — Chapter 2

## Identity, Domain Ownership, Email Addresses & Stalwart Provisioning

You are continuing Norest Mail after Chapter 1.

Chapter 1 is complete and must be treated as the immutable foundation.

Do not redesign Chapter 1.

Do not add new infrastructure.

Do not turn Norest into a mail server.

The purpose of Chapter 2 is to make Norest a real product for the first time:

```text
Norest user
    ↓
Norest account
    ↓
domain
    ↓
email address
    ↓
mailbox record
    ↓
provisioning job
    ↓
Stalwart account
```

At the end of this chapter, a real Norest user must be able to create/use an email identity that is provisioned into Stalwart.

---

# 1. Absolute architecture rule

Preserve the Chapter 1 ownership boundary.

## Norest owns

```text
user identity
product account
domain ownership
address ownership
reservation
product status
provisioning state
product limits
authentication
```

## Stalwart owns

```text
actual mailbox
messages
folders
attachments
MIME
threads
mail flags
search
JMAP
IMAP
POP3
SMTP
delivery
mail storage
```

Norest PostgreSQL must never store:

```text
messages
emails
email bodies
attachments
folders
threads
IMAP state
JMAP state
SMTP queue
mail delivery state
```

This remains a hard architectural rule from Chapter 1.

---

# 2. What Chapter 2 is responsible for

Chapter 2 contains six capabilities:

```text
1. Norest user registration
2. Norest authentication
3. Domain creation/ownership
4. Email-address creation/ownership
5. Mailbox provisioning
6. Stalwart synchronization of provisioning state
```

Do not implement billing yet.

Do not implement webmail yet.

Do not implement DNS verification automation yet.

Do not implement production OAuth/OIDC yet.

Do not implement multi-region infrastructure.

Do not implement Redis/Kafka/RabbitMQ.

---

# 3. End-state user flow

The primary flow must become:

```text
                    USER
                      |
                      v
              POST /v1/auth/register
                      |
                      v
               Norest PostgreSQL
                      |
                      v
              Norest user/account
                      |
                      v
              POST /v1/domains
                      |
                      v
             Domain ownership
                      |
                      v
         POST /v1/domains/:id/addresses
                      |
                      v
             Address reservation
                      |
                      v
           provisioning_job
                      |
                      v
               Norest Worker
                      |
                      v
              Stalwart JMAP
                      |
                      v
               Stalwart Account
                      |
                      v
             mailbox = ACTIVE
```

The important point:

**The user's product address is created by Norest first.**

**The actual mail account is then provisioned into Stalwart.**

---

# 4. User model

Chapter 2 turns the Chapter 1 `users` table into a real identity model.

The user must have:

```text
id
email
password_hash
status
created_at
updated_at
```

Use UUID.

Normalize the email address before uniqueness checks.

At minimum:

```text
email = lowercase(trim(input))
```

Do not silently invent case-folding rules beyond the product's documented policy.

---

# 5. User statuses

Implement a small explicit state machine.

Use:

```text
PENDING
ACTIVE
SUSPENDED
DISABLED
```

Allowed transitions:

```text
PENDING → ACTIVE
PENDING → DISABLED

ACTIVE → SUSPENDED
ACTIVE → DISABLED

SUSPENDED → ACTIVE
SUSPENDED → DISABLED
```

Do not permit arbitrary state mutation.

All status changes must happen through service logic.

Repository code must not contain business-state transitions.

---

# 6. Registration API

Implement:

```http
POST /v1/auth/register
```

Request:

```json
{
  "email": "alice@example.com",
  "password": "a strong password"
}
```

Response:

```json
{
  "id": "uuid",
  "email": "alice@example.com",
  "status": "ACTIVE"
}
```

For Chapter 2, email verification is not required before activation.

However:

```text
email_verified_at
```

may be added now if it simplifies the future model.

Do not build an email-verification delivery system in this chapter.

---

# 7. Password handling

Never store raw passwords.

Use:

```text
Argon2id
```

The password hash belongs in PostgreSQL.

Do not use:

```text
MD5
SHA1
plain SHA256
bcrypt with weak parameters
plaintext
```

Use a secure Argon2id configuration with explicitly documented parameters.

Passwords must never appear in logs.

Passwords must never be returned by APIs.

---

# 8. Login

Implement:

```http
POST /v1/auth/login
```

Request:

```json
{
  "email": "alice@example.com",
  "password": "..."
}
```

Response should establish authenticated Norest access.

For Chapter 2, use a straightforward token model:

```text
short-lived access token
+
refresh token
```

Do not build OAuth/OIDC yet.

Do not use Stalwart admin credentials for user authentication.

Norest authentication and Stalwart mailbox credentials are separate concerns.

---

# 9. Authentication implementation

Create a clean authentication service.

Recommended structure:

```text
internal/auth/
    service.go
    handler.go
    repository.go
    models.go
    password.go
    token.go
```

Add:

```go
AuthenticateUser(...)
CreateAccessToken(...)
CreateRefreshToken(...)
ValidateAccessToken(...)
HashPassword(...)
VerifyPassword(...)
```

Keep cryptographic details inside `auth`.

Do not spread password/token handling throughout the application.

---

# 10. Authentication middleware

Protected API routes must require authentication.

Create:

```text
Authorization: Bearer <token>
```

middleware.

The middleware should:

1. validate the access token
2. identify the Norest user
3. attach user ID to request context
4. reject invalid/expired tokens

Never trust a user ID supplied directly in request JSON when the authenticated identity is already known.

---

# 11. Current-user endpoint

Implement:

```http
GET /v1/me
```

Response:

```json
{
  "id": "uuid",
  "email": "alice@example.com",
  "status": "ACTIVE"
}
```

This establishes the frontend's basic authenticated identity.

---

# 12. Domain ownership

A Norest user can create a domain.

Endpoint:

```http
POST /v1/domains
```

Request:

```json
{
  "name": "example.com"
}
```

The service must:

1. normalize the domain
2. validate it
3. check uniqueness
4. associate it with the authenticated Norest user
5. create the Norest domain record
6. create appropriate provisioning state for Stalwart

---

# 13. Domain normalization

At minimum:

```text
trim
lowercase
remove trailing dot if the product policy requires it
```

Do not normalize by guessing.

Create a dedicated function:

```go
NormalizeDomain(string) (string, error)
```

Test it thoroughly.

Examples:

```text
Example.COM
example.com
 example.com
example.com.
```

must follow one deterministic documented policy.

---

# 14. Domain validation

Reject:

```text
empty domain
invalid hostname syntax
obviously malformed labels
labels exceeding DNS limits
spaces
control characters
```

Do not implement full DNS verification yet.

Chapter 2 establishes ownership/state.

DNS proof belongs to the next product stage unless needed for a minimal bootstrap test.

---

# 15. Domain states

Use:

```text
PENDING
VERIFYING
ACTIVE
SUSPENDED
DISABLED
```

Transitions:

```text
PENDING → VERIFYING
VERIFYING → ACTIVE
VERIFYING → PENDING

ACTIVE → SUSPENDED
ACTIVE → DISABLED

SUSPENDED → ACTIVE
SUSPENDED → DISABLED
```

Do not permit:

```text
DISABLED → ACTIVE
```

without an explicit reactivation path.

Keep the state machine inside `domains/service.go`.

---

# 16. Domain APIs

Implement:

```http
POST   /v1/domains
GET    /v1/domains
GET    /v1/domains/:domainID
PATCH  /v1/domains/:domainID
DELETE /v1/domains/:domainID
```

Do not allow users to access domains they do not own.

Every lookup must include authenticated-user ownership checks.

Do not make:

```http
GET /v1/domains/:id
```

a global unrestricted resource lookup.

---

# 17. Domain response

A normal domain response should contain:

```json
{
  "id": "uuid",
  "name": "example.com",
  "status": "ACTIVE",
  "verification_status": "PENDING",
  "created_at": "...",
  "updated_at": "..."
}
```

Do not return internal Stalwart admin credentials.

It is acceptable to expose a non-secret provisioning status.

---

# 18. Stalwart domain provisioning

When the Norest domain becomes eligible for mail provisioning, provision the corresponding Stalwart domain.

Current Stalwart management exposes the `x:Domain` object over the `urn:stalwart:jmap` capability and supports `x:Domain/set`.

The Norest Stalwart adapter must encapsulate this.

Example conceptual method:

```go
CreateDomain(ctx context.Context, name string) (string, error)
```

The Stalwart ID must be stored in Norest:

```text
domains
    stalwart_domain_id
```

Therefore, add:

```text
stalwart_domain_id TEXT NULL
```

to the Norest domains table.

This is product-to-mail-system linkage only.

Do not store Stalwart's entire domain object in PostgreSQL.

---

# 19. Important provisioning rule

Never provision synchronously inside the HTTP request.

Bad:

```text
POST /domains
    ↓
Norest
    ↓
wait 4 seconds
    ↓
Stalwart
    ↓
respond
```

Instead:

```text
POST /domains
    ↓
PostgreSQL transaction
    ├── domain
    └── provisioning_job
    ↓
HTTP 201
    ↓
worker
    ↓
Stalwart
```

This makes the API fast and resilient.

---

# 20. Provisioning jobs

Expand the Chapter 1 job model.

Job types:

```text
DOMAIN_CREATE
DOMAIN_DELETE
ACCOUNT_CREATE
ACCOUNT_DISABLE
ACCOUNT_ENABLE
ACCOUNT_DELETE
```

Do not add more unless required.

Statuses:

```text
PENDING
PROCESSING
SUCCEEDED
FAILED
```

Recommended additional state:

```text
RETRY_WAIT
```

if useful for retry scheduling.

---

# 21. Job ownership

Each job must contain enough information to identify the resource.

For example:

```text
type = DOMAIN_CREATE
resource_id = domain UUID
```

or:

```text
type = ACCOUNT_CREATE
resource_id = mailbox UUID
```

The worker loads the authoritative Norest object from PostgreSQL.

It does not trust arbitrary payload data from the job.

---

# 22. Transactional creation

Domain creation must happen atomically in Norest.

Conceptually:

```sql
BEGIN;

INSERT INTO domains (...);

INSERT INTO provisioning_jobs (...);

COMMIT;
```

Never allow:

```text
domain created
job missing
```

or:

```text
job created
domain missing
```

---

# 23. Email address model

An email address is a Norest product resource.

Endpoint:

```http
POST /v1/domains/:domainID/addresses
```

Request:

```json
{
  "local_part": "alice"
}
```

Norest computes:

```text
alice@example.com
```

Do not allow the client to independently submit:

```json
{
  "domain": "another-user-domain.com",
  "email": "victim@example.com"
}
```

The server determines the final address from the owned domain plus local part.

---

# 24. Address normalization

Create:

```go
NormalizeLocalPart(string) (string, error)
```

For Chapter 2, implement a clear documented product policy.

Recommended initial policy:

```text
lowercase
trim surrounding whitespace
reject whitespace
reject control characters
reject invalid separators
```

Do not implement complicated SMTP internationalization rules yet.

Design the model so the policy can expand later.

---

# 25. Reserved local parts

Create a reserved-list mechanism.

At minimum reserve:

```text
admin
administrator
postmaster
hostmaster
abuse
security
support
root
noreply
no-reply
```

The list must live in one place.

Do not scatter:

```go
if local == "admin"
```

through handlers.

Create something like:

```go
IsReservedLocalPart(local string) bool
```

Make the reserved list configurable later, but a static Chapter 2 list is acceptable.

---

# 26. Address uniqueness

The database must enforce uniqueness.

Use:

```text
(domain_id, normalized_local_part)
```

unique constraint.

Never rely only on:

```text
SELECT
then INSERT
```

because concurrent requests can race.

The PostgreSQL constraint is authoritative.

---

# 27. Address status

Use:

```text
PENDING
ACTIVE
SUSPENDED
DISABLED
```

The address becomes:

```text
PENDING
```

during provisioning.

It becomes:

```text
ACTIVE
```

only after the corresponding Stalwart account exists successfully.

---

# 28. Mailbox relationship

An address should have one Norest mailbox record.

Conceptually:

```text
domains
   |
   +--- addresses
           |
           +--- mailboxes
```

The mailbox contains:

```text
id
address_id
stalwart_account_id
status
created_at
updated_at
```

Do not put message information here.

---

# 29. Stalwart account provisioning

The current Stalwart management model exposes an `x:Account` object for user/group accounts used for authentication and email access, and supports `x:Account/set`.

Use your existing `internal/stalwart/management.go`.

Add a method conceptually equivalent to:

```go
CreateAccount(ctx context.Context, ...)
```

Inputs should be:

```text
local part
domain identifier
```

The Stalwart account should represent:

```text
alice@example.com
```

Do not invent a parallel mail-user database.

---

# 30. Stalwart account password

The actual mailbox needs credentials so a future mail session can authenticate.

For Chapter 2:

* create a secure random mailbox password
* never return it to the frontend
* never log it
* store only what is necessary to support the next authentication design
* keep the design ready for a future secure credential/session mechanism

Do not expose raw Stalwart mailbox passwords through the Norest API.

Do not reuse the Norest user's password automatically unless explicitly designed and justified.

Prefer separating:

```text
Norest identity credential
```

from:

```text
mail-engine credential
```

---

# 31. Very important credential boundary

There are now three different credential concepts:

```text
1. Norest user authentication
2. Norest → Stalwart administrative provisioning credential
3. Mailbox/user → Stalwart mail authentication credential
```

They are not interchangeable.

Never do:

```text
Norest admin password
        =
Alice mailbox password
```

Never send the admin credential to the browser.

Never return the mailbox provisioning credential from an admin API accidentally.

---

# 32. Mailbox creation workflow

The real workflow should be:

```text
POST /v1/domains/{domainID}/addresses
                |
                v
         Validate ownership
                |
                v
          Begin PostgreSQL
                |
        +-------+--------+
        |                |
        v                v
     address          mailbox
        |                |
        +-------+--------+
                |
                v
       provisioning_job
                |
                v
             COMMIT
                |
                v
          return 201
                |
                v
             worker
                |
                v
        x:Account/set
                |
                v
     store stalwart_account_id
                |
                v
      mailbox = ACTIVE
      address = ACTIVE
```

This is the most important Chapter 2 flow.

---

# 33. Idempotency

Provisioning must be safe to retry.

Example:

```text
Norest creates job
worker calls Stalwart
network timeout
worker does not know if Stalwart succeeded
```

Retrying must not create:

```text
Alice account 1
Alice account 2
Alice account 3
```

Design the Stalwart provisioning operation around a deterministic identity and/or pre-existence lookup.

Before creating:

```text
x:Account/get
```

or query the existing account when appropriate.

Only create when it does not exist.

The same rule applies to domains.

---

# 34. Job retries

Implement exponential retry.

Example:

```text
attempt 1 → 2s
attempt 2 → 5s
attempt 3 → 15s
attempt 4 → 60s
attempt 5 → 5m
```

Cap retries.

Record:

```text
attempts
last_error
next_attempt_at
```

Do not retry permanent validation errors forever.

Classify errors:

```text
temporary
permanent
already_exists
not_found
unauthorized
```

---

# 35. Worker concurrency

Use a small configurable worker pool.

Environment:

```text
PROVISIONING_WORKERS=4
```

Default:

```text
4
```

The worker must process different jobs concurrently while protecting the same job from double-processing.

Use PostgreSQL row locking:

```sql
FOR UPDATE SKIP LOCKED
```

or an equivalent safe job-claim pattern.

Do not add Redis locks.

Do not add distributed lock infrastructure.

PostgreSQL is enough at this stage.

---

# 36. Job claim

A worker should atomically claim a job.

Conceptually:

```sql
UPDATE provisioning_jobs
SET status = 'PROCESSING',
    attempts = attempts + 1
WHERE id = (...)
  AND status IN ('PENDING', 'RETRY_WAIT')
RETURNING ...;
```

Or use a transaction with:

```text
FOR UPDATE SKIP LOCKED
```

so multiple worker instances can safely process jobs later.

This is important for million-user scalability.

---

# 37. Provisioning consistency

After successful domain provisioning:

```text
domains.stalwart_domain_id != NULL
```

After successful mailbox provisioning:

```text
mailboxes.stalwart_account_id != NULL
```

Only then should the corresponding Norest resource become `ACTIVE`.

---

# 38. Recovery / reconciliation

Implement a simple reconciliation service.

It should be possible to detect:

```text
Norest says PENDING
Stalwart already has object
```

or:

```text
Norest says ACTIVE
Stalwart object missing
```

Do not build a complicated distributed sync system.

Create a simple command/service capable of checking a resource.

Example internal methods:

```go
VerifyDomainProvisioned(...)
VerifyMailboxProvisioned(...)
```

The full automated reconciliation loop can come later.

---

# 39. API endpoints for addresses

Implement:

```http
POST   /v1/domains/:domainID/addresses
GET    /v1/domains/:domainID/addresses
GET    /v1/addresses/:addressID
PATCH  /v1/addresses/:addressID
DELETE /v1/addresses/:addressID
```

All protected.

All ownership checked.

---

# 40. Address response

Example:

```json
{
  "id": "uuid",
  "address": "alice@example.com",
  "local_part": "alice",
  "domain_id": "uuid",
  "status": "ACTIVE",
  "mailbox": {
    "status": "ACTIVE"
  },
  "created_at": "...",
  "updated_at": "..."
}
```

Do not expose the Stalwart password.

It is acceptable to expose a stable non-secret Stalwart identifier only to internal/admin APIs, not necessarily public user APIs.

---

# 41. Domain deletion rules

Do not permit this:

```text
DELETE domain
while active mailbox addresses exist
```

Return a conflict:

```http
409 Conflict
```

The user must first remove/disable all addresses.

Do not physically destroy everything immediately.

Design the product lifecycle clearly.

---

# 42. Address deletion rules

For Chapter 2:

```text
ACTIVE
  ↓
DISABLED
```

should be safer than immediately destroying the Stalwart account.

Therefore the initial DELETE operation may mean:

```text
Norest address disabled
```

and create:

```text
ACCOUNT_DISABLE
```

Do not permanently destroy Stalwart data unless a later explicit deletion workflow exists.

This protects against accidental destructive operations.

---

# 43. Domain ownership enforcement

Every domain operation must enforce:

```sql
domain.user_id = authenticated_user_id
```

Every address operation must enforce:

```text
address.domain.user_id = authenticated_user_id
```

Do not trust:

```text
user_id
```

from request bodies.

The authenticated token establishes the user.

---

# 44. API status codes

Use predictable HTTP behavior.

```text
201 Created
200 OK
204 No Content
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
422 Unprocessable Entity
500 Internal Server Error
503 Service Unavailable
```

Examples:

Duplicate address:

```text
409
```

User attempting another user's domain:

```text
404
```

or `403` according to the API privacy policy.

Invalid email:

```text
422
```

---

# 45. Error format

Standardize:

```json
{
  "error": {
    "code": "ADDRESS_ALREADY_EXISTS",
    "message": "The email address already exists."
  }
}
```

Do not expose:

```text
SQL statements
Stalwart credentials
internal stack traces
HTTP authorization headers
```

---

# 46. PostgreSQL schema changes

Add/update migrations after Chapter 1.

Do NOT modify old migrations that have already been applied.

Create new migrations.

Example:

```text
006_user_auth.sql
007_domain_provisioning.sql
008_address_provisioning.sql
009_job_indexes.sql
```

Actual names can differ.

The important rule is:

**append migrations; don't rewrite migration history.**

---

# 47. Domain table changes

Add:

```text
stalwart_domain_id
verification_status
status
```

if already missing.

Recommended indexes:

```text
UNIQUE(name)
INDEX(user_id)
INDEX(status)
```

---

# 48. Address table changes

Recommended:

```text
id
domain_id
local_part
normalized_local_part
status
created_at
updated_at
```

If `local_part` is already intended to be normalized, avoid storing redundant copies unless useful.

The database uniqueness constraint must operate on the canonical value.

---

# 49. Mailbox table changes

Ensure:

```text
id
address_id
stalwart_account_id
status
created_at
updated_at
```

Constraints:

```text
UNIQUE(address_id)
UNIQUE(stalwart_account_id)
```

where appropriate.

---

# 50. Provisioning indexes

Add indexes optimized for the worker:

```text
(status, next_attempt_at)
(type, status)
(resource_id)
```

The worker must not scan the entire job table.

---

# 51. Authentication database indexes

Add:

```text
UNIQUE(lower(email))
```

or equivalent normalized-email storage.

Do not query users with unindexed email columns.

---

# 52. Rate limiting

Do not introduce Redis.

Implement a minimal application-level protection for login/register if needed.

A basic per-process limiter is acceptable for development.

Do not pretend it is the final distributed rate limiter.

The production distributed rate-limiting design belongs later.

---

# 53. Audit events

Chapter 2 should introduce a minimal audit system.

Add:

```text
audit_logs
```

with:

```text
id
user_id
action
resource_type
resource_id
metadata
created_at
```

Actions:

```text
USER_REGISTERED
USER_LOGGED_IN
DOMAIN_CREATED
DOMAIN_DISABLED
ADDRESS_CREATED
ADDRESS_DISABLED
MAILBOX_PROVISIONED
MAILBOX_PROVISIONING_FAILED
```

Do not log passwords/tokens.

Do not build an analytics pipeline.

---

# 54. Provisioning audit

Every provisioning transition should create an audit event.

Example:

```text
ADDRESS_CREATED
ACCOUNT_PROVISIONING_STARTED
ACCOUNT_PROVISIONED
```

A failed job should record a safe error description.

Do not include secret credentials.

---

# 55. Stalwart client changes

Expand:

```text
internal/stalwart/
```

to support:

```text
CreateDomain
GetDomain
DeleteDomain

CreateAccount
GetAccount
DisableAccount
EnableAccount
DeleteAccount
```

The current management API exposes configuration through JMAP objects under `urn:stalwart:jmap`, with methods documented as standard JMAP `get`/`set` operations; the official schema-driven CLI uses the same API.

Do not call Stalwart directly from:

```text
domains/service.go
addresses/service.go
```

Instead:

```text
domains/service
        ↓
provisioning
        ↓
stalwart.Client
```

---

# 56. Do not use stalwart-cli in the production worker

Use `stalwart-cli` for:

```text
development
manual operations
debugging
deployment scripts
```

The Norest worker must call the Stalwart JMAP HTTP API directly through `internal/stalwart`.

The official CLI itself is schema-driven and speaks the same JMAP management API, so using its operations to understand/debug the server is fine, but Norest should not spawn CLI processes for every mailbox.

---

# 57. Stalwart domain provisioning sequence

The worker must:

```text
load Norest domain
        ↓
check status
        ↓
check stalwart_domain_id
        ↓
if already present:
    verify and complete
        ↓
otherwise:
    x:Domain/set create
        ↓
capture returned Stalwart ID
        ↓
update Norest domain
        ↓
mark provisioning success
```

Do not create duplicate domains.

---

# 58. Stalwart account provisioning sequence

The worker must:

```text
load Norest mailbox
        ↓
load address
        ↓
load domain
        ↓
ensure domain has Stalwart ID
        ↓
check whether Stalwart account already exists
        ↓
create x:Account if absent
        ↓
capture account ID
        ↓
update Norest mailbox
        ↓
mark mailbox ACTIVE
        ↓
mark address ACTIVE
```

This creates the actual relationship:

```text
Norest address
        ↓
Stalwart account
```

---

# 59. Failure example

Suppose:

```text
POST /v1/domains/example.com/addresses
```

succeeds but Stalwart is down.

Expected result:

```text
HTTP 201
address.status = PENDING
mailbox.status = PENDING
job.status = PENDING/RETRY_WAIT
```

Not:

```text
HTTP 500
```

The product operation has been accepted; infrastructure provisioning is asynchronous.

---

# 60. Stalwart unavailable

If Stalwart is unavailable:

```text
Norest API
    ↓
still accepts domain/address creation
    ↓
worker retries
```

unless the product operation fundamentally cannot be recorded.

This is why the PostgreSQL transaction and worker are important.

---

# 61. Idempotency endpoint behavior

Add optional support for:

```http
Idempotency-Key: <value>
```

to:

```text
POST /v1/domains
POST /v1/domains/:domainID/addresses
```

For Chapter 2, a simple PostgreSQL-backed idempotency implementation is sufficient if implemented.

Do not introduce Redis.

If implementing this would significantly complicate the chapter, document it as a follow-up rather than overbuilding it.

---

# 62. Concurrency tests

This chapter must explicitly test concurrent address creation.

For example:

```text
100 concurrent requests
        ↓
POST alice@example.com
```

Expected:

```text
1 address
99 conflicts
```

No duplicate addresses.

This proves the PostgreSQL uniqueness constraint is doing its job.

---

# 63. Provisioning concurrency test

Test:

```text
same provisioning job
processed by multiple worker loops
```

Expected:

```text
one Stalwart account
one Norest mailbox
```

No duplicate accounts.

---

# 64. API tests

Add tests for:

```text
register user
duplicate user
login
wrong password
authenticated /me

create domain
duplicate domain
cross-user domain access

create address
duplicate address
reserved address
cross-user address access

delete/disable address
delete domain with addresses
```

---

# 65. Integration test: full provisioning

Create an integration test that performs:

```text
create Norest user
        ↓
login
        ↓
create domain
        ↓
wait for domain provisioning
        ↓
create address
        ↓
wait for mailbox provisioning
        ↓
verify mailbox ACTIVE
        ↓
verify stalwart_account_id exists
```

Then query Stalwart using the management client and verify the corresponding account exists.

---

# 66. Integration test: actual mail login

Using the mailbox created by Norest provisioning:

```text
alice@example.test
```

authenticate against Stalwart JMAP.

Verify:

```text
/.well-known/jmap
        ↓
session
```

works for the newly provisioned mailbox.

Then:

```text
Mailbox/get
```

must work.

This proves the product-created account is actually usable by the mail engine.

---

# 67. Integration test: send/receive regression

Do not delete the Chapter 1 E2E test.

Keep it.

Chapter 2 must still pass:

```text
Alice
  ↓
JMAP
  ↓
Stalwart
  ↓
Bob
```

The new provisioning system must not break the Chapter 1 mail proof.

Keep:

```text
scripts/test-jmap-e2e/
```

and run it after the provisioning tests.

---

# 68. Database isolation regression

Keep:

```text
scripts/verify-db-clean.sh
```

and ensure it still passes.

The expected Norest database contains product/control-plane tables only.

There must still be zero:

```text
messages
folders
attachments
mail bodies
```

---

# 69. API documentation

Expand:

```text
api/openapi.yaml
```

to include:

```text
POST /v1/auth/register
POST /v1/auth/login
POST /v1/auth/refresh
GET  /v1/me

POST   /v1/domains
GET    /v1/domains
GET    /v1/domains/{id}
PATCH  /v1/domains/{id}
DELETE /v1/domains/{id}

POST   /v1/domains/{id}/addresses
GET    /v1/domains/{id}/addresses

GET    /v1/addresses/{id}
PATCH  /v1/addresses/{id}
DELETE /v1/addresses/{id}
```

Do not document internal Stalwart management endpoints as public Norest API.

---

# 70. Folder structure after Chapter 2

Keep the Chapter 1 structure.

Add only what is necessary:

```text
internal/
├── auth/
│   ├── service.go
│   ├── handler.go
│   ├── repository.go
│   ├── models.go
│   ├── password.go
│   └── token.go
│
├── users/
│   ├── service.go
│   ├── repository.go
│   └── models.go
│
├── domains/
│   ├── service.go
│   ├── handler.go
│   ├── repository.go
│   ├── models.go
│   ├── normalize.go
│   └── state.go
│
├── addresses/
│   ├── service.go
│   ├── handler.go
│   ├── repository.go
│   ├── models.go
│   ├── normalize.go
│   └── reserved.go
│
├── provisioning/
│   ├── service.go
│   ├── worker.go
│   ├── repository.go
│   ├── jobs.go
│   └── retry.go
│
├── stalwart/
│   ├── client.go
│   ├── session.go
│   ├── management.go
│   ├── domain.go
│   ├── account.go
│   ├── mailbox.go
│   └── email.go
│
├── audit/
│   ├── service.go
│   └── repository.go
```

Do not create separate microservices for any of these.

---

# 71. No frontend work

Do not rebuild the frontend.

Do not create frontend pages.

Do not create a second frontend.

Implement the backend APIs and make them easy for the existing frontend to consume.

A developer can test them with curl/Postman before the UI is connected.

---

# 72. Security requirements

Never log:

```text
password
access token
refresh token
Stalwart admin credential
mailbox credential
Authorization header
```

Password hashing must be one-way.

Access tokens must expire.

Refresh tokens must be protected.

Ownership checks must be enforced server-side.

Do not expose internal Stalwart IDs unnecessarily.

---

# 73. Scalability requirements

The Chapter 2 code must already be compatible with horizontal scaling.

That means:

```text
no in-memory user state
no in-memory provisioning state
no process-local authoritative sessions
no local mailbox database
no local job ownership
```

Multiple Norest API instances must be able to run simultaneously.

Multiple worker instances must be able to run simultaneously.

PostgreSQL is the coordination authority.

---

# 74. API statelessness

Any API request may reach:

```text
API-1
API-2
API-3
...
```

and get the same result.

Do not depend on:

```text
process memory
sticky sessions
local files
```

for correctness.

---

# 75. Worker scalability

The worker must be safe to scale from:

```text
1 process
```

to:

```text
10 processes
```

without creating duplicate Stalwart accounts.

Use PostgreSQL job claiming.

---

# 76. Observability

Do not create a full observability stack yet.

Add structured logs with fields:

```text
request_id
user_id
resource_id
job_id
job_type
attempt
duration
status
```

Never include secrets.

Metrics/tracing can be added later.

---

# 77. Request IDs

Add a request ID middleware.

If the client sends:

```text
X-Request-ID
```

validate/use it.

Otherwise generate one.

Return it in the response:

```text
X-Request-ID
```

This will make provisioning/API debugging much easier.

---

# 78. Database transaction boundaries

The following operations must be transactional:

```text
domain creation + domain job
address creation + mailbox + provisioning job
user registration
state transitions
job claiming
```

Do not wrap remote Stalwart HTTP calls inside a long PostgreSQL transaction.

Never hold a database transaction open while waiting on Stalwart.

This is critical.

---

# 79. Remote operation pattern

Correct:

```text
DB transaction
    ↓
commit desired state/job
    ↓
remote Stalwart operation
    ↓
new DB transaction
    ↓
record result
```

Incorrect:

```text
BEGIN
   ↓
call Stalwart
   ↓
wait
   ↓
call another system
   ↓
COMMIT
```

Do not hold database locks during network calls.

---

# 80. Provisioning state machine

For mailbox:

```text
PENDING
   ↓
PROVISIONING
   ↓
ACTIVE
```

Failure:

```text
PROVISIONING
   ↓
RETRY_WAIT
   ↓
PROVISIONING
```

Permanent failure:

```text
PROVISIONING
   ↓
FAILED
```

Manual recovery can later transition:

```text
FAILED → PENDING
```

Keep state transitions explicit.

---

# 81. Domain and mailbox state relationship

A mailbox should not become active when its domain is unavailable.

Expected dependency:

```text
Domain ACTIVE
       ↓
Address creation allowed
       ↓
Mailbox provisioning allowed
       ↓
Mailbox ACTIVE
```

If the domain is:

```text
SUSPENDED
DISABLED
```

new mailbox creation must be rejected.

---

# 82. Account limits

Do not implement billing yet.

However, create a place in the service for future limits:

```go
CanCreateDomain(userID)
CanCreateAddress(domainID)
```

For Chapter 2, return true except for hard product safeguards.

Do not create a billing subsystem just to support this.

---

# 83. Reserved platform domains

Create a mechanism to distinguish:

```text
platform-owned domains
```

from:

```text
customer-owned domains
```

but keep implementation minimal.

For Chapter 2, it is enough to add a field such as:

```text
ownership_type
```

or equivalent if the current domain schema supports it cleanly.

Do not build a complete domain marketplace.

---

# 84. Norest-owned email address

The architecture must support:

```text
alice@norest.com
```

eventually being a platform-owned address while:

```text
alice@example.com
```

is customer-domain-owned.

The address table must therefore not assume every domain belongs to a normal customer in perpetuity.

Design this carefully without implementing the full platform-domain system yet.

---

# 85. Development test data

Create a deterministic seed/test command if useful.

Recommended development test data:

```text
user:
alice@example.com

domain:
example.test

addresses:
alice@example.test
bob@example.test
```

Do not put real passwords in Git.

Use environment variables for test passwords.

---

# 86. Chapter 1 regression checklist

Run:

```bash
go test ./...
docker compose config
./scripts/test-foundation.sh
./scripts/verify-db-clean.sh
cd scripts/test-jmap-e2e && go run main.go
```

All must pass.

---

# 87. Chapter 2 new acceptance test

Run a new script:

```text
scripts/test-provisioning-e2e.sh
```

It must prove:

```text
register
   ↓
login
   ↓
create domain
   ↓
domain provisioning
   ↓
create address
   ↓
mailbox provisioning
   ↓
Stalwart account exists
   ↓
mailbox authentication succeeds
   ↓
Mailbox/get succeeds
```

The script must exit non-zero if anything fails.

---

# 88. No manual Stalwart account creation

This is a critical acceptance condition.

Chapter 1 used manually/scripts-created Stalwart users to prove JMAP.

Chapter 2 must prove:

```text
Norest creates the product resource
        ↓
Norest Worker creates Stalwart account
```

The human must not manually create Alice in Stalwart.

That would not prove the product architecture.

---

# 89. No manual Stalwart domain creation

Same rule.

The user creates:

```http
POST /v1/domains
```

and Norest provisions the corresponding Stalwart domain.

No admin UI step should be required for ordinary development provisioning after the initial Stalwart server bootstrap.

---

# 90. Chapter 2 completion criteria

Chapter 2 is complete only when all are true:

```text
[ ] User registration works.

[ ] Password hashing works.

[ ] Login works.

[ ] Access-token authentication works.

[ ] GET /v1/me works.

[ ] Domain creation works.

[ ] Domain ownership checks work.

[ ] Domain normalization works.

[ ] Domain uniqueness is database-enforced.

[ ] Address creation works.

[ ] Address ownership checks work.

[ ] Local-part normalization works.

[ ] Reserved local parts are rejected.

[ ] Address uniqueness is database-enforced.

[ ] Mailbox record is created.

[ ] Provisioning job is created transactionally.

[ ] Worker claims jobs safely.

[ ] Worker retries failures.

[ ] Stalwart domain is created automatically.

[ ] Stalwart account is created automatically.

[ ] Stalwart IDs are stored in Norest.

[ ] No mail contents enter PostgreSQL.

[ ] Provisioned mailbox can authenticate to JMAP.

[ ] Mailbox/get succeeds for a Norest-provisioned account.

[ ] Chapter 1 JMAP E2E test still passes.

[ ] Database-clean test still passes.

[ ] Concurrent duplicate address test passes.

[ ] Multiple worker processes do not duplicate accounts.

[ ] API ownership checks pass.

[ ] go test ./... passes.

[ ] docker compose config passes.

[ ] test-foundation.sh passes.

[ ] test-provisioning-e2e.sh passes.

[ ] README updated.

[ ] OpenAPI updated.

[ ] architecture documentation updated.
```

---

# 91. What Chapter 2 must NOT contain

Do NOT build:

```text
Kafka
Redis
RabbitMQ
Temporal
Kubernetes
service mesh
microservices
Elasticsearch
custom SMTP
custom IMAP
custom JMAP
mail message database
mail search database
webmail backend proxy
billing
Stripe integration
DNS automation
multi-region
multi-tenant service mesh
```

Do not solve tomorrow's problems today.

---

# 92. The exact architecture after Chapter 2

The system should now look like:

```text
                         USER
                          |
                    Norest REST API
                          |
             +------------+-------------+
             |                          |
             v                          v
        PostgreSQL                 Norest Worker
             |                          |
             |                          v
             |                    Stalwart JMAP
             |                          |
             +--------------------------+
                                        |
                                 Actual Mail System
```

More specifically:

```text
Norest PostgreSQL
─────────────────

users
domains
addresses
mailboxes
provisioning_jobs
audit_logs


Stalwart
────────

accounts
mailboxes
folders
messages
attachments
search
delivery
SMTP
IMAP
JMAP
```

Norest stores:

```text
alice@example.com
```

and:

```text
stalwart_account_id = abc123
```

Stalwart stores:

```text
Inbox
Sent
Drafts
message content
attachments
flags
threads
```

---

# 93. What the user should be able to do after Chapter 2

Using only the Norest API:

```text
Register
    ↓
Login
    ↓
Create domain
    ↓
Create email address
    ↓
Wait for provisioning
    ↓
Address becomes ACTIVE
```

And then the actual mailbox exists in Stalwart.

The developer can authenticate the new account against JMAP and retrieve its mailbox list.

That is the first point where Norest becomes a real mail product rather than a server scaffold.

---

# 94. Final implementation instruction to Devin

Work directly in the current repository.

Do not rewrite Chapter 1.

Do not create a new application.

Extend the existing code.

Implement:

```text
identity
authentication
domains
addresses
provisioning
audit
Stalwart domain provisioning
Stalwart account provisioning
```

Then run:

```bash
go test ./...
docker compose config
./scripts/test-foundation.sh
./scripts/verify-db-clean.sh
./scripts/test-provisioning-e2e.sh
cd scripts/test-jmap-e2e && go run main.go
```

Fix every issue.

Do not report "implemented" until the actual end-to-end provisioning path has been executed.

At the end report:

```text
1. Files changed
2. Database migrations added
3. API endpoints added
4. Authentication implementation
5. Domain lifecycle
6. Address lifecycle
7. Provisioning lifecycle
8. Stalwart management operations used
9. Retry behavior
10. Concurrency behavior
11. Full provisioning E2E result
12. JMAP regression result
13. Database-isolation result
14. Exact commands to run the system
15. Known limitations
```

The most important proof is:

```text
Norest
   ↓
POST /v1/auth/register
   ↓
POST /v1/auth/login
   ↓
POST /v1/domains
   ↓
POST /v1/domains/{id}/addresses
   ↓
PostgreSQL job
   ↓
Norest worker
   ↓
Stalwart x:Domain/set
   ↓
Stalwart x:Account/set
   ↓
ACTIVE mailbox
   ↓
authenticate newly-created mailbox with JMAP
   ↓
Mailbox/get
```

**Do not move to Chapter 3 until this entire path works against the real running Stalwart instance.**

The goal is not a larger server.

The goal is:

> **A user can create a real Norest email address, and Norest automatically turns that product resource into a real Stalwart mailbox without Norest ever becoming the mail engine.**
