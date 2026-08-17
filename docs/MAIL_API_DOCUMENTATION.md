# Norest Mail API Documentation

## Architecture Overview

Norest Mail follows a **control plane / data plane separation** architecture:

- **Norest (Control Plane)**: User authentication, domain management, address provisioning, session creation
- **Stalwart (Data Plane)**: Actual mail operations via JMAP protocol (mailboxes, messages, sending, receiving)

```text
Client Application
        ↓
Norest REST API (Authentication, Provisioning, Session Creation)
        ↓
Stalwart JMAP API (Mail Operations using JMAP credentials)
        ↓
Mail Data Storage
```

## Current Norest Mail API Implementation

### Implemented and Usable Now

The following Norest REST endpoints are currently implemented and ready for use:

| API | Method | Endpoint | Purpose | Authentication | Depends on |
|-----|--------|----------|---------|----------------|------------|
| Get Mail Account | GET | `/v1/mail/account` | Retrieve mailbox metadata (ID, status, Stalwart account ID) | Bearer token | User registration & provisioning |
| Check Provisioning Status | GET | `/v1/mail/provisioning-status` | Check mailbox readiness and provisioning state | Bearer token | User registration & provisioning |
| Create Mail Session | POST | `/v1/mail/session` | Create JMAP session with credentials for Stalwart access | Bearer token | Active mailbox provisioning |
| Search Messages | GET | `/v1/mail/search` | Search for messages using text, from, to, subject, etc | Bearer token | Active mailbox provisioning |
| List Messages | GET | `/v1/mail/messages` | List and paginate messages | Bearer token | Active mailbox provisioning |
| Get Message | GET | `/v1/mail/messages/{id}` | Get message details | Bearer token | Active mailbox provisioning |
| Send Mail | POST | `/v1/mail/send` | Send an email with strict idempotency | Bearer token | Active mailbox provisioning |
| Create Draft | POST | `/v1/mail/drafts` | Create a new draft message | Bearer token | Active mailbox provisioning |
| Update Draft | PUT | `/v1/mail/drafts/{id}` | Update an existing draft | Bearer token | Active mailbox provisioning |
| Delete Draft | DELETE | `/v1/mail/drafts/{id}` | Delete a draft | Bearer token | Active mailbox provisioning |
| Message Actions | POST | `/v1/mail/messages/{id}/{action}` | Read, unread, archive, trash | Bearer token | Active mailbox provisioning |
| Upload Attachment | POST | `/v1/mail/attachments` | Upload an attachment up to 25MB | Bearer token | Active mailbox provisioning |
| Download Attachment | GET | `/v1/mail/attachments/{id}` | Securely proxy attachment download | Bearer token | Active mailbox provisioning |
| Threads | GET | `/v1/mail/threads` | List threads | Bearer token | Active mailbox provisioning |
| Reply/Forward | POST | `/v1/mail/messages/{id}/reply` | Reply or forward | Bearer token | Active mailbox provisioning |
| Sync | GET | `/v1/mail/sync` | Incremental sync state | Bearer token | Active mailbox provisioning |
| Realtime | WS | `/v1/mail/realtime` | WebSocket connection for outbox events | Bearer token | Active mailbox provisioning |

### Not Currently Implemented (Future REST API Development)

None. The API is feature complete.

The following mail operations are **NOT** available as Norest REST endpoints. They must be performed via direct JMAP communication with Stalwart:

### Message Operations
- List messages
- Get message details
- Search messages
- Message pagination
- Message filtering

### Compose & Send
- Send mail
- Create drafts
- Update drafts
- Delete drafts
- Send drafts

### Message State Actions
- Mark as read/unread
- Star/unstar
- Archive
- Move to trash
- Restore from trash
- Mark as spam
- Remove from spam
- Add/remove flags

### Thread Operations
- List threads
- Get thread details
- Thread actions

### Attachment Operations
- Upload attachments
- Download attachments
- Get attachment metadata

### Mailbox/Folder Operations
- Create folders
- Rename folders
- Delete folders
- Get folder counts
- Folder state synchronization

## JMAP Operations Available via Stalwart

The Norest codebase includes a comprehensive Stalwart JMAP client with support for:

### Mailbox Operations (RFC 8621)
- `Mailbox/get` - Retrieve mailboxes/folders
- `Mailbox/query` - Query mailboxes
- `Mailbox/set` - Create/update/delete mailboxes

### Email Operations (RFC 8621)
- `Email/get` - Retrieve email objects
- `Email/query` - Query/filter emails
- `Email/set` - Create/update/delete emails
- `Email/changes` - Get email changes

### Thread Operations (RFC 8621)
- `Thread/get` - Retrieve thread information
- `Thread/query` - Query threads

### Search & Filter
- Complex filtering (in mailbox, from, to, subject, date, keywords)
- Sorting options
- Pagination with limits

### Session Management
- JMAP session discovery
- Capability negotiation
- Account information

## Implementation Recommendations

