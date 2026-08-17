# Norest Mail API Implementation Assessment

## A. Already Available Internally

### JMAP Client Capabilities (Stalwart)

#### Mailbox Operations
- ✅ `Mailbox/get` - Retrieve all mailboxes or specific ones by ID
- ✅ `DiscoverMailboxes` - Get role-to-ID mappings (Inbox, Sent, Trash, etc.)
- ✅ `GetMailboxByName` - Find specific mailbox by name
- ❌ `Mailbox/set` - Create/update/delete mailboxes (NOT IMPLEMENTED)
- ❌ `Mailbox/query` - Query mailboxes (NOT IMPLEMENTED)

#### Email Operations
- ✅ `Email/query` - Query emails with filters, sorting, limits
- ✅ `Email/get` - Get emails by IDs with property selection
- ✅ Support for filters, sorting, pagination
- ❌ `Email/set` - Create/update/delete emails (NOT IMPLEMENTED)
- ❌ `Email/import` - Import emails (NOT IMPLEMENTED)
- ❌ `Email/copy` - Copy emails (NOT IMPLEMENTED)
- ❌ `Email/changes` - Get email changes (NOT IMPLEMENTED)

#### Thread Operations
- ✅ Thread information available in Email objects (`threadId`, `totalThreads`, `unreadThreads`)
- ❌ `Thread/get` - Get thread details (NOT IMPLEMENTED)
- ❌ `Thread/query` - Query threads (NOT IMPLEMENTED)
- ❌ `Thread/changes` - Get thread changes (NOT IMPLEMENTED)

#### Sending/Draft Operations
- ❌ `EmailSubmission/set` - Send emails (NOT IMPLEMENTED)
- ❌ Draft creation/update/delete via Email/set (NOT IMPLEMENTED)
- ❌ Draft sending workflow (NOT IMPLEMENTED)

#### Attachment Operations
- ✅ Blob information available in Email objects (`blobId`, `hasAttachment`)
- ❌ `Blob/get` - Download attachments (NOT IMPLEMENTED)
- ❌ Blob upload/import (NOT IMPLEMENTED)
- ❌ Attachment metadata operations (NOT IMPLEMENTED)

#### Session & Discovery
- ✅ `GetJMAPWellKnown` - JMAP discovery endpoint
- ✅ `GetJMAPSession` - Authenticated session with capabilities
- ✅ Upload/Download URLs from session
- ✅ Capability negotiation

#### State/Actions Support
- ✅ Keywords available in Email objects (`keywords` map for flags)
- ✅ Mailbox assignments (`mailboxIds` map for folder membership)
- ❌ No direct flag manipulation methods (requires Email/set)

#### Synchronization
- ✅ Initial sync checkpoint in provisioning worker
- ✅ JMAP state stored in mailboxes table
- ✅ `CanCalculateChanges` in Email/query response
- ❌ No Email/changes implementation
- ❌ No Mailbox/changes implementation
- ❌ No real-time sync mechanism

### Existing Business Logic Infrastructure

#### Authorization & Security
- ✅ JWT authentication with `RequireAuth` middleware
- ✅ User ID extraction from context
- ✅ Admin role checking with `RequireAdmin`
- ✅ Context-based user identification (`UserIDFromContext`)

#### Quota & Policy Management
- ✅ Plan-based limits (MaxDomains, MaxMailboxes, MaxAddresses, MaxStorageBytes)
- ✅ Account status tracking (ACTIVE, SUSPENDED, DISABLED, PENDING)
- ✅ Usage tracking (current counts vs limits)
- ✅ Quota enforcement methods (`CanCreateDomain`, `CanCreateMailbox`, `CanCreateAddress`)
- ✅ Account suspension/reactivation
- ✅ QUOTA_SYNC job for updating Stalwart quotas

#### Ownership & Mapping
- ✅ User → Address → Mailbox → Stalwart Account mapping
- ✅ `GetMailboxByUserID` for user-to-mailbox resolution
- ✅ Mailbox role mappings persisted (mailbox_mappings table)
- ✅ Claimed address tracking
- ✅ Domain ownership validation

#### Provisioning & Synchronization
- ✅ Async job processing with retries
- ✅ Initial JMAP sync with checkpoint persistence
- ✅ Mailbox status lifecycle (pending → provisioning → syncing → active)
- ✅ Stalwart account creation and management
- ✅ App password generation for sessions

---

## B. Can Be Exposed Immediately

