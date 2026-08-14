# Norest Mail — Chapter 4

## Product Control Plane: Plans, Limits, Domain Verification, Quotas, Account Lifecycle & Billing Boundary

You are continuing Norest Mail after Chapter 3.

Chapter 1, Chapter 2, and Chapter 3 are complete and verified.

Treat them as the current stable foundation.

Do not redesign them.

Do not rebuild mail.

Do not create another mail engine.

Do not create a message database.

Do not introduce new infrastructure unless absolutely required by this chapter.

The purpose of Chapter 4 is to make Norest Mail a real **product/business control plane** around the already-proven Stalwart mail engine.

---

# 1. The architectural rule remains absolute

Norest owns the **product/business/control plane**.

Stalwart owns the **mail/data plane**.

## Norest owns

```text
users
product accounts
plans
subscriptions
domain ownership
domain verification state
email-address ownership
address policies
mailbox product status
quota policy
account lifecycle
suspension
billing state
billing provider references
product entitlements
audit
admin/product policy
```

## Stalwart owns

```text
accounts
mailboxes
folders
messages
attachments
threads
flags
read/unread
search
JMAP
IMAP
POP3
SMTP
mail queue
delivery
mail storage
actual mail data
mail-level quota enforcement
```

Do not duplicate mail state in PostgreSQL.

Stalwart already supports account and tenant quotas for storage and object counts, so Norest should define product policy and translate that policy into Stalwart configuration rather than creating its own mail-storage accounting engine.

---

# 2. Chapter 4 objective

At the end of Chapter 4, Norest must be able to answer:

```text
Who is this customer?
What plan do they have?
What are they allowed to do?
Which domains do they own?
Which domains are verified?
Which addresses do they own?
How many mailboxes can they create?
How much storage are they allowed?
Is the account active?
Is it suspended?
What product action is permitted?
What does Norest need to tell Stalwart?
```

The system must be capable of enforcing product rules **without becoming the mail system**.

---

# 3. End-state architecture

The target architecture becomes:

```text
                         NOREST
                           |
       +-------------------+-------------------+
       |                   |                   |
       v                   v                   v
   Identity             Product             Billing
       |                   |                   |
       +-------------------+-------------------+
                           |
                    Entitlements
                           |
                    Provisioning
                           |
                           v
                       STALWART
                           |
       +-------------------+-------------------+
       |                   |                   |
    Accounts            Mailboxes           Delivery
       |                   |                   |
     Messages            Folders             SMTP
     Threads             Attachments         Queue
     Search              JMAP                IMAP
```

Norest decides **what the customer is allowed to have**.

Stalwart enforces the mail-side consequences.

---

# 4. Do not build billing first

Billing is part of Chapter 4 only as a **boundary and subscription model**.

Do not integrate a real payment provider unless absolutely necessary to prove the product architecture.

Do not integrate:

```text
Stripe
Razorpay
Paddle
Adyen
PayPal
```

yet.

Instead build:

```text
plans
subscriptions
entitlements
billing provider references
```

with a provider-neutral design.

Real payment integration can come later.

---

# 5. Product account

Introduce a proper Norest product account concept if Chapter 2 currently equates `users` directly with ownership.

Recommended conceptual model:

```text
users
    |
    v
product_accounts
    |
    +---- domains
    +---- addresses
    +---- subscription
    +---- entitlements
```

This is important for future organization/business accounts.

A person and a billing/product account are not necessarily the same thing.

---

# 6. User vs product account

Do not assume permanently:

```text
1 user = 1 customer
```

Instead:

```text
User
  = login identity

Product Account
  = customer/business/product ownership boundary
```

Chapter 4 may initially provision:

```text
1 user → 1 product account
```

but the database and service layer must not make that assumption impossible to change later.

---

# 7. Product account states

Use:

```text
ACTIVE
SUSPENDED
DISABLED
```

Optional:

```text
PENDING
```

if necessary.

State transitions must be explicit.

Example:

```text
PENDING → ACTIVE
ACTIVE → SUSPENDED
ACTIVE → DISABLED
SUSPENDED → ACTIVE
SUSPENDED → DISABLED
```

Do not allow arbitrary state updates.

---

# 8. Plan model

Create:

```text
plans
```

with product-level fields.

At minimum:

```text
id
code
name
description
status
max_domains
max_mailboxes
max_storage_bytes
max_addresses
max_aliases
max_send_rate
features
created_at
updated_at
```

Do not encode every plan rule directly into handlers.

The plan is data.

---

# 9. Plan codes

Use immutable machine-readable codes:

```text
FREE
STARTER
PRO
BUSINESS
ENTERPRISE
```

These are examples only.

The application must not depend on display names.

For example:

```text
code = PRO
name = "Pro"
```

---

# 10. Entitlements

Do not create giant boolean logic such as:

```go
if user.Plan == "PRO" { ... }
```

throughout the application.

Create an entitlement/policy layer.

Example:

```go
CanCreateDomain(accountID)
CanCreateMailbox(accountID)
CanCreateAddress(accountID)
MaxStorage(accountID)
MaxMailboxes(accountID)
CanUseCustomDomain(accountID)
CanUseMultipleAddresses(accountID)
```

The service resolves these from:

```text
plan
subscription
account status
product policy
```

---

# 11. Entitlement priority

Define a deterministic precedence model.

Recommended:

```text
system policy
    ↓
plan entitlement
    ↓
account-specific override
    ↓
temporary promotional override
```

Do not allow random handlers to override limits.

---

# 12. Subscription model

Create:

```text
subscriptions
```

with approximately:

```text
id
product_account_id
plan_id
status
provider
provider_customer_id
provider_subscription_id
current_period_start
current_period_end
cancel_at_period_end
created_at
updated_at
```

Statuses:

```text
TRIALING
ACTIVE
PAST_DUE
CANCELED
EXPIRED
PAUSED
```

Keep the provider generic.

---

# 13. Billing provider boundary

Create:

```text
internal/billing/
```

with a provider abstraction.

Conceptually:

```go
type BillingProvider interface {
    CreateCustomer(...)
    CreateSubscription(...)
    CancelSubscription(...)
}
```

