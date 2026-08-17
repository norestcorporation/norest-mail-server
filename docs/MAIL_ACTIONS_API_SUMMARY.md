# Mail Actions API Summary

## Implementation Status Report

### ✅ Implemented and Usable Now (Norest REST APIs)

| API | Method | Endpoint | Purpose | Authentication | Status |
|-----|--------|----------|---------|----------------|--------|
| Get Mail Account | GET | `/v1/mail/account` | Retrieve mailbox metadata (ID, status, Stalwart account ID) | Bearer token | ✅ Working |
| Check Provisioning Status | GET | `/v1/mail/provisioning-status` | Check mailbox readiness and provisioning state | Bearer token | ✅ Working |
| Create Mail Session | POST | `/v1/mail/session` | Create JMAP session with credentials for Stalwart access | Bearer token | ✅ Working |

### 🔄 Implemented but Incomplete (JMAP Client Available)

The following JMAP operations are supported by the Stalwart client but **NOT exposed as Norest REST endpoints**. They are available via direct JMAP communication:

#### Mailbox Operations
- `Mailbox/get` - Retrieve mailboxes/folders
- `Mailbox/query` - Query mailboxes  
- `Mailbox/set` - Create/update/delete mailboxes

#### Email Operations
- `Email/get` - Retrieve email objects
- `Email/query` - Query/filter emails
- `Email/set` - Create/update/delete emails
- `Email/changes` - Get email changes

#### Thread Operations
- `Thread/get` - Retrieve thread information
- `Thread/query` - Query threads

#### Session Operations
- JMAP session discovery
- Capability negotiation
- Account information

### ❌ Not Implemented (REST API Endpoints)

The following mail operations are **NOT available as Norest REST endpoints** and must be performed via direct JMAP communication:

#### Message Operations
- List messages
- Get message details
- Search messages
- Message pagination
- Message filtering
- Message content retrieval

#### Compose & Send
- Send mail
- Create drafts
- Update drafts
- Delete drafts
- Send drafts
- Reply/forward functionality

#### Message State Actions
- Mark as read/unread
- Star/unstar
- Archive
- Move to trash
- Restore from trash
- Mark as spam
- Remove from spam
- Add/remove flags
- Move between folders

#### Thread Operations
- List threads
- Get thread details
- Thread actions
- Thread state changes

#### Attachment Operations
- Upload attachments
- Download attachments
- Get attachment metadata
- Delete attachments

#### Mailbox/Folder Operations
- Create folders
- Rename folders
- Delete folders
- Get folder counts
- Folder state synchronization

#### Search Operations
- Full-text search
- Advanced filtering
- Search by sender/recipient
- Search by date range
- Search by subject/body

## Architecture Pattern

```text
Client Application
        ↓
Norest REST API (Authentication, Provisioning, Session Creation)
        ↓
Stalwart JMAP API (Mail Operations using JMAP credentials)
        ↓
Mail Data Storage
```

## Current Workflow

1. **User Registration**: `POST /v1/auth/register`
2. **Provisioning Check**: `GET /v1/mail/provisioning-status` (poll until ready)
3. **Session Creation**: `POST /v1/mail/session` → Returns JMAP credentials
4. **Mail Operations**: Direct JMAP communication with Stalwart

## Postman Collection Structure

### Folder 06: Mail Actions (Current Norest APIs)
- **Get Mail Account** - Metadata retrieval
- **Check Provisioning Status** - Readiness check  
- **Create Mail Session** - JMAP credential generation

### Folder 07: JMAP Examples (Direct Stalwart Communication)
- **JMAP - Get Session** - Capability discovery
- **JMAP - Mailbox/get** - Folder listing
- **JMAP - Email/query** - Message querying
- **JMAP - Email/get** - Message retrieval

## Environment Variables

### Existing Variables
- `base_url` - Norest API base URL
- `access_token` - JWT authentication token
- `mail_account_id` - Mailbox ID
- `provisioning_status` - Current provisioning status

### New Mail Session Variables
- `mail_session_provider` - Always "stalwart"
- `mail_session_jmap_url` - JMAP endpoint URL
- `mail_session_access_token` - App password for JMAP auth
- `mail_session_account_id` - Stalwart account ID

## Testing Instructions

### Current Implementation Testing
1. Run the registration flow from existing Postman collection
2. Wait for provisioning to complete (use provisioning status endpoint)
3. Create mail session to get JMAP credentials
4. Use JMAP examples to test direct Stalwart communication
5. Verify mail operations work via JMAP

### Expected Results
- ✅ User registration succeeds
- ✅ Provisioning completes successfully
- ✅ Mail session creation returns valid JMAP credentials
- ✅ JMAP session discovery works
- ✅ JMAP mailbox operations work
- ✅ JMAP email operations work

## Future Development Recommendations

### Option 1: Continue Current Architecture
- Maintain direct JMAP integration
- Provide client libraries for JMAP communication
- Document JMAP patterns extensively
- Focus on Norest control plane features

### Option 2: Add REST API Wrappers
Implement Norest REST endpoints that wrap JMAP operations for simplified client integration:

**Priority 1 - Core Message Operations**
- `GET /v1/mail/messages` - List messages with pagination
- `GET /v1/mail/messages/{id}` - Get message details
- `POST /v1/mail/send` - Send email
- `POST /v1/mail/drafts` - Create draft

**Priority 2 - Message State Operations**
- `POST /v1/mail/messages/{id}/read` - Mark as read
- `POST /v1/mail/messages/{id}/unread` - Mark as unread
- `POST /v1/mail/messages/{id}/star` - Star message
- `POST /v1/mail/messages/{id}/archive` - Archive message
- `POST /v1/mail/messages/{id}/trash` - Move to trash

**Priority 3 - Mailbox Operations**
- `GET /v1/mail/mailboxes` - List mailboxes
- `POST /v1/mail/mailboxes` - Create mailbox
- `PUT /v1/mail/mailboxes/{id}` - Update mailbox
- `DELETE /v1/mail/mailboxes/{id}` - Delete mailbox

**Priority 4 - Advanced Features**
- `POST /v1/mail/search` - Search messages
- Attachment upload/download endpoints
- Thread operations
- Advanced filtering and sorting

## Key Findings

1. **Current State**: Norest implements only 3 mail-related REST endpoints focused on provisioning and session creation
2. **Architecture**: Clear separation between control plane (Norest) and data plane (Stalwart)
3. **JMAP Integration**: Comprehensive JMAP client available but not exposed via REST
4. **Client Expectation**: Clients expected to use JMAP directly for mail operations
5. **Future Path**: Option to add REST wrappers for simplified client integration

## Conclusion

The current Norest implementation provides a solid foundation for mail service provisioning with a clean architectural separation. The updated Postman collection accurately reflects the current implementation state and provides both Norest REST endpoints and JMAP examples for complete mail functionality testing.

For clients requiring simplified REST APIs, future development should focus on implementing REST wrappers around the existing JMAP client functionality, starting with core message operations and expanding to advanced features based on user requirements.