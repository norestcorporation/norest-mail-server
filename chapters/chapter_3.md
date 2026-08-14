# Norest Mail — Chapter 3

## Real Mail Access: JMAP Session, Folders, Messages, Search, Send & Sync

You are continuing Norest Mail after Chapter 2.

Chapter 1 and Chapter 2 are complete and verified.

Treat both chapters as the current foundation.

Do not redesign their architecture.

Do not introduce new infrastructure unless this chapter absolutely requires it.

Do not create a second mail system.

Do not create a Norest message database.

Do not build a custom JMAP server.

Do not proxy every mail operation through Norest.

The goal of Chapter 3 is to make a Norest-created mailbox a **real usable mailbox from the existing frontend**.

---

# 1. Chapter 3 objective

At the end of Chapter 3 the following user journey must work:

```text
Norest user
    ↓
Norest login
    ↓
Norest knows the user's mailbox
    ↓
obtain secure mail authorization
    ↓
connect to Stalwart JMAP
    ↓
list folders
    ↓
list messages
    ↓
open message
    ↓
mark read/unread
    ↓
star/unstar
    ↓
move message
    ↓
delete message
    ↓
search
    ↓
create draft
    ↓
send message
    ↓
receive message
    ↓
synchronize changes
```

Actual mail state remains in Stalwart.

Norest remains the product/control plane.

---

# 2. Immutable architecture

The system remains:

```text
                         Browser
                           |
              +------------+------------+
              |                         |
              v                         v
        Norest REST API            Stalwart JMAP
              |                         |
              v                         v
         PostgreSQL                  Mail Data
```

Norest provides:

```text
identity
account
domain
address
subscription
product state
mail authorization/bootstrap
```

Stalwart provides:

```text
mailbox
email
thread
search
draft
submission
attachment
mail state
synchronization
push
```

Do not blur these boundaries.

Stalwart's JMAP implementation is specifically intended to handle mail synchronization and large volumes of data efficiently.

---

# 3. Critical rule: Norest does not become a mail proxy

Do NOT implement:

```http
GET /v1/messages
GET /v1/folders
GET /v1/threads
POST /v1/send
PATCH /v1/messages/:id
```

as the normal frontend mail path.

That would create:

```text
Browser
   ↓
Norest
   ↓
Stalwart
```

for every mail operation.

Do not build that.

The normal architecture is:

```text
Browser
   |
   +---- Norest REST
   |       |
   |    identity/product
   |
   +---- Stalwart JMAP
           |
        mail data
```

---

# 4. The authentication problem

Chapter 2 established two identities:

```text
Norest identity
Stalwart mail account
```

Chapter 3 establishes the secure bridge between them.

The user must never receive:

```text
Stalwart admin credential
```

The browser must never receive:

```text
Norest → Stalwart administrative credential
```

The browser must not receive a long-lived provisioning credential.

The browser should receive only user-level mail authorization.

Stalwart's current OAuth model uses short-lived access tokens, while its management API keys are specifically not valid for JMAP mail access.

---

# 5. First task: inspect the existing Stalwart authentication capability

Before implementing a new authentication mechanism, inspect the running Stalwart v0.16 instance.

Determine:

```text
OAuth server status
OIDC provider configuration
JMAP bearer-token support
mail account password support
application-password support
JMAP permissions
```

Do not guess.

Use the actual running Stalwart JMAP configuration and current documentation.

Stalwart supports internal-directory passwords, OAuth, and OIDC-based authentication.

Document exactly which mechanism is used for Chapter 3.

---

# 6. Preferred mail-authentication architecture

Prefer:

```text
Norest
    ↓
authenticated Norest user
    ↓
controlled mail-access authorization
    ↓
short-lived user mail token
    ↓
Stalwart JMAP
```

Do not create a second token validator inside Stalwart unless there is a documented need.

Do not make Stalwart trust the Norest JWT simply because both are JWTs.

A JWT is not automatically interoperable between two systems.

If Stalwart's OAuth/OIDC mode is selected, configure the issuer, audience, scopes, signing keys, and user identity mapping deliberately.

Stalwart supports acting as an OAuth/OIDC provider as well as delegating authentication to an external OIDC provider.

---

# 7. Do not rush into a custom OIDC server

Do not build an entire identity-provider implementation just to satisfy this chapter.

First determine whether Stalwart's built-in OAuth server can provide the required browser JMAP authorization cleanly.

If it can, prefer using it.

If it cannot satisfy the Norest product requirements, document the limitation and implement the smallest secure bridge possible.

Do not create a large OAuth framework.

---

# 8. Norest mail session endpoint

Create:

```http
POST /v1/mail/session
```

This is a Norest **authorization/bootstrap** endpoint, not a mail proxy.

Authenticated Norest user:

```text
Authorization: Bearer <Norest access token>
```

Norest verifies:

