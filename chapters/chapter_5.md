# Norest Mail — Chapter 5

## Production Hardening, Reliability, Security, Disaster Recovery & Scale

You are continuing Norest Mail after Chapter 4.

Chapter 1, Chapter 2, Chapter 3, and Chapter 4 are complete and frozen.

Treat them as the stable foundation.

Do NOT redesign the architecture.

Do NOT add product features.

Do NOT build frontend functionality.

Do NOT introduce infrastructure merely because it is common in large systems.

The purpose of Chapter 5 is to prove that the existing Norest Mail architecture is production-capable and can scale toward millions of users without becoming unnecessarily complicated.

---

# 1. Immutable architecture

The architecture remains:

```text id="0r9o6a"
                    INTERNET
                        |
                 Load Balancer
                        |
              +---------+---------+
              |         |         |
            API-1     API-2     API-N
              |         |         |
              +---------+---------+
                        |
                   PostgreSQL
                        |
                 Provisioning
                    Workers
                        |
                    Stalwart
                        |
          +-------------+-------------+
          |             |             |
         JMAP         IMAP           SMTP
          |
       Mail Data
```

Norest remains:

```text id="33o3lb"
identity
product account
domains
addresses
plans
subscriptions
entitlements
billing state
policy
provisioning
audit
authorization
```

Stalwart remains:

```text id="jv3r1p"
mailboxes
messages
folders
threads
attachments
search
JMAP
IMAP
SMTP
delivery
mail queue
mail storage
```

PostgreSQL remains the Norest control-plane authority.

Stalwart remains the mail-plane authority.

---

# 2. Chapter 5 objective

At the end of Chapter 5, we must be able to say:

> Norest Mail can run as a horizontally scalable production system, survive normal infrastructure failures, protect customer data, recover from crashes, prevent duplicate provisioning, and operate against a large user population without changing the core architecture.

This chapter is about **proof**, not feature count.

---

# 3. What Chapter 5 must NOT introduce

Do NOT introduce:

```text id="w8k5t0"
Kafka
Redis
RabbitMQ
NATS
Temporal
Elasticsearch
Kubernetes
service mesh
distributed lock service
custom SMTP
custom IMAP
custom JMAP
Norest message storage
Norest attachment storage
Norest search engine
multi-region active/active
```

unless a benchmark or failure test in this chapter proves that the existing architecture cannot meet a documented requirement.

Do not introduce technology just because a "large company" might use it.

---

# 4. Chapter 5 philosophy

Use this rule throughout:

```text id="v8jpj0"
Measure first.
Fix second.
Introduce infrastructure only when justified.
```

Every major scalability claim must have:

```text id="c8f2c2"
test
measurement
result
```

Do not claim "millions of users supported" simply because the code is horizontally scalable.

Demonstrate the architecture's scaling characteristics and identify the remaining infrastructure assumptions.

---

# 5. Production configuration separation

Create a hard separation between:

```text id="k7zh0e"
development
test
production
```

Production must not inherit development shortcuts.

No production environment may use:

```text id="w0v1s8"
change-me-development-only
test passwords
automatic Stalwart bootstrap
development CORS
unsafe HTTP
debug endpoints
```

---

# 6. Development-only Stalwart bootstrap

The existing development bootstrap mechanism must remain development-only.

Production startup must never automatically execute:

```text id="cv5p9v"
x:Bootstrap/set
```

Production Stalwart bootstrap/configuration must require a secure deployment process.

The application must fail safely if a production environment incorrectly exposes development bootstrap variables.

Add an explicit guard such as:

```text id="4q4o5z"
APP_ENV=production
```

must disable all development bootstrap code.

---

# 7. Secrets management

Review every secret used by the system.

Potential secrets include:

```text id="dkwg24"
DATABASE_URL credentials
JWT signing secret
refresh-token secret
Stalwart admin credential
Stalwart management credential
mail provisioning credentials where applicable
verification secrets
billing webhook secret
production API keys
```

Requirements:

* no secret committed to Git
* no secret hardcoded in Go source
* no secret in shell scripts except development placeholders
* no secret in logs
* no secret in error responses
* no secret in OpenAPI
* no secret in documentation
* no secret in Docker image layers

---

# 8. Environment validation

Create strong production configuration validation.

At startup:

```text id="sa8y8v"
APP_ENV=production
```

must require secure values.

Examples:

