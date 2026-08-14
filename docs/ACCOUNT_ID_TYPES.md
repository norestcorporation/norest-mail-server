# Account ID Types and Authentication Flow

## Account ID Types

### 1. Norest User ID
- **Table**: `users.id`
- **Type**: UUID
- **Purpose**: Platform user identity for authentication
- **Used by**: Norest API authentication, JWT tokens
- **Example**: `9bd66203-4578-4b75-b190-81c6e4b46b68`

### 2. Norest Product Account ID
- **Table**: `product_accounts.id`
- **Type**: UUID
- **Purpose**: Billing and subscription entity
- **Used by**: Product layer, billing integration
- **Example**: `6ebadd5b-79b5-4831-82dc-ffc374f3b947`

### 3. Norest Domain ID
- **Table**: `domains.id`
- **Type**: UUID
- **Purpose**: Domain control-plane entity
- **Used by**: Domain management, provisioning
- **Example**: `ca85be68-f7ad-4af3-86ea-3085f2b3b575`

### 4. Norest Address ID
- **Table**: `addresses.id`
- **Type**: UUID
- **Purpose**: Email address control-plane entity
- **Used by**: Address management, provisioning
- **Example**: `e49b92db-4d55-4680-a150-6abb3c6d5e13`

### 5. Norest Mailbox ID
- **Table**: `mailboxes.id`
- **Type**: UUID
- **Purpose**: Links address to Stalwart account
- **Used by**: Mail session creation, provisioning
- **Example**: `4950225a-7e1b-42f9-a6b4-8cf417256859`

### 6. Stalwart Domain ID
- **Table**: `domains.stalwart_domain_id`
- **Type**: TEXT
- **Purpose**: Stalwart domain identifier
- **Used by**: Stalwart management operations
- **Example**: `c`, `e`, `h`, `i`

### 7. Stalwart Account ID
- **Table**: `mailboxes.stalwart_account_id`
- **Type**: TEXT
- **Purpose**: Stalwart account identifier for JMAP operations
- **Used by**: JMAP authentication, mail operations
- **Example**: `c`, `f`, `h`, `i`

## ID Usage by Layer

### Norest API Layer
- **User ID**: Authentication tokens, `/v1/me` endpoint
- **Domain ID**: Domain management endpoints
- **Address ID**: Address management endpoints
- **Mailbox ID**: Mail session creation (internal)

### JMAP Layer
- **Stalwart Account ID**: JMAP `accountId` parameter
- **Email Address**: JMAP Basic Auth username
- **App Password**: JMAP Basic Auth password

## Current API Exposure

### Existing Endpoints
- `GET /v1/me` - Returns Norest User ID, email, status
- `POST /v1/mail/session` - Returns Stalwart Account ID and AppPassword
- `GET /v1/mail/account` - Returns mailbox information

### Account ID Retrieval

The Stalwart Account ID is already exposed through the mail session endpoint:

```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer {{norest_access_token}}"
```

Response:
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "c"
}
```

The `account_id` in the response is the **Stalwart Account ID** required for JMAP operations.

## ID Flow for Mail Operations

1. **User Registration**: Creates Norest User ID
2. **Domain Creation**: Creates Norest Domain ID + Stalwart Domain ID
3. **Address Creation**: Creates Norest Address ID + Norest Mailbox ID
4. **Account Provisioning**: Updates Norest Mailbox ID with Stalwart Account ID
5. **Mail Session**: Returns Stalwart Account ID + AppPassword
6. **JMAP Operations**: Use Stalwart Account ID with AppPassword

## No Changes Required

The current API already correctly exposes the Stalwart Account ID through the mail session endpoint. No additional API changes are needed for account ID exposure.