```text
user exists
user is ACTIVE
mailbox exists
mailbox is ACTIVE
address is ACTIVE
domain is available
```

Then it establishes or returns the appropriate short-lived user-level mail access information.

The endpoint must NOT return:

```text
Stalwart admin password
raw provisioning password
Stalwart API key
Norest refresh token
```

unless a deliberately designed authentication flow requires a particular browser-visible artifact.

---

# 9. Mail session response

The exact response depends on the selected Stalwart authentication mechanism.

The response should contain only what the frontend requires.

Conceptually:

```json
{
  "mail": {
    "provider": "stalwart",
    "jmap_session_url": "https://mail.example.com/.well-known/jmap",
    "access_token": "...",
    "expires_at": "..."
  }
}
```

Do not expose internal Stalwart database IDs unless needed.

Do not expose admin information.

Document the exact chosen model.

---

# 10. JMAP endpoint discovery

The frontend/client must discover the JMAP session.

Use:

```text
/.well-known/jmap
```

Stalwart's JMAP endpoint is:

```text
/jmap
```

and the well-known resource provides session details.

Do not hard-code every JMAP capability in the frontend.

The JMAP session is authoritative for:

```text
apiUrl
downloadUrl
uploadUrl
eventSourceUrl
capabilities
accounts
```

The frontend should consume the session.

---

# 11. Do not hardcode the JMAP account ID

Norest already stores:

```text
stalwart_account_id
```

for provisioning purposes.

The browser should still use the JMAP session account information returned for the authenticated user.

Do not assume:

```text
Norest UUID == Stalwart account ID
```

They are different namespaces.

Chapter 2 already demonstrated that Stalwart assigns its own IDs.

---

# 12. Create a JMAP client module

The frontend should have a small JMAP client abstraction.

Do not duplicate raw JMAP request construction across every component.

Conceptually:

```text
frontend/
    mail/
        jmap/
            session
            request
            mailbox
            email
            thread
            submission
            sync
```

The actual frontend framework and current project structure must be preserved.

Do not rebuild the frontend framework.

---

# 13. JMAP request model

JMAP uses method calls rather than traditional REST resources.

A request can contain multiple calls.

Use that capability.

Example:

```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:mail"
  ],
  "methodCalls": [
    [
      "Mailbox/get",
      {
        "accountId": "..."
      },
      "0"
    ]
  ]
}
```

Do not turn this into:

```text
GET /folders
```

inside Norest.

---

# 14. First mail operation: folders

Implement:

```text
Mailbox/get
```

This must provide enough information for the frontend to render:

```text
Inbox
Sent
Drafts
Trash
Junk
Archive
```

including:

```text
id
name
role
parentId
sortOrder
totalEmails
unreadEmails
```

Use the exact fields exposed by the JMAP session/server.

Do not assume that every server-created folder has the same names.

Use mailbox roles when available.

---

# 15. Folder representation in frontend

Create a frontend representation similar to:

```text
Mailbox {
    id
    name
    role
    parentId
    totalEmails
    unreadEmails
}
```

This is a client representation only.

Do not save the folder tree in PostgreSQL.

Optional UI caching can exist in frontend state, but Stalwart remains authoritative.

---

# 16. List messages

Implement:

```text
Email/query
```

for a selected mailbox.

The initial view should support:

```text
Inbox
Sent
Drafts
Trash
Junk
Archive
```

The query should support:

```text
position
limit
filter
sort
```

Do not request full message bodies when listing the inbox.

---

# 17. Inbox row data

The initial email list should request only the fields necessary for the list.

For example:

```text
id
threadId
mailboxIds
keywords
from
to
subject
receivedAt
preview
hasAttachment
```

Do not load the entire MIME body for every message.

This is critical for performance.

---

# 18. Pagination

Use JMAP query pagination.

Frontend should support:

```text
limit
position
total
canCalculateTotal
```

Do not load all user messages at once.

A mailbox may eventually contain millions of messages.

---

# 19. Large-account requirement

A mailbox with:

```text
100 messages
```

and a mailbox with:

```text
10 million messages
```

must use the same architecture.

Never do:

```text
Email/get every message
```

on mailbox open.

Always:

```text
query
→ page
→ fetch only visible data
```

---

# 20. Open a message

When the user clicks a message:

```text
Email/get
```

fetch the required properties.

Load:

```text
headers
subject
from
to
cc
bcc
date
receivedAt
bodyStructure
bodyValues
attachments
keywords
mailboxIds
threadId
```

Only request what is needed.

---

# 21. Message bodies

Do not copy message bodies into Norest.

Do not create:

```text
message_body_cache
```

in PostgreSQL.

Do not send the entire message through the Norest API.

The JMAP client communicates with Stalwart.

---

# 22. Mark as read

Opening a message should result in the appropriate JMAP update.

Use:

```text
Email/set
```