Do not create a real provider implementation unless required.

A development/mock provider may exist only for product testing and must never be confused with production billing.

---

# 14. Webhook boundary

Define:

```http
POST /v1/webhooks/billing
```

but do not connect it to a specific vendor yet unless necessary.

Requirements:

* verify authenticity when a real provider exists
* idempotently process events
* never trust arbitrary client-supplied subscription status
* record provider event ID
* reject duplicate events safely

---

# 15. Billing event table

Create a product-level table such as:

```text
billing_events
```

with:

```text
id
provider
provider_event_id
event_type
payload_hash
status
processed_at
created_at
```

Unique constraint:

```text
(provider, provider_event_id)
```

This prevents duplicate webhook processing.

Do not store unnecessary payment card information.

---

# 16. Product limits are NOT mail storage

This distinction is critical.

Norest might say:

```text
Pro plan:
50 GB storage
20 mailboxes
100 addresses
```

That means:

> "This customer is entitled to use up to these amounts."

Stalwart is responsible for actually enforcing mail storage/object limits.

Stalwart's account quotas support disk and object limits, including message/mailbox/submission/identity-related caps.

---

# 17. Quota mapping

Create a quota translation service:

```text
Norest entitlement
        ↓
Quota policy
        ↓
Stalwart Account quotas
```

Example:

```text
Norest:
max_storage = 50 GB

        ↓

Stalwart:
quotas.maxDiskQuota = 53687091200
```

Do not maintain a separate "used mail bytes" counter in Norest.

---

# 18. Quota ownership

Norest owns:

```text
allowed limit
```

Stalwart owns:

```text
actual usage
```

The product can obtain current usage from Stalwart when needed.

Do not continuously copy usage into PostgreSQL unless there is a clearly defined product/reporting reason.

---

# 19. Quota synchronization

When a plan changes:

```text
FREE
  ↓
PRO
```

Norest:

1. changes subscription/entitlement
2. creates quota synchronization job
3. worker updates Stalwart account quota
4. records synchronization result

Do not synchronously wait for Stalwart in the HTTP request.

---

# 20. Quota downgrade

A downgrade may produce:

```text
current_usage > new_limit
```

Example:

```text
current = 40 GB
new plan = 10 GB
```

Do NOT delete mail.

Do NOT automatically delete messages.

Instead:

```text
account remains intact
product status = QUOTA_EXCEEDED
```

and define a product policy such as:

```text
no new storage-consuming mail
cannot create additional addresses
cannot upload additional content
```

The exact enforcement mechanism should use Stalwart where possible.

---

# 21. Never delete mail because of billing

This is a hard rule.

Subscription cancellation or downgrade must not immediately destroy:

```text
messages
folders
attachments
mailbox
```

unless a future explicit retention/deletion policy has been designed and communicated.

---

# 22. Domain verification

Chapter 4 makes custom domain verification a real product capability.

Domain states should now be:

```text
PENDING
VERIFYING
ACTIVE
SUSPENDED
DISABLED
```

The meaning:

```text
PENDING
domain record created

VERIFYING
verification has been initiated

ACTIVE
domain ownership is proven and usable

SUSPENDED
temporarily unavailable

DISABLED
permanently/internally disabled
```

---

# 23. Verification strategy

For Chapter 4, implement DNS-based ownership verification.

Do not depend on manual admin approval.

Use a deterministic verification record.

Recommended concept:

```text
_norest-verification.example.com
```

TXT record:

```text
norest-verification=<random-token>
```

The exact prefix can be chosen carefully.

---

# 24. Verification token

Generate a cryptographically random token.

Store:

```text
verification_token_hash
```

rather than unnecessarily storing a reusable plaintext secret.

Do not place credentials in URLs.

Do not use predictable tokens.

---

# 25. DNS verification API

Implement:

```http
POST /v1/domains/:domainID/verification/start
GET  /v1/domains/:domainID/verification
POST /v1/domains/:domainID/verification/check
```

The client should be able to retrieve instructions such as:

```json
{
  "type": "TXT",
  "name": "_norest-verification.example.com",
  "value": "..."
}
```

Do not claim the domain is verified until the DNS record is actually observed.

---

# 26. DNS checker

Create a small internal DNS verification module.

It must support:

```text
TXT lookup
```

with proper:

```text
timeouts
errors
multiple answers
DNS propagation delays
```

Do not run a DNS server.

Do not cache DNS forever.

---

# 27. DNS verification retries

Verification may initially fail because of propagation.

Use a retry model:

```text
PENDING
 ↓
VERIFYING
 ↓
check DNS
 ↓
not found
 ↓
RETRY
```

Do not mark the domain FAILED permanently after one DNS lookup.

---

# 28. Verification success

When the correct token is found:

```text
verification_status = VERIFIED
status = ACTIVE
```

Then:

```text
DOMAIN_CREATE / DOMAIN_ACTIVATE
```

or the appropriate Stalwart synchronization operation must be triggered.

---

# 29. Domain already provisioned

A domain may already exist in Stalwart from a previous attempt.

Provisioning must remain idempotent.

Never create duplicate Stalwart domains because of verification retries.

---

# 30. Domain deletion

A verified domain cannot be deleted casually.

Require:

```text
no active addresses
no active mailboxes
```

or a defined product-level deletion workflow.

For Chapter 4, prefer a two-step process:

```text
ACTIVE
 ↓
DISABLED
 ↓
eventual deletion
```

Do not immediately destroy customer mail.

---

# 31. Platform-owned domains

Chapter 4 must introduce the distinction between:

```text
PLATFORM
CUSTOMER
```

domain ownership.

Examples:

```text
norest.com
PLATFORM

customer-example.com
CUSTOMER
```

Norest should be able to provision platform domains without asking a customer to prove ownership.

---

# 32. Platform domain table

Add fields such as:

```text
ownership_type
```

or an equivalent model.

The product must be able to answer:

```text
Is this domain owned by Norest?
Is this domain owned by a customer?
```

---

# 33. Platform address reservation

Platform domains must support:

```text
reserved local parts
system addresses
service addresses
```

Examples:

```text
postmaster
abuse
support
security
admin
noreply
```

The actual list must remain centrally managed.

---

# 34. Address policy

Chapter 4 should make address policy a proper product service.

Create:

```text
internal/policy/
```

or an equivalent policy package.

It should answer:

```text
Can this address be created?
Can it be reused?
Is it reserved?
Is it blocked?
Does this plan permit it?
```

Do not put policy decisions into controllers.

---

# 35. Address lifecycle

Use:

```text
PENDING
ACTIVE
SUSPENDED
DISABLED
```

and later:

```text
DELETED
```

only when a real deletion mechanism exists.

---

# 36. Account suspension

Implement Norest account suspension.

When:

```text
product_account.status = SUSPENDED
```

Norest must:

```text
prevent new product operations
prevent new provisioning
schedule mail-account restriction
```

It must NOT directly manipulate messages.

---

# 37. Stalwart suspension mapping

Use the Stalwart account lifecycle to disable user access where appropriate.

The exact Stalwart field/method must be confirmed from the current schema before implementation.

Do not guess the field.

Implement through:

```text
internal/stalwart/
```

only.

---

# 38. Suspension must be idempotent

Calling suspend twice must produce the same final state.

```text
SUSPEND
SUSPEND
SUSPEND
```

must not create multiple jobs or corrupt state.

---

# 39. Reactivation

Implement:

```http
POST /v1/account/reactivate
```

or an appropriate account endpoint.

Reactivation must:

1. validate subscription/product status
2. update Norest
3. provision a Stalwart re-enable job
4. mark active only when appropriate

---

# 40. Mailbox disabling

When an address is disabled:

```text
Norest address = DISABLED
```

and a provisioning job should update the associated Stalwart account.

Do not delete the mailbox.

---

# 41. Product vs mail status

Do not collapse every state into one field.

For example:

```text
Norest mailbox:
ACTIVE

Stalwart account:
disabled

```

is a reconciliation issue.

Model:

```text
product_status
provisioning_status
```

or an equivalent separation where necessary.

---

# 42. Provisioning remains asynchronous

Every control-plane → Stalwart mutation remains:

```text
HTTP request
   ↓
PostgreSQL transaction
   ↓
job
   ↓
worker
   ↓
Stalwart
```

Do not put remote Stalwart calls inside long-running DB transactions.

---

# 43. New Chapter 4 job types

Add only necessary job types:

```text
DOMAIN_VERIFY
DOMAIN_ACTIVATE
DOMAIN_SUSPEND
DOMAIN_DISABLE

ACCOUNT_QUOTA_SYNC
ACCOUNT_SUSPEND
ACCOUNT_REACTIVATE

ACCOUNT_PLAN_SYNC
```

Do not create dozens of tiny job types.

---

# 44. Job idempotency

Every new job must be safe to retry.

Especially:

```text
quota sync
suspend
reactivate
domain activate
```

Use the existing `FOR UPDATE SKIP LOCKED` worker architecture.

Do not replace it.

---

# 45. Plan-change workflow

The plan change flow:

```text
User/account changes plan
        ↓
Norest validates entitlement
        ↓
subscription updated
        ↓
entitlements recalculated
        ↓
control-plane audit event
        ↓
Stalwart synchronization job
        ↓
worker
        ↓
quota/policy update
```

---

# 46. Subscription is authoritative for product entitlement

Do not infer plan solely from Stalwart.

Norest PostgreSQL is authoritative for:

```text
plan
subscription
entitlement
```

Stalwart is authoritative for:

```text
mail enforcement state
```

---

# 47. Entitlement snapshot

For performance and auditability, it is acceptable to maintain a calculated entitlement snapshot.

Example:

```text
account_entitlements
```

Fields:

```text
product_account_id
max_domains
max_mailboxes
max_addresses
max_storage_bytes
features
version
updated_at
```

The source of truth remains:

```text
plan + subscription + product policy
```

The snapshot is a derived projection.

---

# 48. Do not event-source the entire system

Do NOT make every plan change an event-sourcing system.

Use normal PostgreSQL transactions plus:

```text
audit log
provisioning job
derived entitlement state
```

That is enough.

---

# 49. Product limits

Implement limits for:

```text
domains
mailboxes
addresses
storage
```

Only.

Do not implement every possible future resource limit.

---

# 50. Limit enforcement

Before creation:

```text
count current
compare entitlement
reserve/create atomically
```

Do not perform:

```text
SELECT count
if count < limit
INSERT
```

without concurrency protection.

Use database constraints/transactional logic to avoid race conditions.

---

# 51. Concurrent mailbox creation

Example:

```text
limit = 5
current = 4
```

Two simultaneous requests must not create:

```text
mailbox 5
mailbox 6
```

The product must enforce the limit safely.

Use PostgreSQL locking or transactional reservation logic.

Do not rely solely on application memory.

---

# 52. Domain limit

If:

```text
max_domains = 3
```

then a fourth domain request must fail with:

```text
409 Conflict
```

or an appropriate domain-specific error.

Do not create the fourth domain and clean it up asynchronously.

---

# 53. Mailbox limit

Similarly:

```text
max_mailboxes = 10
```

means the eleventh address that creates a new mailbox must be rejected before the product resource is committed.

Do not count pending/active ambiguously.

Define the policy.

Recommended:

```text
PENDING + ACTIVE + PROVISIONING
```

count toward the limit.

This prevents users from racing provisioning jobs.

---

# 54. Address limit

Decide whether aliases count.

Define clearly:

```text
mailbox address
alias address
```

as separate concepts.

Chapter 4 only needs the primary mailbox-address limit.

Do not build a full alias system yet.

---

# 55. Alias boundary

Do not create a full alias product in Chapter 4.

However, define the model so that:

```text
primary mailbox
aliases
```

can exist later.

Stalwart already supports aliases as a domain/account concept. Domain aliases and address aliases can be managed in the mail plane.

Norest should eventually own product authorization for those aliases.

---

# 56. Usage reporting

Implement a minimal read-only usage endpoint:

```http
GET /v1/account/usage
```

Return product-level usage:

```json
{
  "domains": {
    "used": 2,
    "limit": 5
  },
  "mailboxes": {
    "used": 7,
    "limit": 10
  },
  "addresses": {
    "used": 9,
    "limit": 25
  }
}
```

For storage:

```text
used
limit
```

may be fetched from Stalwart when appropriate.

Do not maintain an independent storage counter.

---

# 57. Usage consistency

Usage counts must be defined.

Example:

```text
mailboxes.used
```

should count only states designated by product policy.

Document:

```text
ACTIVE
PENDING
SUSPENDED
DISABLED
```

in relation to usage calculations.

---

# 58. Stalwart quota synchronization

Create:

```text
internal/provisioning/
    quota.go
```

or an equivalent clean module.

Do not put quota logic inside billing handlers.

Flow:

```text
Entitlement
   ↓
QuotaPolicy
   ↓
Stalwart Account update
```

---

# 59. Stalwart quota keys

Use only documented/current quota fields.

For storage and mail object counts, Stalwart currently supports account-level quotas such as `maxDiskQuota`, `maxEmails`, `maxMailboxes`, `maxEmailIdentities`, `maxEmailSubmissions`, and other object limits.

Do not invent keys.

Before coding any quota field, inspect the actual v0.16 schema/API available in the running instance.

---

# 60. No tenant architecture yet

Do NOT make each Norest customer a Stalwart tenant in Chapter 4 unless there is a demonstrated product requirement.

Stalwart supports tenants and tenant-level quotas, but introducing them now would bind the architecture to a more complex mail-isolation model unnecessarily.

For Chapter 4:

```text
Norest product account
    ↓
Stalwart account/domain
```

is sufficient.

Keep the abstraction ready for future tenant mapping.

---

# 61. Domain verification and Stalwart

Important:

Norest domain verification means:

```text
customer proved ownership
```

Stalwart domain existence means:

```text
mail server recognizes domain
```

These are different states.

Do not use:

```text
Stalwart domain exists
```

as proof of ownership.

---

# 62. Verified domain activation

Only after Norest verification:

```text
verification_status = VERIFIED
```

should the domain become product-active.

Then sync the corresponding Stalwart domain.

---

# 63. Domain DNS information

The domain API should eventually provide DNS requirements.

For Chapter 4, create:

```http
GET /v1/domains/:domainID/dns
```

Return product-required records:

```text
verification
MX
SPF
DKIM
DMARC
```

But do not build automated DNS management.

---

# 64. DNS record ownership

Separate:

```text
records Norest requires
```

from:

```text
records the user has actually published
```

Do not mark them as active merely because they are generated.

---

# 65. Mail readiness

Add a domain readiness concept.

Example:

```text
verification
mail
security
```

A domain may be:

```text
ownership verified
mail not fully configured
```

Do not collapse all DNS requirements into a single boolean.

---

# 66. MX records

Norest may display expected MX records.

Do not make Norest receive SMTP.

Stalwart remains the actual mail receiver.

---

# 67. SPF/DKIM/DMARC

Chapter 4 may provide configuration guidance.

Do not build an external DNS writer.

Do not create a second DKIM implementation.

Stalwart remains responsible for mail signing/authentication behavior.

A domain is also an anchor for DKIM/TLS/mail handling inside Stalwart.

---

# 68. Admin product APIs

Create a minimal internal/admin API boundary.

Do not build the full admin UI.

Possible endpoints:

```http
GET /v1/admin/accounts
GET /v1/admin/accounts/:id
POST /v1/admin/accounts/:id/suspend
POST /v1/admin/accounts/:id/reactivate

GET /v1/admin/domains
GET /v1/admin/provisioning/jobs
GET /v1/admin/audit
```

These must be protected by admin roles.

---

# 69. Admin roles

Do not create a complicated RBAC platform.

Start with:

```text
USER
ADMIN
```

Optionally:

```text
SUPERADMIN
```

only if necessary.

Keep authorization centralized.

---

# 70. Admin safety

Admin APIs must never expose:

```text
passwords
mailbox credentials
message bodies
attachments
```

unless a future explicit support/debug system is designed.

Admin access to product state is not the same as access to customer mail contents.

---

# 71. Audit logging

Expand Chapter 2 audit logging.

Record:

```text
PLAN_CHANGED
SUBSCRIPTION_CHANGED
DOMAIN_VERIFICATION_STARTED
DOMAIN_VERIFIED
DOMAIN_SUSPENDED
DOMAIN_DISABLED
ADDRESS_SUSPENDED
ACCOUNT_SUSPENDED
ACCOUNT_REACTIVATED
QUOTA_CHANGED
ADMIN_ACTION
```

Every sensitive product mutation must be auditable.

---

# 72. Audit data

Never record:

```text
passwords
tokens
payment secrets
full message bodies
attachment contents
```

Metadata only.

---

# 73. Audit immutability

Audit logs should be append-only.

Normal application code must not update/delete audit rows.

Provide retention later.

---

# 74. Account deletion

Do NOT implement irreversible account deletion in Chapter 4.

Instead implement:

```text
DISABLED
```

and document future retention/deletion policy.

Deletion of customer mail is a high-impact product/legal decision.

---

# 75. Suspension semantics

Define exactly what suspension means.

Recommended:

```text
No new domains
No new addresses
No new mailboxes
No new product operations
Existing mailbox may be disabled for mail login/submission
Existing mail data remains intact
```

The exact Stalwart enforcement should be implemented using account state/permissions as supported by the current API.

---

# 76. Plan downgrade semantics

Downgrade should not destroy mail.

Possible result:

```text
subscription = ACTIVE
plan = FREE
quota_status = EXCEEDED
```

New storage-generating actions may be blocked by Stalwart quota.

---

# 77. Past-due semantics

If a future billing integration reports:

```text
PAST_DUE
```

do not immediately suspend.

Define a grace-period product policy.

Chapter 4 should model the state but not implement a sophisticated collections engine.

---

# 78. Entitlement recalculation

Create one canonical function:

```go
ResolveEntitlements(accountID)
```

It must be deterministic.

Input:

```text
product account
subscription
plan
overrides
status
```

Output:

```text
Entitlements
```

Every API/service requiring a limit calls this layer.