```text id="z8nyf2"
JWT_SECRET
DATABASE_URL
STALWART management credentials
allowed origins
```

Development-only values must not silently fall back in production.

Fail fast.

---

# 9. Credential rotation

Document how these credentials can be rotated:

```text id="6pmmuy"
JWT signing secret
database password
Stalwart management credential
billing webhook secret
```

Do not build an elaborate secret-management platform.

Provide safe configuration boundaries and documented rotation procedures.

---

# 10. Authentication hardening

Review Chapter 2 authentication.

Verify:

```text id="bq9nqz"
password hashing
access-token expiry
refresh-token security
invalid token handling
user suspension behavior
admin role enforcement
```

Do not redesign authentication unless a real vulnerability is found.

---

# 11. Access-token rules

Access tokens must:

```text id="r8m2k8"
expire
be audience/issuer validated where applicable
be rejected after invalidation conditions where applicable
not contain sensitive product data
```

Do not place:

```text id="8h3mjo"
passwords
mail credentials
billing secrets
```

inside tokens.

---

# 12. Refresh-token rules

Refresh tokens must be treated as sensitive credentials.

Review:

```text id="v40d91"
storage
expiration
rotation policy
revocation
reuse behavior
```

Do not log refresh tokens.

Do not expose refresh tokens to admin APIs.

---

# 13. Account suspension and authentication

Verify that:

```text id="pvk6g1"
ACTIVE user
    ↓
valid token
```

works.

After:

```text id="lm2h6o"
SUSPEND
```

future authenticated operations must obey the suspension policy.

Do not rely solely on an already-issued JWT to keep a suspended account fully operational forever.

Define the intended behavior explicitly and test it.

---

# 14. Authorization audit

Review every protected endpoint.

Build a matrix:

```text id="1v9dca"
endpoint
method
anonymous
user
admin
resource owner
```

Verify:

```text id="um8y3f"
a user cannot access another user's domain
a user cannot access another user's address
a user cannot access another user's account
a user cannot call admin APIs
a suspended user cannot create resources
```

---

# 15. Object-level authorization

Do not rely on:

```text id="s6d9f6"
UUID secrecy
```

for authorization.

Every resource lookup must verify ownership.

Test ID substitution attacks:

```text id="4znbbu"
Alice token
+
Bob domain UUID
=
must fail
```

---

# 16. Input validation

Audit all external inputs:

```text id="x3w0sc"
email
domain
local_part
plan identifiers
resource IDs
pagination
sort values
search strings
verification data
admin parameters
```

Reject malformed input early.

Prevent:

```text id="znm5j1"
SQL injection
header injection
path traversal
unsafe shell execution
unexpected JSON structures
```

Use parameterized SQL exclusively.

---

# 17. HTTP hardening

Implement production-safe HTTP defaults:

```text id="vfh2r3"
read timeout
write timeout
idle timeout
request body limits
header limits
connection limits where appropriate
```

Do not allow a request to hold a connection forever.

---

# 18. Outgoing HTTP safety

All Norest → Stalwart and external HTTP calls must use:

```text id="l9p3b7"
context
timeout
bounded response handling
explicit error handling
```

No indefinite HTTP requests.

No uncontrolled retries.

---

# 19. Retry policy

Define a common retry classification:

```text id="4w0o25"
temporary
permanent
authentication failure
authorization failure
not found
conflict
rate limited
timeout
```

Only retry errors that are genuinely retryable.

---

# 20. Retry backoff

Use the existing worker architecture.

Retry schedule should use:

```text id="0pav3o"
exponential backoff
jitter
maximum delay
maximum attempts
```

Do not use:

```text id="yf0iv6"
busy loops
fixed 1-second retries
infinite retries
```

---

# 21. Provisioning idempotency

This is one of the most important requirements.

For every job:

```text id="q8dntp"
DOMAIN_CREATE
ACCOUNT_CREATE
ACCOUNT_DISABLE
ACCOUNT_REACTIVATE
ACCOUNT_QUOTA_SYNC
DOMAIN_VERIFY
```

a retry must not create duplicate resources.

---

# 22. Worker crash testing

Test:

```text id="c3gi4f"
worker claims job
worker crashes
```

before completion.

After restart:

```text id="m1kq7r"
job must be recoverable
```

There must be no permanent stuck state.

---

# 23. Worker visibility timeout

Define how a `PROCESSING` job is recovered.