with the message's keywords.

For example:

```text
$seen = true
```

Do not create:

```text
messages.is_read
```

in Norest.

---

# 23. Mark as unread

Use:

```text
Email/set
```

to remove:

```text
$seen
```

Again:

```text
Stalwart = authority
```

---

# 24. Star / flag

Support:

```text
$flagged
```

through:

```text
Email/set
```

Frontend should not maintain an independent permanent flag database.

---

# 25. Draft status

Draft messages are represented through the JMAP model.

Use the appropriate mailbox/keyword semantics exposed by Stalwart.

Do not create a Norest:

```text
drafts
```

table.

---

# 26. Delete message

Implement soft/delete semantics according to Stalwart/JMAP.

For normal deletion:

```text
Email/set
```

or mailbox movement as appropriate.

Do not create Norest deletion tombstones.

Stalwart retains the actual message lifecycle.

---

# 27. Move messages

Implement moving between mailboxes.

The JMAP mechanism is to update the message's mailbox membership.

Conceptually:

```text
Inbox
   ↓
Archive
```

or:

```text
Inbox
   ↓
Trash
```

All changes happen through Stalwart.

---

# 28. Bulk operations

The frontend must eventually support:

```text
select 50 messages
mark read
mark unread
archive
delete
move
flag
```

Use JMAP batching.

Do not make 50 independent Norest API calls.

JMAP is designed to batch operations efficiently. Stalwart's v0.16 architecture specifically highlights JMAP's batch-operation capability.

---

# 29. Search

Implement JMAP search using:

```text
Email/query
```

with filters.

Initial search fields:

```text
from
to
subject
text
body
hasKeyword
inMailbox
```

The exact supported filter semantics must follow the JMAP server capability.

Do not introduce Elasticsearch.

Stalwart owns mail search.

---

# 30. Search API design

The frontend should have:

```text
searchMessages(query)
```

internally translating into JMAP.

Do not create:

```http
GET /v1/search?q=
```

that queries Norest PostgreSQL.

Search belongs to Stalwart.

---

# 31. Threads

Implement thread support using:

```text
Thread/query
Thread/get
```

where supported by the JMAP Mail capabilities.

Do not build a Norest thread database.

The server is authoritative for thread relationships.

---

# 32. Compose

Implement the first real compose flow.

User enters:

```text
To
Cc
Bcc
Subject
Body
Attachments
```

The client constructs the message using JMAP.

Do not POST the message to:

```text
/v1/send
```

The message should be created in Stalwart.

---

# 33. Draft creation

The user should be able to start a draft.

Use:

```text
Email/set
```

with the draft semantics required by JMAP.

Drafts live in Stalwart.

---

# 34. Draft autosave

Implement basic autosave.

Do not autosave every keystroke.

Use a debounce.

For example:

```text
wait 1–2 seconds after typing stops
```

then update the draft.

The exact value is configurable.

Do not build a queue service.

---

# 35. Send message

The real send flow should use:

```text
Email/set
      ↓
EmailSubmission/set
      ↓
Stalwart
      ↓
SMTP/MTA
```

Stalwart's current permission model exposes JMAP email submission operations for creating and updating submissions.

Do not implement SMTP in Norest.

---

# 36. Sender identity

Before sending, retrieve the user's allowed identities.

Use:

```text
Identity/get
```

The identity should correspond to an address the user is authorized to send from.

Do not allow arbitrary:

```text
From: ceo@some-other-domain.com
```

unless Stalwart and Norest explicitly authorize it.

---

# 37. Send flow

Expected:

```text
Compose
   ↓
Create Email
   ↓
Create EmailSubmission
   ↓
submission accepted
   ↓
Stalwart MTA
   ↓
SMTP delivery
```

The frontend should display:

```text
sending
sent
failed
```

based on JMAP submission results.

---

# 38. Attachment upload

Use the JMAP upload mechanism from the session.

Do not upload attachments to Norest PostgreSQL.

Do not create:

```text
attachments
```

table.

The JMAP session provides the upload URL.

Use that.

---

# 39. Attachment download

Use the JMAP download URL from the session.

Do not proxy attachment downloads through Norest.

This is important because attachments can be very large.

---

# 40. Message preview

Inbox previews should use:

```text
preview
```

or an equivalent limited representation.

Do not download the complete HTML body for every message.

---

# 41. HTML email rendering security

This is a frontend security requirement.

Never directly inject arbitrary email HTML into the main application DOM without sanitization.

Implement:

```text
HTML email
    ↓
sanitization
    ↓
safe rendering
```

Disallow active content such as scripts.

Images and external resources should be handled deliberately.

Do not allow email content to access Norest application storage or authentication tokens.

---

# 42. Link handling

Email links should open safely.

Do not allow an email body to navigate the main application using privileged internal routes automatically.

Use normal browser security controls.

---