Based on existing JMAP capabilities, these Norest REST endpoints can be implemented with minimal work:

### Mailbox/Folder Operations
- `GET /v1/mail/mailboxes` - List all mailboxes for authenticated user
- `GET /v1/mail/mailboxes/{id}` - Get specific mailbox details
- `GET /v1/mail/mailboxes/roles` - Get role-to-ID mappings (Inbox, Sent, etc.)

### Message Operations
- `GET /v1/mail/messages` - List messages with filtering, sorting, pagination
- `GET /v1/mail/messages/{id}` - Get specific message details
- `GET /v1/mail/messages/search` - Search messages (using Email/query filters)

### Read-Only Operations
- These require only JMAP Email/query and Email/get
- Authorization already available via middleware
- Ownership validation via existing user→mailbox mapping

---

## C. Requires New JMAP Adapter Work

The following functionality requires implementing missing JMAP methods in the Stalwart client:

### Message State Actions
- **Requires**: `Email/set` implementation
- Mark read/unread (via keywords `$seen`)
- Star/unstar (via keywords `$flagged`)
- Archive (move to Archive mailbox)
- Move (change mailboxIds)
- Trash (move to Trash mailbox)
- Restore (move from Trash)
- Spam/Junk (move to Spam mailbox)

### Compose & Send
- **Requires**: `Email/set` and `EmailSubmission/set` implementation
- Create draft (Email/set with draft mailbox)
- Update draft (Email/set with existing ID)
- Delete draft (Email/set with destroy)
- Send mail (EmailSubmission/set)
- Reply/Forward (Email/set with in-reply-to/references)

### Mailbox Management
- **Requires**: `Mailbox/set` implementation
- Create folder (Mailbox/set with create)
- Rename folder (Mailbox/set with update)
- Delete folder (Mailbox/set with destroy)

### Thread Operations
- **Requires**: `Thread/get` and potentially `Thread/query` implementation
- Get thread details
- List threads
- Thread-based actions

### Attachment Operations
- **Requires**: `Blob/get` and blob upload implementation
- Upload attachment
- Download attachment
- Get attachment metadata
- Link attachments to emails

### Synchronization
- **Requires**: `Email/changes`, `Mailbox/changes` implementation
- Incremental sync
- Change notifications
- State management

---

## D. Requires New Business Logic

The following requires Norest-side business logic implementation:

### Authorization & Ownership Validation
- **Required**: Validate user owns requested mailbox/message/thread
- **Implementation**: Use existing `GetMailboxByUserID` + JMAP ID validation
- **Security**: Prevent ID spoofing and cross-user access

### Quota & Policy Enforcement
- **Required**: Enforce sending limits, storage quotas before operations
- **Implementation**: Integrate with existing `policy.Service` methods
- **Checks**: Account status, plan limits, sending quotas

### Audit & Logging
- **Required**: Track mail operations for compliance and debugging
- **Implementation**: Add audit logging for all mail operations
- **Events**: Send, delete, move, quota changes, policy violations

### Attachment Handling
- **Required**: Secure upload, virus scanning, size limits
- **Implementation**: Norest-side validation before Stalwart upload
- **Security**: File type validation, size limits, quota enforcement

### Synchronization State Management
- **Required**: Track sync state, handle conflicts, manage checkpoints
- **Implementation**: Extend existing sync checkpoint system
- **Features**: Incremental sync, conflict resolution, state persistence

### Error Handling & Translation
- **Required**: Convert JMAP errors to user-friendly Norest errors
- **Implementation**: Error mapping layer between JMAP and REST
- **User Experience**: Clear error messages, proper HTTP status codes

---

## E. Recommended Final Mail API

### Phase 1: Read-Only Operations (Immediate)

#### Mailbox/Folder Operations
```
GET    /v1/mail/mailboxes
GET    /v1/mail/mailboxes/{id}
GET    /v1/mail/mailboxes/roles
```

#### Message Operations
```
GET    /v1/mail/messages
GET    /v1/mail/messages/{id}
GET    /v1/mail/messages/search
```

**Implementation**: Uses existing Email/query and Email/get with authorization wrapper.

### Phase 2: Message State Actions (Requires Email/set)

#### Message State Operations
```
POST   /v1/mail/messages/{id}/read
POST   /v1/mail/messages/{id}/unread
POST   /v1/mail/messages/{id}/star
POST   /v1/mail/messages/{id}/unstar
POST   /v1/mail/messages/{id}/archive
POST   /v1/mail/messages/{id}/move
POST   /v1/mail/messages/{id}/trash
POST   /v1/mail/messages/{id}/restore
POST   /v1/mail/messages/{id}/spam
```