---

# 79. Feature flags vs entitlements

Do not confuse:

```text
feature flag
```

with:

```text
plan entitlement
```

Feature flags control deployment/testing.

Entitlements control customer product rights.

Keep them separate.

---

# 80. Product policy configuration

Do not hardcode every reserved address and global limit into source.

Use configuration or database-backed policy where appropriate.

Keep it simple.

Do not create a dynamic rules engine.

---

# 81. Scalability

Chapter 4 must remain compatible with millions of users.

Requirements:

```text
stateless API
PostgreSQL authority
transactional limits
indexed usage queries
async provisioning
idempotent jobs
horizontal workers
```

Do not use in-memory counts as authoritative limits.

---

# 82. Database indexes

Add appropriate indexes for:

```text
product_accounts.user_id
domains.product_account_id
addresses.domain_id
mailboxes.address_id
subscriptions.product_account_id
audit_logs.product_account_id
provisioning_jobs.status,next_attempt_at
billing_events.provider,provider_event_id
```

Do not add random indexes without query justification.

---

# 83. Database uniqueness

Enforce:

```text
plan.code
subscription provider IDs
billing event IDs
domain canonical name
address canonical local part + domain
mailbox address
```

where appropriate.

---

# 84. Race-condition testing

Test concurrently:

```text
creating domains near limit
creating mailboxes near limit
changing plan while provisioning
suspending while provisioning
verifying domain while deleting/disable is requested
```

The final state must be deterministic.

---

# 85. Plan change during provisioning

Example:

```text
FREE
  ↓
create mailbox
  ↓
provisioning pending
  ↓
upgrade PRO
```

The job must use the current authoritative product state at execution time.

Do not blindly execute stale quota values from the original job payload.

---

# 86. Suspension during provisioning

Example:

```text
address provisioning pending
        ↓
account suspended
        ↓
worker gets job
```

The worker must re-check account/domain/address state before acting.

Do not provision a mailbox for a disabled account.

---

# 87. Verification during suspension

Similarly:

```text
domain verifying
    ↓
account suspended
```

should prevent activation until the product account is permitted to use the domain.

---

# 88. No hidden side effects

API operations should be predictable.

For example:

```text
POST /domains
```

should create the Norest domain and provisioning job.

It should not silently:

```text
change subscription
modify unrelated domains
alter user password
```

---

# 89. API endpoints

Implement the following Chapter 4 public API.

## Plans

```http
GET /v1/plans
GET /v1/account/plan
```

## Subscription

```http
GET /v1/account/subscription
POST /v1/account/subscription
POST /v1/account/subscription/cancel
```

Provider implementation may remain development-only.

## Usage

```http
GET /v1/account/usage
GET /v1/account/entitlements
```

## Domain verification

```http
POST /v1/domains/:id/verification/start
GET  /v1/domains/:id/verification
POST /v1/domains/:id/verification/check
GET  /v1/domains/:id/dns
```

## Account lifecycle

```http
POST /v1/account/suspend
POST /v1/account/reactivate
```

## Admin

```http
GET  /v1/admin/accounts
GET  /v1/admin/accounts/:id
POST /v1/admin/accounts/:id/suspend
POST /v1/admin/accounts/:id/reactivate
GET  /v1/admin/provisioning/jobs
GET  /v1/admin/audit
```

Do not build all admin functionality beyond these basics.

---

# 90. API authorization

Public user endpoints require authenticated Norest identity.

Admin endpoints require:

```text
role = ADMIN
```

Never trust role information supplied by the browser.

---

# 91. HTTP error semantics

Use consistent errors:

```text
PLAN_LIMIT_REACHED
DOMAIN_NOT_VERIFIED
ACCOUNT_SUSPENDED
SUBSCRIPTION_REQUIRED
QUOTA_EXCEEDED
RESOURCE_CONFLICT
PROVISIONING_IN_PROGRESS
```

Do not expose internal Stalwart errors directly.

---

# 92. Product API vs Stalwart API

Never expose:

```text
x:Account/set
x:Domain/set
x:Account/get
```

directly as public Norest endpoints.

The public Norest API describes product intent.

The internal Stalwart client translates product state into mail-engine operations.

---

# 93. Quota failure

If Stalwart rejects a quota update:

```text
Norest product state must not falsely say "synchronized".
```

Record:

```text
sync pending/failed
```

and retry.

Do not silently swallow the error.

---

# 94. Product status vs sync status

For domains, accounts, quotas, and plans, consider:

```text
product_state
sync_state
```

separately.

Example:

```text
plan = PRO
quota_sync = PENDING
```

This makes operational failures visible without corrupting business state.

---

# 95. Provisioning reconciliation

Add a lightweight reconciliation command:

```bash
scripts/reconcile-chapter4.sh
```

It should inspect:

```text
Norest domain ↔ Stalwart domain
Norest mailbox ↔ Stalwart account
Norest entitlement ↔ Stalwart quota
```

Do not make it a continuous distributed scheduler.

It can initially be a manually executable repair/verification tool.

---

# 96. Stalwart source of truth rules

Never infer:

```text
customer subscription
```

from Stalwart.

Never infer:

```textcustomer ownership
```

from Stalwart.

Never infer:

```textbilling state
```

from Stalwart.

Stalwart is mail authority only.

---

# 97. Norest source of truth rules

Never infer:

```textmessage existence
folder state
read state
mail body
attachment data
mail search results
```

from Norest PostgreSQL.

Use Stalwart.

---

# 98. External mail clients

Chapter 4 must not change external-client architecture.

The eventual model remains:

```text
Outlook
Thunderbird
Apple Mail
other clients
        |
        v
     Stalwart
   SMTP/IMAP/JMAP
```

Norest controls the product account and provisioning.

---

# 99. No frontend

This chapter is backend/control-plane only.

Do not build:

```text
pricing UI
billing UI
domain verification UI
admin dashboard
mail UI
```

Those will be built later.

You may make the APIs testable with curl/Go scripts.

---

# 100. No frontend assumptions

Do not change frontend code.

Do not require frontend changes to prove Chapter 4.

All acceptance tests must be executable from:

```text
curl
Go integration tests
shell scripts
```

---

# 101. Chapter 4 database additions

Expected new tables may include:

```text
product_accounts
plans
subscriptions
account_entitlements
billing_events
```

Potentially:

```text
domain_verifications
```

if the existing domains table should not carry all verification state.

Potentially:

```text
account_product_state
```

if necessary.

Do not create separate storage for mail.

---

# 102. Domain verification storage

Store:

```text
domain_id
token_hash
status
started_at
verified_at
last_checked_at
last_error
```

Do not store DNS answers indefinitely.

---

# 103. Plan schema principle

Do not store arbitrary JSON for the entire product plan if simple columns are sufficient.

Use structured columns for core limits:

```text
max_domains
max_mailboxes
max_addresses
max_storage_bytes
```

Use JSON only for genuinely flexible feature flags.

---

# 104. Entitlement versioning

Give entitlements a version/revision.

Example:

```text
entitlements.version = 7
```

When a plan changes:

```text
version increments
```

This makes quota synchronization and audit easier.

---

# 105. Quota synchronization idempotency

A quota synchronization job should be safely repeatable.

Example:

```text
set maxDiskQuota = 50GB
```

Calling it three times should yield the same state.

Do not use relative mutations.

---

# 106. No usage event stream

Do not create:

```text
mail_usage_events
quota_events
message_count_stream
```

in Chapter 4.

Query Stalwart when product usage must be shown or reconciled.

---

# 107. Rate limits

Norest should define product API rate limits separately from Stalwart protocol limits.

Stalwart already has configurable HTTP and mail/JMAP rate limiting.

Do not attempt to duplicate all Stalwart rate limiting in Norest.

Norest protects:

```text
registration
login
domain creation
address creation
verification
billing
admin
```

Stalwart protects:

```text
JMAP
IMAP
SMTP
```

---

# 108. Login/account abuse

Add reasonable protection for:

```text
register
login
domain verification checks
```

Do not add Redis.

A simple database-backed or process-level development mechanism is sufficient until the production distributed rate-limiter chapter.

---

# 109. Plan seed data

Create deterministic development plans:

```text
FREE
PRO
BUSINESS
```

with realistic but clearly development/test limits.

Do not hardcode production pricing.

---

# 110. Test plan

Create:

```text
scripts/test-chapter4.sh
scripts/verify-chapter4/
```

Tests must cover:

```text
registration
product account creation
plan retrieval
entitlement calculation
domain limits
mailbox limits
address limits
domain verification
plan change
quota synchronization
account suspension
reactivation
admin authorization
audit
billing event idempotency
```

---

# 111. Domain verification E2E

Test:

```text
create domain
        ↓
verification PENDING
        ↓
start verification
        ↓
simulate DNS test environment
        ↓
verification succeeds
        ↓
domain ACTIVE
        ↓
Stalwart domain synchronized
```

The test environment may use a deterministic test DNS mechanism.

Do not fake production logic.

---

# 112. Plan-limit E2E

Example:

```text
plan.max_domains = 2
```

Create:

```text
domain 1
domain 2
```

third attempt:

```text
409 PLAN_LIMIT_REACHED
```

---

# 113. Mailbox-limit E2E

Example:

```text
max_mailboxes = 2
```

Create:

```text
alice
bob
```

third mailbox:

```text
409 PLAN_LIMIT_REACHED
```

No extra provisioning job may be committed.

---

# 114. Concurrency limit test

Run:

```text
100 concurrent mailbox creation requests
```

with:

```text
max_mailboxes = 5
```

Expected:

```text
exactly 5 accepted
remaining requests rejected
```

No sixteenth/extra mailbox.

---

# 115. Suspension E2E

Test:

```text
ACTIVE account
    ↓
SUSPEND
    ↓
new domain creation rejected
    ↓
new address creation rejected
    ↓
mail-access provisioning rejected
    ↓
Stalwart account restriction job processed
```

Then:

```text
REACTIVATE
    ↓
product active
    ↓
Stalwart re-enabled
```

---

# 116. Quota E2E

Test:

```text
FREE = 1GB
PRO = 10GB
```

Upgrade:

```text
FREE → PRO
```

Verify Stalwart account quota changes.

Then downgrade:

```text
PRO → FREE
```

Verify the lower quota is synchronized.

Do not delete mail if usage exceeds the new quota.

---

# 117. Billing event idempotency test

Send the same event twice:

```text
provider_event_id = evt_123
```

Expected:

```text
first = processed
second = ignored/already processed
```

No duplicate subscription or entitlement mutation.

---

# 118. Audit E2E

Verify each important operation produces audit entries.

At minimum:

```text
USER_REGISTERED
PLAN_CHANGED
DOMAIN_CREATED
DOMAIN_VERIFICATION_STARTED
DOMAIN_VERIFIED
ADDRESS_CREATED
ACCOUNT_SUSPENDED
ACCOUNT_REACTIVATED
QUOTA_CHANGED
```

---

# 119. Admin authorization test

Verify:

```text
USER token → admin endpoint = 403
ADMIN token → admin endpoint = allowed
```

Do not let resource IDs override role checks.

---

# 120. Data isolation regression

Run:

```bash
./scripts/verify-db-clean.sh
```

after every Chapter 4 E2E suite.

It must continue to prove:

```text
no messages
no folders
no threads
no attachments
no mail bodies
```

in PostgreSQL.

---

# 121. Chapter 1 regression

Run:

```bash
./scripts/test-foundation.sh
```

must still pass.

---

# 122. Chapter 2 regression

Run:

```bash
./scripts/test-chapter2.sh
go run scripts/verify-chapter2/main.go
```

must still pass.

---

# 123. Chapter 3 regression

Run:

```bash
go run scripts/verify-chapter3/main.go
```

must still pass.

Product control-plane changes must not break JMAP mail access.

---

# 124. Architecture regression

After Chapter 4:

```text
Norest
   ↓
PostgreSQL
   ↓
product state

Norest Worker
   ↓
Stalwart management JMAP
   ↓
mail-engine state
```

Still no mail proxy.

Still no mail database.

---

# 125. Documentation