# 43. External image policy

Implement a basic policy:

```text
external images disabled initially
```

or:

```text
user chooses "load images"
```

Do not proxy external images through Norest in Chapter 3.

A mail-image proxy is a later privacy/product feature.

---

# 44. Synchronization

This is one of the most important parts of Chapter 3.

Do not repeatedly reload the entire Inbox.

Use JMAP state/change mechanisms.

Implement:

```text
Email/changes
Mailbox/changes
Email/queryChanges
```

where applicable.

Stalwart exposes permissions for these change-tracking operations.

---

# 45. Local synchronization state

The browser may maintain:

```text
mailboxState
emailState
queryState
```

as runtime/client state.

Do not store these as authoritative Norest records.

For the first implementation, they can be persisted in browser storage if useful.

Stalwart remains authoritative.

---

# 46. Initial synchronization

When the application opens:

```text
1. Get JMAP session
2. Get Mailboxes
3. Query first page of Inbox
4. Fetch visible email metadata
5. Fetch unread counts
```

Do not download all messages.

---

# 47. Incremental synchronization

After initial load:

```text
Email/changes
Mailbox/changes
```

should be used to identify changes since the last known state.

The frontend then updates only affected records.

---

# 48. Push notifications

Do not poll every second.

Stalwart currently supports event-source notifications and WebSocket-based JMAP push mechanisms. Event source uses a long-lived `text/event-stream`; the JMAP session exposes the event source URL.

Implement a clean push abstraction:

```text
MailPushClient
```

Do not hardwire event-source code into UI components.

---

# 49. Push implementation

Preferred initial implementation:

```text
JMAP EventSource
```

because it is simple HTTP streaming.

The event-source endpoint is exposed by the JMAP session and supports notification types.

Do not build WebSocket infrastructure unless it provides a concrete benefit for the current frontend.

---

# 50. Push fallback

If event source is unavailable:

```text
fallback to change polling
```

Do not treat push as the authoritative state.

Push tells the client:

```text
something changed
```

The client then synchronizes authoritative state from JMAP.

---

# 51. WebSocket

Do not make WebSocket mandatory in Chapter 3.

Stalwart supports JMAP over WebSocket, including heartbeat and throttling configuration, but event source is sufficient for the initial implementation.

Keep the client abstraction ready to support WebSocket later.

---

# 52. Mail state architecture

The complete relationship should be:

```text
Browser state
    ↓
JMAP
    ↓
Stalwart state
```

Not:

```text
Browser
   ↓
Norest PostgreSQL
   ↓
Stalwart
```

---

# 53. Norest database additions

Do NOT add:

```text
messages
emails
folders
threads
attachments
search_index
mail_flags
```

The only Chapter 3 database addition that may be necessary is a minimal product-level mail preference/state table.

Examples:

```text
mail_preferences
```

only if the frontend actually requires Norest-owned preferences such as:

```text
timezone
default_sender
signature preference
```

Do not store mail content.

Prefer adding no database table in Chapter 3 unless necessary.

---

# 54. Norest API additions

Keep Norest APIs minimal.

Implement only:

```http
POST /v1/mail/session
GET  /v1/mail/account
```

The second endpoint should return product information, for example:

```json
{
  "address": "alice@example.com",
  "status": "ACTIVE"
}
```

Do not return messages.

Do not return folders.

Do not return message bodies.

Do not return attachments.

---

# 55. Mail account endpoint

Implement:

```http
GET /v1/mail/account
```

Authenticated user only.

Return:

```text
address
status
mailbox status
plan/limit information if available
```

Do not expose administrative Stalwart information.

---

# 56. Mail session endpoint security

`POST /v1/mail/session` must verify:

```text
authenticated Norest user
        +
mailbox ownership
        +
mailbox ACTIVE
        +
address ACTIVE
```

Never accept:

```json
{
  "mailbox_id": "someone-elses-id"
}
```

as the sole authority.

Determine the mailbox from the authenticated Norest user.

---

# 57. Multiple addresses

A Norest user may eventually have:

```text
alice@example.com
alice@company.com
support@example.com
```

Chapter 3 should design the mail-access model so multiple authorized mail accounts can be supported later.

Do not assume one user permanently equals one mailbox.

The first implementation may expose one primary mailbox.

Document the limitation.

---

# 58. Multiple accounts in JMAP

JMAP sessions can expose multiple accounts.

The frontend should not assume:

```text
accounts = exactly 1
```

even if Chapter 3 currently provisions one primary mailbox.

Use the account ID returned in the session.

---

# 59. Shared mailboxes

Do not implement shared mailbox delegation in Chapter 3.

However, do not build architecture that prevents it later.

Stalwart's authorization model supports per-user/group/tenant permissions for JMAP operations.

Leave the feature for a later chapter.

---

# 60. Mail API client architecture

The frontend JMAP implementation should have:

```text
MailClient
    |
    +-- SessionClient
    +-- MailboxClient
    +-- EmailClient
    +-- ThreadClient
    +-- SubmissionClient
    +-- AttachmentClient
    +-- SyncClient
    +-- PushClient
```

Do not create hundreds of abstraction layers.

Keep each client thin.

---

# 61. Request batching

The JMAP client should support:

```text
methodCalls[]
```

and allow dependent calls using JMAP result references.

For example:

```text
Email/query
    ↓
Email/get using returned IDs
```

can be performed efficiently.

Stalwart's v0.16 architecture explicitly highlights JMAP batch operations as one of the advantages of the unified JMAP management/data model.

---

# 62. Error handling

The JMAP client must distinguish:

```text
authentication failure
authorization failure
invalid arguments
not found
rate limit
server unavailable
network timeout
temporary failure
```

Do not show raw JMAP JSON errors to users.

Translate them into frontend-friendly states.

---

# 63. Authentication expiry

When the mail access token expires:

```text
JMAP request
    ↓
401
    ↓
mail session refresh/reacquire
    ↓
retry once
```

Do not retry indefinitely.

Do not retry non-idempotent send operations blindly.

---

# 64. Send retry safety

Sending email requires extra care.

Do not automatically retry:

```text
EmailSubmission/set
```

multiple times after an ambiguous network failure unless the implementation can determine whether the submission already succeeded.

Otherwise one user action could send multiple emails.

Implement a safe submission state.

---

# 65. Frontend loading states

The frontend must distinguish:

```text
loading folders
loading messages
loading message
sending
saving draft
syncing
offline
authentication expired
server unavailable
```

Do not block the entire application while loading a single message.

---

# 66. Offline behavior

Do not build a complete offline mail client yet.

The initial implementation should gracefully handle:

```text
temporary network loss
```

and reconnect.

Do not create a complicated offline database.

---

# 67. Performance requirements

The frontend must not:

```text
fetch all messages
fetch full bodies for list
send individual REST calls for every message
poll constantly
download all attachments
```

The architecture should remain efficient for large accounts.

Stalwart's JMAP implementation is designed for large-volume synchronization and supports push/change mechanisms.

---

# 68. Backend responsibility

Norest backend should NOT:

```text
fetch every user's email
cache every message
index mail
store mail
proxy attachments
proxy message bodies
```

The Norest API should remain lightweight.

Its job is authorization/product state.

---

# 69. Backend Stalwart client responsibility

The Go Stalwart client remains responsible for:

```text
administrative provisioning
product-level validation where necessary
future reconciliation
```

It should not become the runtime mail proxy.

The existing `internal/stalwart` management client remains separate from browser mail access.

---

# 70. Runtime mail traffic

Normal webmail traffic should be:

```text
Browser
   ↓
Stalwart JMAP
```

not:

```text
Browser
   ↓
Norest Go API
   ↓
Stalwart
```

This is critical for millions of users.

---

# 71. CORS

Because the browser will eventually communicate directly with Stalwart, configure the deployment/origin model carefully.

Do not simply enable:

```text
Access-Control-Allow-Origin: *
```

with credentials.

Use the actual Norest web origin.

The final deployment architecture must have:

```text
https://mail.norest.com
```

and the Stalwart JMAP endpoint configured appropriately.

For local development, configure:

```text
http://localhost:<frontend-port>
```

explicitly.

---

# 72. Development URLs

Keep Chapter 3 development environment simple.

Example:

```text
Norest API:
http://localhost:8080

Stalwart:
http://localhost:8081

JMAP:
http://localhost:8081/jmap
```

The frontend must use the configured values, not hardcoded production URLs.

---

# 73. No new infrastructure

Do not add:

```text
Redis
Kafka
RabbitMQ
Elasticsearch
NATS
Temporal
Kubernetes
```

The mail traffic is already handled by Stalwart.

---

# 74. No Norest mail cache

Do not implement:

```text
Redis mail cache
PostgreSQL mail cache
filesystem mail cache
Elastic mail index
```

If frontend caching is useful, use browser/client state.

---

# 75. Mailbox list acceptance test

A newly provisioned mailbox:

```text
alice@example.test
```

must be able to:

```text
authenticate
↓
JMAP session
↓
Mailbox/get
```

and return at least the expected system mailboxes.

This must use the actual account provisioned by Chapter 2.

No manually-created Stalwart account.

---

# 76. Inbox acceptance test

After provisioning:

```text
alice@example.test
bob@example.test
```

send:

```text
alice → bob
```

Then Bob's frontend/JMAP client must:

```text
Email/query
Email/get
```

and display the message.

---

# 77. Read/unread acceptance test

Initial:

```text
$seen = false
```

Open message.

Then:

```text
$seen = true
```

Refresh/re-query.

The server must remain authoritative.