If:

```text id="vp18s2"
status = PROCESSING
```

and the worker dies:

```text id="m1cgr0"
job must eventually become reclaimable
```

Do not leave jobs permanently locked.

Implement a lease/heartbeat/recovery mechanism using PostgreSQL.

Do not add Redis.

---

# 24. Worker heartbeat

If appropriate, add:

```text id="d1e0w5"
claimed_at
heartbeat_at
```

or equivalent.

The system must distinguish:

```text id="8s1wq0"
active processing
dead worker
slow operation
```

Do not overbuild this.

A simple PostgreSQL lease is sufficient.

---

# 25. Multiple worker instances

Run:

```text id="d24rjl"
3 workers
```

against the same database.

Submit many provisioning jobs.

Verify:

```text id="h17n8f"
no duplicate processing
no duplicate Stalwart accounts
no corrupted states
```

---

# 26. API horizontal scaling

Run:

```text id="2zti6o"
API-1
API-2
API-3
```

against the same PostgreSQL and Stalwart.

All APIs must remain stateless.

Requests may hit any instance.

No correctness may depend on:

```text id="9ol5gi"
sticky sessions
process memory
local filesystem
```

---

# 27. API session behavior

Do not store authoritative session state only in process memory.

If an API container dies:

```text id="z0zwnr"
another API instance
```

must be able to continue serving the user.

---

# 28. Database connection pooling

Review PostgreSQL connections.

Requirements:

```text id="b6xv2u"
bounded pool
context cancellation
connection health
idle timeout
max connection lifetime
```

Do not create one database connection per request without pooling.

---

# 29. Database timeout policy

Every database operation must have a sensible context deadline.

No unbounded:

```text id="muwzqj"
SELECT
INSERT
UPDATE
```

operations.

---

# 30. PostgreSQL indexes

Review actual queries generated by:

```text id="l6bpwq"
authentication
domains
addresses
mailboxes
subscriptions
entitlements
jobs
audit
billing events
```

Verify useful indexes exist.

Do not add indexes simply because columns "might be searched."

---

# 31. Slow query detection

Provide a production path to identify:

```text id="0eu3zh"
slow queries
lock waits
connection saturation
```

Do not deploy a full observability platform yet.

Configuration/documentation is sufficient.

---

# 32. Transaction review

Audit every transaction.

Ensure no transaction holds locks while waiting for:

```text id="fq6h9q"
Stalwart
DNS
billing provider
external HTTP
```

A database transaction must not span long remote calls.

---

# 33. Deadlock handling

Identify likely lock-order problems.

Add tests for:

```text id="sh1w7w"
concurrent limit enforcement
concurrent subscription changes
concurrent provisioning
concurrent suspension
```

Transactions must fail cleanly if PostgreSQL reports a transient serialization/deadlock condition.

Only safe transactions may retry.

---

# 34. Readiness vs liveness

Implement separate health endpoints.

For example:

```text id="lw4xqk"
/health/live
/health/ready
/health/db
/health/stalwart
```

Meaning:

```text id="9ffndw"
live
= process is alive

ready
= process can serve traffic

db
= PostgreSQL dependency

stalwart
= mail-engine dependency
```

Do not use a generic `/health = OK` for all purposes.

---

# 35. Readiness behavior

If Stalwart is temporarily unavailable:

```text id="nqmmtr"
Norest process may remain alive
```

but dependency-sensitive operations should fail gracefully.

Do not necessarily kill the entire API just because Stalwart is restarting.

---

# 36. Startup orchestration

Remove arbitrary sleeps as the primary readiness mechanism.

Do not depend on:

```text id="p1qsz3"
sleep 10
```

for correctness.

Use active readiness checks.

Development scripts may still retry while services start.

Production orchestration must use actual readiness.

---

# 37. Stalwart dependency handling

Test:

```text id="m0pv2d"
Stalwart stopped
Stalwart restarting
Stalwart slow
Stalwart returns 5xx
Stalwart times out
```

Norest must:

```text id="7i2q8a"
avoid request hangs
record meaningful errors
retry only where safe
keep product state consistent
recover automatically where possible
```

---

# 38. Stalwart provisioning ambiguity

Critical case:

```text id="a7w2mw"
Norest → Stalwart CREATE
network timeout
```

The operation may have succeeded even though Norest did not receive the response.

The retry path must first check whether the object already exists.

This prevents:

```text id="u4pcaf"
duplicate domain
duplicate account
```

---

# 39. DNS failure behavior

Verify:

```text id="7yksuz"
DNS unavailable
DNS times out
wrong answer
empty answer
temporary resolver error
```

The worker must not incorrectly mark a domain verified.

---

# 40. Billing provider failure

Where billing integration exists:

```text id="k18h1n"
timeout
5xx
duplicate webhook
malformed webhook
```

must not corrupt subscription state.

---

# 41. Rate limiting

Implement a production-safe rate-limiting abstraction.

Protect at minimum:

```text id="1b6ygi"
POST /v1/auth/register
POST /v1/auth/login
POST /v1/domains
POST /v1/domains/:id/addresses
POST /v1/domains/:id/verification/check
POST /v1/mail/session
admin endpoints
billing webhook where appropriate
```

---

# 42. Do not introduce Redis for rate limiting yet

Start with the simplest system that allows horizontal deployment.

If distributed rate limiting is necessary, document the boundary for a future implementation.

Do not build a giant Redis cluster merely to rate-limit Chapter 5.

---

# 43. Login protection

Implement sensible protection against:

```text id="ydpsv9"
brute force
credential stuffing
rapid repeated failed logins
```

Avoid locking accounts permanently after a few failures.

Use temporary controls.

---

# 44. Registration abuse

Protect registration against:

```text id="oa8h4x"
automated account creation
rapid repeated signups
```

A production CAPTCHA/email verification system can come later.

Chapter 5 only needs the backend rate-control boundary.

---

# 45. Admin protection

Admin endpoints need stronger rate limiting and auditability than normal endpoints.

All admin mutations must generate audit records.

---

# 46. CORS

Production CORS must never be:

```text id="oxrd6v"
*
```

when credentials are involved.

Use an explicit allowed-origin list.

Development origins may differ from production origins.

---

# 47. Security headers

Add appropriate HTTP security headers where applicable:

```text id="67akdy"
X-Content-Type-Options
Content-Security-Policy where applicable
Referrer-Policy
frame restrictions
```

Do not blindly copy a header set that breaks the API.

---

# 48. TLS

Production should use TLS end-to-end where appropriate.

Development HTTP may remain allowed for local Docker usage.

Do not embed production certificates in the repository.

---

# 49. Database backup

Define a PostgreSQL backup strategy.

At minimum document:

```text id="0jkn93"
logical backup
restore
backup verification
retention
encryption
```

Do not call a backup "working" until restore has been tested.

---

# 50. Backup test

Perform:

```text id="9m18o9"
backup database
destroy development database
restore backup
run application
run verification
```

Verify:

```text id="1cz4jx"
users
domains
addresses
mailbox linkage
plans
subscriptions
audit
```

are recovered correctly.

---

# 51. Important data distinction in backups

Norest PostgreSQL does NOT contain the actual mail.

Therefore document:

```text id="0cnmqd"
Norest backup
    =
control-plane recovery

Stalwart backup
    =
mail-data recovery
```

Do not claim a PostgreSQL backup restores customer mail.

---

# 52. Stalwart recovery

Document and test the Stalwart data recovery process separately.

The Norest system must know how to recover:

```text id="u9v9dt"
Stalwart mail data
Stalwart configuration
Norest-to-Stalwart identifiers
```

Do not build a second mail-backup system inside Norest.

---

# 53. Cross-system recovery

Test:

```text id="g6d0ms"
Norest DB restored
Stalwart intact
```

and:

```text id="3ph0xz"
Stalwart restored
Norest DB intact
```

Define reconciliation behavior for mismatches.

---

# 54. Critical identifier recovery

Norest stores:

```text id="yk5v3g"
stalwart_domain_id
stalwart_account_id
```

These mappings are critical.

Backups and restore procedures must preserve them.

Do not regenerate them casually.

---

# 55. Reconciliation after restore

After recovery, run:

```text id="p0g64n"
domain reconciliation
mailbox/account reconciliation
quota reconciliation
```

and identify:

```text id="klt7hz"
missing
extra
mismatched
```

objects.

---

# 56. Audit logging

Review audit logging for production.

All important control-plane mutations must be auditable.

Log:

```text id="9if5z9"
who
what
resource
when
result
request_id
```

Do not log secrets.

---

# 57. Request IDs

Every request should have:

```text id="u3b94f"
request_id
```