Create/update:

```text
docs/product-model.md
docs/entitlements.md
docs/domain-verification.md
docs/quotas.md
docs/billing-boundary.md
docs/account-lifecycle.md
```

Keep each document concise and authoritative.

---

# 126. Product model documentation

Document:

```text
User
Product Account
Plan
Subscription
Entitlement
Domain
Address
Mailbox
```

and how they relate.

---

# 127. Entitlement documentation

Document:

```text
how plan limits are resolved
how account status affects entitlement
how overrides work
how quota sync works
```

---

# 128. Domain verification documentation

Document:

```text
verification token
DNS record
verification lifecycle
activation
failure
retry
```

---

# 129. Quota documentation

Document:

```text
Norest limit
Stalwart limit
usage authority
synchronization
downgrade behavior
```

---

# 130. Billing boundary documentation

Document:

```text
Norest owns subscription state.
External billing provider owns payment transactions.
Stalwart does not know payment details.
```

---

# 131. Account lifecycle documentation

Document:

```text
ACTIVE
SUSPENDED
DISABLED
```

and exact effects on:

```text
domain creation
address creation
mailbox provisioning
mail access
```

---

# 132. Folder structure

Preserve previous chapters.

Add:

```text
internal/
├── accounts/
│   ├── service.go
│   ├── repository.go
│   └── models.go
│
├── plans/
│   ├── service.go
│   ├── repository.go
│   └── models.go
│
├── subscriptions/
│   ├── service.go
│   ├── repository.go
│   └── models.go
│
├── entitlements/
│   ├── service.go
│   └── models.go
│
├── billing/
│   ├── service.go
│   ├── provider.go
│   └── webhook.go
│
├── verification/
│   ├── service.go
│   ├── dns.go
│   └── models.go
│
├── policy/
│   ├── service.go
│   └── reserved.go
│
├── audit/
│
└── provisioning/
    └── ...
```

Combine modules when existing code already provides the same responsibility.

Do not create folders just to make the tree larger.

---

# 133. Database migrations

Do not edit existing applied migrations.

Create new append-only migrations.

Example:

```text
006_product_accounts.sql
007_plans.sql
008_subscriptions.sql
009_entitlements.sql
010_domain_verification.sql
011_billing_events.sql
012_audit_extensions.sql
```

Actual names may differ.

---

# 134. API OpenAPI

Update:

```text
api/openapi.yaml
```

with all Chapter 4 public APIs.

Document authentication requirements.

Document response schemas.

Document error codes.

Do not document internal Stalwart management operations.

---

# 135. Configuration

Add only necessary environment variables.

Examples:

```text
DOMAIN_VERIFICATION_TTL
PROVISIONING_WORKERS
```

Billing provider settings should remain absent or development-only until a real integration exists.

Do not put plan limits entirely in environment variables.

Plans belong in product data.

---

# 136. No hardcoded business logic

Do not do:

```go
if plan == "PRO" {
    maxMailboxes = 20
}
```

Use plan records.

---

# 137. No hardcoded billing provider

Do not do:

```go
if provider == "stripe" { ... }
```

through product logic.

Keep provider-specific code behind the billing boundary.

---

# 138. Security rules

Never log:

```text
passwords
JWTs
refresh tokens
mail credentials
verification secrets
billing secrets
API keys
```

Hash verification tokens where possible.

Protect admin APIs.

Do not expose internal Stalwart credentials.

---

# 139. Product-account ownership

Every domain/address/subscription must belong to the authenticated product account.

Never trust:

```text
account_id
user_id
domain_id
```

from arbitrary client input without checking ownership.

---

# 140. Transaction rules

These operations must be transactional:

```text
create product account
change subscription + update entitlement version
domain creation + verification state
mailbox reservation + plan limit
billing event processing
suspension + audit
```

Do not hold DB transactions during DNS or Stalwart network calls.

---

# 141. Remote-operation rule

Never:

```text
BEGIN
   DNS lookup
   Stalwart API
   billing API
COMMIT
```

Instead:

```text
DB transaction
   ↓
commit product state/job
   ↓
remote system
   ↓
result transaction
```

---

# 142. Reconciliation

The control plane must tolerate remote failure.

Example:

```text
Norest:
quota = 50GB

Stalwart:
quota = 10GB
```

This is a sync failure, not a reason to corrupt product state.

Create a job and reconcile.

---

# 143. Retry strategy

Use the Chapter 2 worker.

Retry temporary failures.

Do not retry:

```text
invalid plan
invalid domain
permission permanently denied
malformed request
```

indefinitely.

---

# 144. Admin repair

Provide a minimal way for an administrator to requeue a failed provisioning/sync job.

Example:

```http
POST /v1/admin/provisioning/jobs/:id/retry
```

Do not build a full workflow UI.

---

# 145. Product event audit vs provisioning log

Keep them separate.

```text
audit_logs
    = what the customer/admin did

provisioning_jobs
    = what the system needs to synchronize

billing_events
    = what the billing provider told us
```

Do not combine all three.

---

# 146. No event-driven overengineering

Do not introduce Kafka.

PostgreSQL remains enough for:

```text
transactions
jobs
billing events
audit
entitlements
```

The architecture can later evolve if actual scale demands it.

---

# 147. Million-user requirement

Chapter 4 must remain horizontally scalable.

Norest API instances must be stateless.

Workers can scale horizontally.

PostgreSQL remains the control-plane authority.

Stalwart remains independent.

Do not introduce per-user in-memory state.

---

# 148. Million-account database design

Use:

```text
UUIDs
indexed ownership columns
canonical normalized values
database constraints
pagination
efficient counts
```

Do not use unbounded JSON blobs as the primary product model.

---

# 149. API pagination

Admin list endpoints must support:

```text
limit
cursor
```

or equivalent.

Do not return millions of accounts in one response.

---

# 150. Product usage pagination

Usage is summary data and does not require pagination.

Admin resource listings do.

---

# 151. Chapter 4 acceptance criteria

Chapter 4 is COMPLETE only when all are true:

```text
[ ] Product account model exists.

[ ] User/product-account relationship is defined.

[ ] Plans exist.

[ ] Plans are data-driven.

[ ] Subscription model exists.

[ ] Entitlements are resolved centrally.

[ ] Domain limits work.

[ ] Mailbox limits work.

[ ] Address limits work.

[ ] Concurrent limit enforcement works.

[ ] Domain verification works.

[ ] Verification state machine works.

[ ] Verification retries work.

[ ] Verified domain activates correctly.

[ ] Stalwart domain synchronization works.

[ ] Platform-owned domain model exists.

[ ] Reserved-address policy is centralized.

[ ] Account suspension works.

[ ] Account reactivation works.

[ ] Suspension provisioning is asynchronous and idempotent.

[ ] Plan changes create synchronization jobs.

[ ] Stalwart quota synchronization works.

[ ] Downgrade does not delete mail.

[ ] Usage endpoint works.

[ ] Product billing boundary exists.

[ ] Billing event idempotency works.

[ ] Audit records sensitive product actions.

[ ] Admin authorization works.

[ ] Failed provisioning can be retried.

[ ] Reconciliation command works.

[ ] No mail state is added to Norest PostgreSQL.

[ ] No mail proxy is introduced.

[ ] No Redis/Kafka/RabbitMQ is introduced.

[ ] Chapter 1 regression passes.

[ ] Chapter 2 regression passes.

[ ] Chapter 3 regression passes.

[ ] go test ./... passes.

[ ] test-chapter4.sh passes.

[ ] docs are updated.
```

---

# 152. Required Chapter 4 E2E

Create:

```text
scripts/verify-chapter4/
scripts/test-chapter4.sh
```

The E2E test must perform:

```text
1. Register user.
2. Confirm product account.
3. Get available plans.
4. Verify current entitlement.
5. Create domain.
6. Start verification.
7. Verify DNS challenge.
8. Activate domain.
9. Create mailbox/address.
10. Verify mailbox limit.
11. Verify Stalwart provisioning.
12. Upgrade plan.
13. Verify entitlement changes.
14. Verify Stalwart quota changes.
15. Downgrade plan.
16. Verify new quota synchronization.
17. Suspend account.
18. Verify new product actions are blocked.
19. Verify Stalwart account restriction.
20. Reactivate account.
21. Verify Stalwart account re-enabled.
22. Verify audit events.
23. Verify billing event idempotency.
24. Verify reconciliation.
```

---

# 153. Mail regression requirement

After all Chapter 4 changes:

```text
Alice → Bob
```

must still work.

Run:

```text
go run scripts/verify-chapter3/main.go
```

and require success.

Plan/account-control changes must never break existing mail.

---

# 154. Final architecture after Chapter 4

The finished Chapter 4 system should be:

```text
                         NOREST
                           |
       +-------------------+-------------------+
       |                   |                   |
   Identity             Product             Billing
       |                   |                   |
   Users              Accounts             Events
       |               Plans/Entitlements
       |                   |
       |              Domains/Addresses
       |                   |
       +-------------------+
                           |
                     Provisioning
                           |
                           v
                       STALWART
                           |
          +----------------+----------------+
          |                |                |
       Accounts         Mailboxes         SMTP
          |                |                |
       Messages         Folders          Delivery
       Threads          Attachments       Queue
       Search           JMAP              IMAP
```

---

# 155. Absolute Chapter 4 rules

Do NOT:

```text
build a mail database
build a message cache
build a search engine
build an SMTP server
build an IMAP server
build a JMAP server
proxy mail traffic
store message bodies
store attachments
store folders
store read/unread
store message flags
introduce Kafka
introduce Redis
introduce RabbitMQ
introduce Elasticsearch
introduce Kubernetes
introduce Temporal
build a payment provider
build a DNS provider
build a full RBAC platform
build a frontend
```

Do:

```text
build product accounts
build plans
build subscriptions
build entitlements
build domain verification
build product limits
build quota synchronization
build account lifecycle
build audit
build billing boundary
build admin product APIs
```

---

# 156. Final implementation instruction to Devin

Work directly in the current repository.

Do not rewrite Chapters 1–3.

Do not revisit the Chapter 3 mail-session design.

Do not investigate OAuth again.

Do not modify the frontend.

Chapter 4 is a **backend product/control-plane chapter**.

Implement:

```text
product accounts
plans
subscriptions
entitlements
domain verification
domain readiness
platform-domain ownership
product limits
quota synchronization
account suspension/reactivation
billing boundary
audit
admin product APIs
```

Reuse the existing:

```text
PostgreSQL
Go API
worker
Stalwart client
provisioning_jobs
audit
authentication
```

Do not introduce infrastructure.

Use the current Stalwart JMAP schema/documentation for every mail-engine operation and quota field. Stalwart's management layer uses the same JMAP API for programmatic configuration, and current account/tenant quota fields are explicitly documented.

Then run:

```bash
go test ./...
docker compose config
./scripts/test-foundation.sh
./scripts/test-chapter2.sh
./scripts/verify-db-clean.sh
go run scripts/verify-chapter2/main.go
go run scripts/verify-chapter3/main.go
./scripts/test-chapter4.sh
go run scripts/verify-chapter4/main.go
```

Fix all failures.

Do not stop at "implemented".

At the end report:

```text
1. Product account model
2. Plan model
3. Subscription model
4. Entitlement resolution
5. Domain verification flow
6. Domain lifecycle
7. Product limits
8. Quota synchronization
9. Account suspension/reactivation
10. Billing boundary
11. Audit model
12. Admin APIs
13. New database migrations
14. New worker jobs
15. Stalwart operations used
16. Concurrency test results
17. Domain verification E2E
18. Plan-limit E2E
19. Quota-sync E2E
20. Suspension/reactivation E2E
21. Billing-event idempotency E2E
22. Chapter 1 regression
23. Chapter 2 regression
24. Chapter 3 regression
25. Database mail-isolation result
26. Exact startup/test commands
27. Known limitations
```

The final principle is:

> **Norest decides what the customer is entitled to. Stalwart enforces and executes the mail-side reality. PostgreSQL records the product truth. The worker synchronizes the two.**

Do not let Chapter 4 turn Norest into a second mail server.