---

# 78. Send acceptance test

The frontend should send:

```text
alice → bob
```

and the destination mailbox must receive it.

Use:

```text
Email/set
EmailSubmission/set
```

not SMTP from Norest.

---

# 79. Search acceptance test

After several messages exist:

```text
search "hello"
```

must return the matching messages using Stalwart JMAP filtering.

No PostgreSQL search.

No Elasticsearch.

---

# 80. Push acceptance test

With Bob's mailbox open:

```text
Alice sends new email
        ↓
Stalwart
        ↓
Bob event source
        ↓
notification
        ↓
Email/changes or sync
        ↓
frontend updates
```

Do not trust the push event itself as the full message state.

Use it as a trigger to synchronize authoritative data.

---

# 81. Pagination acceptance test

Generate enough messages to exceed the first page.

Verify:

```text
page 1
page 2
page 3
```

without loading every message at once.

---

# 82. Attachment acceptance test

Send a message with an attachment.

Verify:

```text
attachment appears
download works
message body works
```

without storing the attachment in Norest.

---

# 83. Large-message safety

Set reasonable frontend safeguards.

Do not render enormous message bodies without limits.

Do not load massive attachments automatically.

Use streaming/download URLs when appropriate.

---

# 84. Security acceptance tests

Test that:

```text
Alice's Norest token
cannot access Bob's Norest account
```

and:

```text
Alice's mail authorization
cannot access Bob's Stalwart account
```

Also test:

```text
expired access token
invalid token
missing token
wrong audience
wrong account
```

The last two depend on the selected authentication mechanism.

---

# 85. Stalwart permissions

Check the actual Stalwart permissions for the user account.

The account must have enough permission for:

```text
JMAP session
Mailbox/get
Mailbox/query
Email/get
Email/query
Email/set
EmailSubmission/set
Email/change operations
```

but not administrative configuration.

Stalwart's permissions model exposes separate JMAP permissions for mailbox/email/submission/change operations.

Do not grant:

```text
system management
server configuration
domain management
account management
```

to normal users.

---

# 86. Admin/user separation

The system must have:

```text
Norest administrative Stalwart credential
```

used only by:

```text
internal/stalwart management
```

and:

```text
normal user mail authorization
```

used only for:

```text
JMAP mail
```

Never mix them.

---

# 87. No admin JMAP from browser

The frontend must never make:

```text
x:Domain/set
x:Account/set
x:Bootstrap/set
x:Server/set
```

requests.

Those remain server-side management operations.

---

# 88. No mailbox provisioning from browser

The browser must never call Stalwart management methods to create accounts.

The user performs:

```text
Norest API
```

product operations.

The worker performs:

```text
Stalwart management
```

provisioning operations.

The browser performs:

```text
Stalwart JMAP mail operations
```

mail operations.

Three clear responsibilities.

---

# 89. Chapter 3 folder structure

Preserve existing repository structure.

Add only the required frontend/backend organization.

Backend:

```text
internal/
├── auth/
├── users/
├── domains/
├── addresses/
├── provisioning/
├── stalwart/
├── audit/
├── mail/
│   ├── service.go
│   ├── handler.go
│   └── models.go
└── ...
```

`internal/mail` should remain very small.

It should primarily contain:

```text
mail account authorization
mail session bootstrap
mail-access policy
```

It must NOT contain:

```text
messages
folders
email storage
search engine
SMTP server
```

Frontend should extend the existing project rather than creating a second application.

---

# 90. Backend mail service

Implement:

```go
type MailService struct {
    ...
}
```

Responsibilities:

```text
GetMailAccount(userID)
CreateMailSession(userID)
ValidateMailAccess(userID)
```

No:

```go
GetMessages()
CreateMessage()
SearchMessages()
```

Those are JMAP client operations.

---

# 91. Frontend mail service

Implement:

```text
MailService
```

with:

```text
getSession
getMailboxes
queryEmails
getEmails
updateEmails
queryThreads
getThreads
createDraft
send
sync
subscribeToPush
```

The frontend service talks directly to JMAP.

---

# 92. API state model

Norest's view:

```text
User
   ↓
Address
   ↓
Mailbox
   ↓
ACTIVE
```

Stalwart's view:

```text
Account
   ↓
Mailboxes
   ↓
Emails
   ↓
Threads
```

Do not merge these models.

---

# 93. API performance

`POST /v1/mail/session` should be cheap.

Do not call:

```text
Email/query
Mailbox/get
Email/get
```

inside it.

It should establish authorization/session information only.

---

# 94. Mail session caching

Do not cache mail contents.

If a short-lived mail authorization/session artifact is safe to cache, keep caching minimal.

Do not use Redis.

Start without a server-side cache.

---

# 95. Refresh behavior

When a mail authorization expires:

```text
frontend
    ↓
Norest /v1/mail/session
    ↓
new short-lived mail access
```