and the ID should appear in:

```text id="scou2t"
HTTP response
structured logs
relevant job logs
audit context
```

---

# 58. Structured logging

Use structured logs.

Recommended fields:

```text id="bknfmu"
timestamp
level
service
request_id
user_id
account_id
resource_id
job_id
job_type
duration_ms
status
error_code
```

Do not log full request bodies by default.

---

# 59. Log redaction

Explicitly redact:

```text id="8s2h6b"
Authorization headers
passwords
JWTs
refresh tokens
verification tokens
Stalwart credentials
billing secrets
```

Test log redaction.

---

# 60. Error response hardening

Production errors must not expose:

```text id="8rc9wq"
stack traces
SQL statements
internal filesystem paths
Stalwart credentials
environment variables
```

Return stable error codes.

---

# 61. Metrics

Implement lightweight metrics.

At minimum measure:

```text id="t5cyq2"
HTTP request count
HTTP error count
HTTP latency
database latency
database errors
Stalwart request latency
Stalwart errors
provisioning jobs processed
provisioning failures
provisioning retries
queue depth
```

Do not build a full metrics platform if there is already an acceptable deployment mechanism.

Expose metrics through a standard endpoint if appropriate.

---

# 62. Queue metrics

Track:

```text id="2y2wp5"
pending jobs
processing jobs
failed jobs
retry-wait jobs
oldest pending job
```

This identifies provisioning backlog.

---

# 63. Dependency metrics

Measure:

```text id="qps0bp"
PostgreSQL latency
Stalwart latency
DNS lookup latency
billing provider latency
```

where those dependencies exist.

---

# 64. Alert thresholds

Document sensible operational alarms.

For example:

```text id="8j2i8l"
worker backlog rising
database connection saturation
high 5xx
high Stalwart latency
many failed provisioning jobs
readiness failures
disk capacity warnings
```

Do not hardcode vendor-specific monitoring assumptions.

---

# 65. Capacity planning

Document what each component scales on.

## Norest API

Scales on:

```text id="fm3gn5"
CPU
memory
HTTP concurrency
```

## Worker

Scales on:

```text id="b8y9tp"
job backlog
Stalwart latency
DB contention
```

## PostgreSQL

Scales on:

```text id="f23e1o"
connections
IO
CPU
storage
locks
query volume
```

## Stalwart

Scales on:

```text id="ek8lpy"
mail traffic
storage
JMAP concurrency
SMTP volume
mailbox count
```

Do not pretend one component's scaling solves every bottleneck.

---

# 66. API load test

Create a reproducible load test.

At minimum:

```text id="iuvxus"
1,000 concurrent authenticated requests
```

against low-cost endpoints such as:

```text id="x4f76d"
GET /v1/me
GET /v1/account/usage
GET /v1/account/entitlements
```

Measure:

```text id="f0ft15"
requests/sec
p50
p95
p99
error rate
CPU
memory
DB connections
```

---

# 67. Provisioning load test

Run a controlled test with:

```text id="vhlty8"
1,000+ address/mailbox provisioning jobs
multiple workers
```

Measure:

```text id="3dy4iv"
throughput
job latency
retry rate
DB contention
Stalwart latency
duplicate rate
failure rate
```

---

# 68. Do not claim millions from a 1,000-user test

The test does not prove:

```text id="4vd1jd"
"supports 10 million users"
```

Instead report:

```text id="p9r40x"
measured throughput
measured bottlenecks
estimated scaling behavior
remaining assumptions
```

This is much more trustworthy.

---

# 69. Database scaling assessment

Measure PostgreSQL behavior under Chapter 5 load.

Identify:

```text id="4u6m8j"
slowest query
highest connection consumer
highest lock contention
largest table growth
```

Do not prematurely shard PostgreSQL.

---

# 70. Table growth

Review expected growth of:

```text id="1jptbz"
users
domains
addresses
mailboxes
subscriptions
audit_logs
provisioning_jobs
billing_events
```

Identify which tables may become large.

---

# 71. Retention policies

Define initial retention for:

```text id="oxj8md"
audit logs
billing events
completed provisioning jobs
failed provisioning jobs
```

Do not allow provisioning_jobs to grow indefinitely.

Archive/delete safely according to documented retention.

---

# 72. Audit retention

Audit logs may need longer retention than worker jobs.

Define a retention policy.

Do not casually delete audit history.

---