**Implementation**: Requires Email/set JMAP method + ownership validation + audit logging.

### Phase 3: Compose & Send (Requires Email/set + EmailSubmission/set)

#### Draft Operations
```
POST   /v1/mail/drafts
GET    /v1/mail/drafts/{id}
PUT    /v1/mail/drafts/{id}
DELETE /v1/mail/drafts/{id}
```

#### Send Operations
```
POST   /v1/mail/send
POST   /v1/mail/drafts/{id}/send
```

**Implementation**: Requires Email/set for drafts + EmailSubmission/set for sending + quota enforcement + policy checks.

### Phase 4: Advanced Features (Requires Additional JMAP)

#### Thread Operations
```
GET    /v1/mail/threads
GET    /v1/mail/threads/{id}
```

#### Attachment Operations
```
POST   /v1/mail/attachments
GET    /v1/mail/attachments/{id}
DELETE /v1/mail/attachments/{id}
```

#### Mailbox Management
```
POST   /v1/mail/mailboxes
PUT    /v1/mail/mailboxes/{id}
DELETE /v1/mail/mailboxes/{id}
```

#### Synchronization
```
GET    /v1/mail/sync/changes
POST   /v1/mail/sync/checkpoint
```

---

## Security Architecture

### Ownership Validation Flow
```
1. Extract userID from JWT context
2. Get user's mailbox via GetMailboxByUserID(userID)
3. Get user's Stalwart account ID from mailbox
4. Validate requested resource belongs to user's Stalwart account
5. Reject if ownership cannot be verified
```

### Quota Enforcement Flow
```
1. Extract userID from JWT context
2. Get user's entitlement via policy.GetEntitlement(userID)
3. Check account status (ACTIVE vs SUSPENDED)
4. Check plan limits (MaxStorageBytes, sending limits)
5. Enforce limits before calling Stalwart
6. Track usage for quota sync jobs
```

### Authorization Layers
```
1. JWT Authentication (RequireAuth middleware)
2. User-to-Mailbox mapping (GetMailboxByUserID)
3. Stalwart account ownership validation
4. Resource-level authorization (message/thread/mailbox IDs)
5. Policy/quota enforcement
6. Audit logging
```

---

## Implementation Priority

### Priority 1: Foundation (Week 1)
1. Implement Email/set in Stalwart client
2. Implement ownership validation service
3. Add read-only message endpoints
4. Add mailbox listing endpoints

### Priority 2: Message Actions (Week 2)
1. Implement message state endpoints (read/unread, star, move, trash)
2. Add audit logging
3. Add comprehensive error handling
4. Security testing and validation

### Priority 3: Compose & Send (Week 3)
1. Implement EmailSubmission/set in Stalwart client
2. Implement draft management
3. Add quota enforcement for sending
4. Add policy checks
5. Send endpoint with rate limiting

### Priority 4: Advanced Features (Week 4+)
1. Implement Thread operations
2. Implement attachment handling
3. Implement mailbox management
4. Implement synchronization
5. Performance optimization

---

## Testing Strategy

### Unit Tests
- JMAP client method tests
- Authorization logic tests
- Quota enforcement tests
- Error mapping tests

### Integration Tests
- End-to-end mail operations
- Stalwart integration tests
- Concurrency and race conditions
- Error recovery scenarios

### Security Tests
- Authorization bypass attempts
- ID spoofing attempts
- Quota limit enforcement
- Cross-user access prevention

---

## Conclusion

The current Norest implementation has a solid foundation with:
- ✅ Comprehensive JMAP client for read operations
- ✅ Robust authentication and authorization
- ✅ Quota and policy management infrastructure
- ✅ Ownership mapping and validation
- ✅ Async provisioning and synchronization

The primary gaps are:
- ❌ Missing JMAP write operations (Email/set, EmailSubmission/set, etc.)
- ❌ Missing JMAP advanced operations (Thread, Blob, Changes)
- ❌ No mail-specific business logic layer
- ❌ No audit logging for mail operations

The recommended approach is to implement in phases, starting with read-only operations (immediate) and progressively adding write operations as JMAP client capabilities are expanded. This maintains the architectural principle of Norest as the stable client-facing API while keeping Stalwart behind the integration boundary.