Do not require the user's Norest password again every few minutes.

Do not expose the Stalwart account password.

---

# 96. Token scopes

When using OAuth, request only the scopes required for webmail.

At minimum the chosen scopes should cover the JMAP mail operations needed by the application.

Do not request broad administrative access.

Document the exact scope names supported by the deployed Stalwart version.

---

# 97. Token lifetime

Use short-lived mail access tokens.

Do not create one token that lives forever.

The exact lifetime should be configurable.

The chapter must document:

```text
access-token lifetime
refresh/reacquisition behavior
logout behavior
revocation behavior
```

---

# 98. Logout

Norest logout must invalidate Norest authentication.

Frontend should also discard:

```text
mail access token
JMAP session state
cached account data
```

Do not persist mail authorization longer than necessary.

---

# 99. Browser storage

Do not place:

```text
Stalwart admin credential
Norest refresh token
```

in localStorage.

For mail access tokens, use the safest storage model supported by the chosen authentication flow.

Prefer an architecture where browser JavaScript does not need to hold long-lived credentials.

---

# 100. Chapter 3 acceptance criteria

Chapter 3 is complete only when:

```text
[ ] Norest authenticated user can establish mail access.

[ ] Mail access is user-scoped.

[ ] Admin Stalwart credentials never reach frontend.

[ ] JMAP session discovery works.

[ ] Newly Norest-provisioned mailbox can access JMAP.

[ ] Mailbox/get works.

[ ] Inbox list works.

[ ] Message pagination works.

[ ] Email/query works.

[ ] Email/get works.

[ ] Message preview works.

[ ] Read/unread works.

[ ] Flag/star works.

[ ] Move works.

[ ] Delete works.

[ ] Search works.

[ ] Thread/query works.

[ ] Thread/get works.

[ ] Draft creation works.

[ ] Draft update/autosave works.

[ ] Identity/get works.

[ ] EmailSubmission/set works.

[ ] Send works.

[ ] Attachment upload works.

[ ] Attachment download works.

[ ] Initial synchronization works.

[ ] Incremental synchronization works.

[ ] Push/event-source works.

[ ] Push failure falls back safely to synchronization.

[ ] No mail data exists in PostgreSQL.

[ ] No Norest mail proxy exists.

[ ] No Elasticsearch exists.

[ ] No Redis exists.

[ ] Chapter 1 tests pass.

[ ] Chapter 2 provisioning tests pass.

[ ] Database isolation test passes.

[ ] Multi-worker provisioning still passes.

[ ] Authentication security tests pass.

[ ] Cross-account access tests pass.

[ ] go test ./... passes.

[ ] frontend production build passes.

[ ] full mail E2E test passes.
```

---

# 101. Required Chapter 3 E2E test

Create:

```text
scripts/test-chapter3-e2e/
```

or an equivalent test location.

The test must perform:

```text
1. Create/provision Alice through Norest.
2. Create/provision Bob through Norest.
3. Authenticate Alice.
4. Obtain Alice's mail access.
5. Get Alice's JMAP session.
6. Get Alice's mailboxes.
7. Authenticate Bob / establish Bob's session.
8. Send Alice → Bob.
9. Query Bob inbox.
10. Retrieve the message.
11. Verify subject/from/body.
12. Verify unread state.
13. Mark it read.
14. Verify $seen.
15. Search for the message.
16. Flag it.
17. Move it.
18. Delete it.
19. Restore/recover where applicable.
20. Create/send an attachment message.
21. Verify attachment retrieval.
22. Verify push/change notification.
```

The test must use:

```text
real Norest
real PostgreSQL
real Stalwart
real JMAP
```

No mocks for the end-to-end path.

---

# 102. Performance E2E test

Create at least:

```text
1000 test messages
```

and verify:

```text
first-page query
second-page query
search
open one message
```

without excessive response size.

Do not create thousands of messages in every normal test run.

Provide a separate load-test command.

---

# 103. Mail data isolation test

The existing:

```text
scripts/verify-db-clean.sh
```

must continue passing.

Additionally inspect:

```text
information_schema.tables
```

and fail if any table appears related to:

```text
message
email
folder
thread
attachment
mail
```

unless explicitly documented as a product preference/configuration table.

---

# 104. Security test

Verify that a user cannot:

```text
change another user's mailbox state
read another user's JMAP session
obtain another user's mail token
```

Use two independent accounts.

---

# 105. No authorization confusion

Do not use:

```text
Norest user UUID
```

as a substitute for:

```text
Stalwart account authorization
```

The authenticated identity must be mapped deliberately.

---

# 106. Documentation

Update:

```text
README.md
docs/architecture.md
api/openapi.yaml
```

Add:

```text
docs/mail-access.md
```

Document:

```text
Norest authentication
mail authorization
JMAP discovery
JMAP operations
frontend architecture
synchronization
push
security boundaries
```