# 73. Billing-event retention

Billing webhook payloads may be sensitive.

Store only what is necessary.

Hash large payloads when appropriate.

Do not retain payment secrets.

---

# 74. Background cleanup

Implement a simple cleanup worker or scheduled command for:

```text id="r9qyd5"
old completed provisioning jobs
old failed jobs beyond retention
expired temporary verification records
expired temporary state
```

Do not introduce a scheduling platform.

---

# 75. Job backlog protection

If provisioning backlog becomes large:

```text id="7sl9ww"
API remains responsive
```

and the system must provide visibility into backlog.

Do not allow unlimited in-memory job queues.

PostgreSQL remains the queue.

---

# 76. Graceful shutdown

API and workers must:

```text id="38h2pk"
stop accepting new work
finish safe in-flight operations
release DB connections
release HTTP connections
stop polling
```

within a configurable shutdown deadline.

---

# 77. Worker shutdown during Stalwart operation

If shutdown occurs during a remote Stalwart call:

```text id="x5m0ai"
do not mark job successful prematurely
```

The job must become safely retryable.

---

# 78. API shutdown

If API receives SIGTERM:

```text id="g75j07"
stop accepting new requests
finish in-flight requests
exit cleanly
```

No request should be silently abandoned if avoidable.

---

# 79. Docker hardening

Review the Docker images.

Ensure:

```text id="ycw7f8"
non-root
minimal runtime
pinned versions
no development tools in runtime image
no secrets baked into image
```

---

# 80. Container healthchecks

All critical services should expose useful health/readiness checks.

Do not use a healthcheck that only checks whether a process exists.

Check actual service availability.

---

# 81. Resource limits

Development Compose may remain simple.

For production deployment documentation, define recommended:

```text id="7k6qrn"
CPU
memory
file descriptors
database connections
```

Do not invent production values without measurements.

Document them as starting ranges.

---

# 82. File descriptor planning

Mail services use many network connections.

Review system/container file descriptor assumptions for:

```text id="0z3ah8"
SMTP
IMAP
JMAP
HTTP
database
```

Do not leave accidental low defaults.

---

# 83. Connection limits

Review:

```text id="x0txxd"
PostgreSQL connections
Norest HTTP connections
Stalwart JMAP connections
SMTP connections
```

Avoid exhausting one service by allowing unlimited concurrency.

---

# 84. Backpressure

When dependencies are overloaded:

```text id="mby8sb"
Norest must degrade gracefully
```

Use:

```text id="x3i2fd"
bounded concurrency
timeouts
429/503 where appropriate
job backpressure
```

Do not queue unlimited work in RAM.

---

# 85. Mail-session abuse

Review:

```text id="ob78ga"
POST /v1/mail/session
```

for abuse.

Prevent attackers from repeatedly generating credentials/session artifacts.

Use short-lived artifacts and rate limiting.

---

# 86. Security scanning

Run available security tooling against:

```text id="9di5xn"
Go dependencies
Dockerfile
container image
configuration
```

Fix critical/high findings that are actually relevant.

Do not introduce a huge security platform.

---

# 87. Dependency management

Review:

```text id="kn97qt"
Go dependencies
Docker base images
Stalwart version
PostgreSQL version
```

Pin versions deliberately.

Do not use:

```text id="8rrd9b"
latest
```

for critical production dependencies.

---

# 88. Dependency update policy

Document:

```text id="boh44g"
how dependencies are updated
how security updates are tested
how Stalwart upgrades are validated
```

Do not automatically upgrade production components without tests.

---

# 89. Stalwart upgrade compatibility

Because Norest interacts with Stalwart's management/JMAP schemas:

Every Stalwart version update must run:

```text id="n3g3vr"
Chapter 1 tests
Chapter 2 tests
Chapter 3 tests
Chapter 4 tests
Chapter 5 compatibility tests
```

before adoption.

---

# 90. API compatibility

Do not silently break existing API contracts.

Keep OpenAPI updated.

Use versioning:

```text id="1w4k0j"
/v1
```

for current public APIs.

---

# 91. Production configuration validation

Create a command such as:

```bash id="k80qfs"
./scripts/check-production-config.sh
```

It must fail if:

```text id="qrk0h1"
development bootstrap enabled
development passwords present
APP_ENV incorrect
unsafe CORS
missing JWT secret
missing database credentials
missing Stalwart management credential
```

---