### Option 1: Direct JMAP Integration (Current Approach)
- Clients implement JMAP protocol directly
- Minimal REST API surface
- Standardized protocol (RFC 8620/8621)
- Requires JMAP client library implementation

### Option 2: REST API Wrappers (Future Development)
Implement Norest REST endpoints that wrap JMAP operations:

```go
// Example future endpoints
GET    /v1/mail/mailboxes
GET    /v1/mail/mailboxes/{id}
POST   /v1/mail/mailboxes
PUT    /v1/mail/mailboxes/{id}
DELETE /v1/mail/mailboxes/{id}

GET    /v1/mail/messages
GET    /v1/mail/messages/{id}
POST   /v1/mail/messages
PUT    /v1/mail/messages/{id}
DELETE /v1/mail/messages/{id}

POST   /v1/mail/messages/{id}/read
POST   /v1/mail/messages/{id}/unread
POST   /v1/mail/messages/{id}/star
POST   /v1/mail/messages/{id}/archive
POST   /v1/mail/messages/{id}/trash

POST   /v1/mail/send
POST   /v1/mail/drafts
GET    /v1/mail/drafts/{id}
PUT    /v1/mail/drafts/{id}
DELETE /v1/mail/drafts/{id}

POST   /v1/mail/search
```

## Send Idempotency

The `POST /v1/mail/send` endpoint implements rigorous server-side idempotency.

`Idempotency-Key` is REQUIRED for `POST /v1/mail/send`.

**Semantics:**

* **Same key + same payload:**
  → returns the cached original result (e.g., `201 Created` or `202 Accepted`)
* **Same key + different payload:**
  → returns `400 idempotency_mismatch`
* **Request already executing:**
  → returns `409 idempotency_in_progress` (if blocked waiting for an outcome) OR the cached completed result if the previous request finishes quickly
* **Definitive pre-submission failure (e.g., system error before Stalwart):**
  → Norest clears the key, making it fully retryable
* **Submission outcome unknown (e.g., network timeout after submitting to Stalwart):**
  → returns `202 delivery_status_unknown`
  → The key remains protected in the `AMBIGUOUS` state
  → A client retry MUST NOT blindly submit again, but will instantly get the `202` cached response back

> **IMPORTANT: An idempotency key does not guarantee delivery. It guarantees that Norest will not intentionally submit the same request more than once after the outcome is known or ambiguous.**

## Postman Collection Structure

The updated Postman collection includes:

### Folder 06: Mail Actions (Current Norest APIs)
- Get Mail Account - Metadata retrieval
- Check Provisioning Status - Readiness check
- Create Mail Session - JMAP credential generation

### Folder 07: JMAP Examples (Direct Stalwart Communication)
- JMAP - Get Session - Capability discovery
- JMAP - Mailbox/get - Folder listing
- JMAP - Email/query - Message querying
- JMAP - Email/get - Message retrieval

### Environment Variables
- `mail_session_provider` - Stalwart
- `mail_session_jmap_url` - JMAP endpoint
- `mail_session_access_token` - App password for JMAP auth
- `mail_session_account_id` - Stalwart account ID

## Testing Strategy

### Current Testing
1. Register user via Norest
2. Wait for provisioning completion
3. Create mail session
4. Use JMAP examples to test mail operations
5. Verify Stalwart communication

### Future Testing (with REST wrappers)
1. Test all REST endpoints for mail operations
2. Verify JMAP integration
3. Test error handling and edge cases
4. Validate authentication and authorization
5. Performance testing for pagination and large datasets

## Security Considerations

### Current Architecture
- Norest handles user authentication via JWT
- Mail sessions use short-lived app passwords
- JMAP credentials are scoped to specific accounts
- Rate limiting on session creation

### Future REST API Development
- Maintain JWT authentication for all endpoints
- Implement proper authorization checks
- Rate limiting per user for mail operations
- Input validation and sanitization
- Secure handling of attachments
- Protection against email injection attacks

## Performance Considerations

### Current Architecture
- Session creation is rate limited
- Direct JMAP communication reduces latency
- Stalwart handles mail data storage and indexing
- Norest focuses on control plane operations

### Future REST API Development
- Implement caching for frequently accessed data
- Pagination for large message lists
- Optimized JMAP query generation
- Connection pooling for Stalwart communication
- Async processing for expensive operations

## Monitoring & Observability

### Current Metrics
- Session creation success/failure rates
- Provisioning status distribution
- Stalwart health checks
- API response times

### Future Metrics (with REST wrappers)
- Per-endpoint request rates
- JMAP operation latency
- Message query performance
- Error rates by operation type
- User activity patterns

## Conclusion

The current Norest implementation provides a solid foundation for mail service provisioning with a clean separation between control plane (Norest) and data plane (Stalwart). The architecture supports both direct JMAP integration (current) and future REST API wrapper development for simplified client integration.

The updated Postman collection reflects the current implementation state and provides examples for both Norest REST endpoints and direct JMAP communication with Stalwart.