---

# 107. Architecture diagram

Add:

```text
                         USER
                           |
                     Norest Login
                           |
                    Norest REST API
                           |
                  /v1/mail/session
                           |
                           v
                 Mail authorization
                           |
                           v
                     JMAP Session
                           |
                           v
                    STALWART JMAP
                           |
       +-------------------+-------------------+
       |                   |                   |
    Mailboxes            Emails             Search
       |                   |                   |
      Folders            Threads           Query
       |                   |                   |
       +-------------------+-------------------+
                           |
                      Submission
                           |
                          SMTP
```

---

# 108. Important data-ownership diagram

Document:

```text
Norest PostgreSQL
─────────────────
User
Domain
Address
Mailbox linkage
Subscription later
Audit

Stalwart
────────
Mailbox
Message
Thread
Attachment
Flag
Read state
Search
Submission
Delivery
```

---

# 109. Chapter 3 must not solve Chapter 4

Do not implement:

```text
billing
plans
payment
DNS verification
DKIM UI
SPF management
custom domain DNS automation
anti-spam product
anti-phishing product
advanced filtering
shared mailboxes
delegation
admin console
mobile push infrastructure
multi-region
multi-cluster
```

Those come later.

---

# 110. Important distinction: IMAP/SMTP

Chapter 3 webmail uses:

```text
JMAP
```

Do not create an Norest IMAP or SMTP server.

External mail clients can continue talking directly to Stalwart:

```text
Outlook ─── IMAP/SMTP ───> Stalwart
Thunderbird ─ IMAP/SMTP ─> Stalwart
Apple Mail ── IMAP/SMTP ─> Stalwart
```

Stalwart already provides those protocols.

Norest does not need to proxy them.

---

# 111. Current external mail flow

The actual production direction is:

```text
Internet
   |
   v
Stalwart SMTP
   |
   v
mailbox
```

and:

```text
External sender
   ↓
Stalwart
   ↓
Alice mailbox
   ↓
JMAP
   ↓
Norest webmail frontend
```

Norest does not sit between SMTP and the mailbox.

---

# 112. Current internal mail flow

For:

```text
alice@norest.com
        →
bob@norest.com
```

Stalwart handles the actual message delivery.

Norest does not create an internal mail routing service.

---

# 113. Final success condition

Chapter 3 is complete when a human can perform this with the existing frontend:

```text
LOGIN
  ↓
Open Inbox
  ↓
See folders
  ↓
See messages
  ↓
Open message
  ↓
Mark read
  ↓
Search
  ↓
Compose
  ↓
Attach file
  ↓
Send
  ↓
Recipient receives it
  ↓
New mail notification appears
  ↓
Open received message
```

All actual mail operations happen through Stalwart JMAP.

---

# 114. Final instruction to YOU

Work directly in the current repository.

Do not rewrite Chapter 1.

Do not rewrite Chapter 2.

Do not introduce new infrastructure.

First inspect the current frontend and identify the existing mail UI/components that should be connected to JMAP.

Then implement the smallest complete Mail Access Layer.

Create:

```text
/v1/mail/session
/v1/mail/account
```

only where necessary.

Implement the frontend JMAP client.

Implement:

```text
Mailbox/get
Mailbox/query
Email/query
Email/get
Email/set
Thread/query
Thread/get
EmailSubmission/set
Identity/get
Email/changes
Mailbox/changes
Email/queryChanges
```

only as required by the actual user flows.

Implement JMAP event-source synchronization.

Implement secure mail authorization.

Do not invent a custom mail REST API.

Then run:

```bash
go test ./...
docker compose config
./scripts/test-foundation.sh
./scripts/test-chapter2.sh
./scripts/verify-db-clean.sh
go run scripts/verify-chapter2/main.go
```

and the new:

```bash
scripts/test-chapter3-e2e
```

plus the frontend production build.

Fix all failures.

Do not stop at "implemented".

At the end report:

```text
1. Authentication architecture selected
2. Why that authentication architecture was selected
3. Exact Norest mail endpoints
4. Exact JMAP methods implemented
5. Frontend files changed
6. Backend files changed
7. Synchronization design
8. Push design
9. Attachment design
10. Security boundaries
11. Cross-account security test
12. Full send/receive E2E result
13. Search result
14. Read/unread result
15. Attachment result
16. Push/sync result
17. Database isolation result
18. Chapter 1 regression result
19. Chapter 2 regression result
20. Exact commands to run
21. Known limitations
```

Most importantly:

**Do not build a second mail system.**

The purpose of Chapter 3 is to connect the existing Norest product to the actual Stalwart mail engine so that:

```text
Norest
  =
identity + product + authorization

Stalwart
  =
mail
```

The end result must be a real usable mailbox, not another backend abstraction that merely says the mailbox exists.