# 92. Disaster simulation

Perform controlled failure tests:

```text id="j6omad"
kill API container
kill worker
restart PostgreSQL
restart Stalwart
remove a provisioning job from processing
cause a timeout
cause a temporary 5xx
```

Verify recovery.

---

# 93. Recovery objective

Define reasonable initial targets for:

```text id="g1e55n"
RTO = recovery time objective
RPO = recovery point objective
```

Do not claim enterprise guarantees yet.

Document what the current architecture can actually achieve.

---

# 94. Restore drill

Do at least one documented restore exercise.

Show:

```text id="wzx1tp"
backup
destroy
restore
start
migrate/check
health
regression tests
```

---

# 95. Cross-system disaster scenario

Simulate:

```text id="g0h5pt"
Norest PostgreSQL unavailable
```

and separately:

```text id="brj8ak"
Stalwart unavailable
```

Expected behavior must be documented.

---

# 96. Mail-data integrity

Never assume:

```text id="f06a9q"
Norest DB backup
```

contains customer mail.

Document the separate recovery responsibilities.

---

# 97. Million-user architecture review

At the end of the chapter, write a capacity document:

```text id="a1rr0x"
docs/scale-model.md
```

It must explain:

```text
API scaling
worker scaling
PostgreSQL scaling
Stalwart scaling
storage scaling
network scaling
connection limits
likely bottlenecks
```

Do not claim an exact user limit without evidence.

---

# 98. Scale bottleneck report

Identify the current bottleneck under load:

```text id="0ktgoe"
CPU
memory
PostgreSQL
Stalwart
network
worker throughput
```

Do not optimize everything simultaneously.

Optimize actual bottlenecks.

---

# 99. Chapter 5 load-test report

Create:

```text id="q5v4sl"
docs/load-test-report.md
```

Include:

```text
test environment
number of API instances
number of workers
database size
Stalwart configuration
concurrency
requests/sec
p50/p95/p99
error rate
DB CPU
DB connections
worker backlog
Stalwart latency
results
bottlenecks
```

---

# 100. Chapter 5 production architecture document

Create/update:

```text id="x12z5z"
docs/production-architecture.md
```

Document:

```text
internet
load balancer
Norest API
workers
PostgreSQL
Stalwart
storage
backups
secrets
monitoring
failure handling
```

---

# 101. Final folder additions

Preserve the existing repository structure.

Add only:

```text id="3v2b3v"
scripts/
├── check-production-config.sh
├── load-test-api.sh
├── load-test-provisioning.sh
├── backup-postgres.sh
├── restore-postgres.sh
├── reconcile.sh
└── test-failures.sh

docs/
├── production-architecture.md
├── scale-model.md
├── load-test-report.md
├── disaster-recovery.md
├── security.md
└── operations.md
```

Do not create unnecessary services.

---

# 102. Failure test suite

Create:

```text id="2u3krn"
scripts/test-failures.sh
```

It must exercise:

```text
API restart
worker restart
PostgreSQL restart
Stalwart restart
Stalwart timeout
duplicate job
stuck job
DNS failure
dependency recovery
```

---

# 103. Production test suite

Create:

```text id="c1f2l6"
scripts/test-production-readiness.sh
```

It must verify:

```text
configuration
health
readiness
database
Stalwart
authentication
security headers
rate limits
secrets
docker configuration
migration state
```

---

# 104. Complete regression suite

At the end of Chapter 5, the following must all pass:

```bash id="u74vsf"
go test ./...
go build ./...

./scripts/test-foundation.sh
./scripts/test-chapter2.sh
./scripts/verify-db-clean.sh

go run scripts/verify-chapter2/main.go
go run scripts/verify-chapter3/main.go

./scripts/test-chapter4-full.sh

./scripts/test-failures.sh
./scripts/test-production-readiness.sh
```

---

# 105. Security acceptance criteria

Chapter 5 is incomplete if any of these fail:

```text id="fe4q5x"
[ ] no committed secrets
[ ] no secrets in logs
[ ] no secrets in error responses
[ ] user cannot access another user's resources
[ ] user cannot access admin APIs
[ ] suspended users are handled correctly
[ ] admin authorization works
[ ] production bootstrap is disabled
[ ] unsafe CORS is rejected
[ ] malformed input is rejected
[ ] brute-force protection works
```

---

# 106. Reliability acceptance criteria

```text id="j57m8n"
[ ] API restart is safe
[ ] worker restart is safe
[ ] PostgreSQL restart is recoverable
[ ] Stalwart restart is recoverable
[ ] stuck jobs are recoverable
[ ] duplicate jobs are safe
[ ] Stalwart timeout is recoverable
[ ] DNS failures are retryable
[ ] remote operation ambiguity is handled
[ ] no DB transaction spans remote calls
```

---

# 107. Scalability acceptance criteria

```text id="9o9w3u"
[ ] multiple API instances work
[ ] multiple worker instances work
[ ] no process-local correctness state
[ ] concurrency tests pass
[ ] 1000+ API concurrent load tested
[ ] 1000+ provisioning jobs load tested
[ ] p95/p99 measured
[ ] DB connection usage measured
[ ] Stalwart latency measured
[ ] bottleneck documented
```

---

# 108. Recovery acceptance criteria

```text id="rcj1wh"
[ ] PostgreSQL backup succeeds
[ ] PostgreSQL restore succeeds
[ ] restored Norest starts
[ ] Stalwart data recovery documented
[ ] Norest/Stalwart reconciliation works
[ ] identifiers remain consistent
[ ] RTO documented
[ ] RPO documented
```

---

# 109. Operational acceptance criteria

```text id="8oqczw"
[ ] structured logging
[ ] request IDs
[ ] job IDs
[ ] health endpoints
[ ] readiness endpoint
[ ] metrics
[ ] provisioning backlog visibility
[ ] error categorization
[ ] production config validation
[ ] dependency version policy
```

---

# 110. Chapter 5 final architecture

The final architecture should remain:

```text id="k7jtvn"
                         INTERNET
                            |
                       LOAD BALANCER
                            |
               +------------+------------+
               |            |            |
             API-1        API-2        API-N
               |            |            |
               +------------+------------+
                            |
                       PostgreSQL
                            |
                    +-------+-------+
                    |               |
                 Worker-1        Worker-N
                    |               |
                    +-------+-------+
                            |
                        Stalwart
                            |
          +-----------------+-----------------+
          |                 |                 |
         JMAP              IMAP              SMTP
          |                 |                 |
       Webmail          Mail Clients       Internet
```

Supporting systems:

```text id="1vw7h7"
Backups
Secrets
Monitoring
Logging
Alerts
```

No extra distributed platform is required unless the measurements prove otherwise.

---

# 111. Absolute Chapter 5 rules

Do NOT:

```text id="t7xgji"
build new product features
build frontend
build message storage
build search
build a mail proxy
introduce Kafka
introduce Redis
introduce RabbitMQ
introduce Kubernetes
introduce Elasticsearch
introduce Temporal
rewrite Stalwart
replace PostgreSQL prematurely
shard PostgreSQL prematurely
claim millions of users without measurements
```

Do:

```text id="8i5dw9"
harden security
harden authentication
harden authorization
harden configuration
harden database usage
harden worker recovery
harden Stalwart integration
add readiness
add rate limiting
add observability
test backups
test recovery
load-test
measure bottlenecks
document capacity
```

---

# 112. Final implementation instruction to Devin

Work directly in the existing repository.

Do not redesign Chapters 1–4.

Do not add frontend work.

Do not add product features.

Do not start a new architecture.

Make the existing architecture production-safe.

First inspect the current code and identify weaknesses.

Then implement only the hardening/reliability work required by this chapter.

Run real failure tests.

Run real concurrent tests.

Run real load tests.

Run real backup/restore tests.

Do not replace measurements with theoretical claims.

At the end report:

```text id="l64p9u"
1. Security hardening completed
2. Secret-handling audit
3. Authentication audit
4. Authorization audit
5. Rate limiting
6. Database reliability
7. Worker reliability
8. Stalwart failure handling
9. Readiness/liveness
10. Backup/restore
11. Disaster recovery
12. Observability
13. API horizontal scaling
14. Worker horizontal scaling
15. API load-test results
16. Provisioning load-test results
17. Bottleneck analysis
18. RTO/RPO results
19. Complete regression results
20. Known limitations
```

Do not say:

> "Norest supports millions of users."

Instead report measured results and explain how horizontal scaling extends those results.

The success criterion is:

> **The Chapter 1–4 architecture remains simple, but it is now secure, observable, horizontally scalable, failure-tolerant, recoverable, and measurable enough to become a real production system.**
