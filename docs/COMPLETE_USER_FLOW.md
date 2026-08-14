# Complete User Flow Guide - From Registration to Mail Actions

This guide walks you through the complete user journey from registration to performing mail operations using the Norest Mail API.

## Prerequisites

- **Running Services**: All services must be running (`docker-compose up -d`)
- **Base URLs**:
  - Norest API: `http://localhost:8080`
  - Stalwart JMAP: `http://localhost:8081`

---

## Step 1: Health Check (Optional)

### Purpose
Verify all services are running and healthy.

### API
```
GET http://localhost:8080/health
```

### Authentication
**None** - Public endpoint

### What to Enter
Nothing - just send the request.

### Expected Response
```json
{
  "status": "ok"
}
```

---

## Step 2: Register New User

### Purpose
Create a new user account and get authentication token.

### API
```
POST http://localhost:8080/v1/auth/register
```

### Authentication
**None** - Public endpoint

### What to Enter
```json
{
  "email": "alice@example.com",
  "password": "SecurePassword123!"
}
```

**Requirements**:
- Email: Valid email format
- Password: Minimum 8 characters

### Expected Response
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "status": "pending",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### What to Save
- **`access_token`**: Copy this - you'll need it for all authenticated requests
- **`email`**: Copy this - you'll need it for JMAP authentication

### Important Notes
- User status starts as `"pending"`
- User will become `"active"` after successful mail provisioning
- Save the `access_token` - this is your Norest JWT token

---

## Step 3: Login (Optional - if you already have an account)

### Purpose
Get authentication token for existing user.

### API
```
POST http://localhost:8080/v1/auth/login
```

### Authentication
**None** - Public endpoint

### What to Enter
```json
{
  "email": "alice@example.com",
  "password": "SecurePassword123!"
}
```

### Expected Response
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "status": "active",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### What to Save
- **`access_token`**: Copy this for authenticated requests
- **`email`**: Copy this for JMAP authentication

---

## Step 4: Get Current User Info

### Purpose
Check your user status and information.

### API
```
GET http://localhost:8080/v1/me
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
Add this header to your request:
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

Replace `YOUR_ACCESS_TOKEN` with the token from Step 2 or 3.

### What to Enter
Nothing - just send the request with the authorization header.

### Expected Response
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "status": "pending"
}
```

---

## Step 5: Create Domain

### Purpose
Create a domain for your email addresses.

### API
```
POST http://localhost:8080/v1/domains
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
```json
{
  "name": "example.com"
}
```

**Requirements**:
- Domain name: Valid domain format (e.g., `example.com`)

### Expected Response
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "product_account_id": "789e0123-e456-78d9-a123-456789012345",
  "name": "example.com",
  "status": "pending",
  "verification_status": "pending",
  "created_at": "2026-08-14T10:00:00Z",
  "updated_at": "2026-08-14T10:00:00Z"
}
```

### What to Save
- **`id`**: Copy this - you'll need it to create addresses
- **`name`**: Copy this - you'll need it for email addresses

### Important Notes
- Domain status will be `"pending"` initially
- Background worker will provision the domain in Stalwart
- Wait ~30 seconds for provisioning to complete

---

## Step 6: Wait for Domain Provisioning

### Purpose
Allow background workers to provision the domain in Stalwart.

### What to Do
Wait approximately **30 seconds** after creating the domain.

### Verify Provisioning
Check domain status:
```
GET http://localhost:8080/v1/domains/DOMAIN_ID
```

Replace `DOMAIN_ID` with the ID from Step 5.

### Expected Status Change
- Initial: `"status": "pending"`
- After provisioning: `"status": "active"`

---

## Step 7: Create Email Address

### Purpose
Create an email address for your domain.

### API
```
POST http://localhost:8080/v1/domains/DOMAIN_ID/addresses
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
```json
{
  "local_part": "alice"
}
```

**Requirements**:
- `local_part`: The username part before `@` (e.g., `alice` for `alice@example.com`)
- Replace `DOMAIN_ID` in URL with your domain ID from Step 5

### Expected Response
```json
{
  "id": "abc12345-def67-8901-abcd-ef1234567890",
  "domain_id": "123e4567-e89b-12d3-a456-426614174000",
  "local_part": "alice",
  "status": "reserved",
  "created_at": "2026-08-14T10:01:00Z",
  "updated_at": "2026-08-14T10:01:00Z"
}
```

### What to Save
- **`id`**: Address ID (optional - for address management)
- Full email will be: `alice@example.com`

### Important Notes
- This automatically creates a mailbox record
- Background worker will provision the Stalwart account
- User status will transition from `"pending"` to `"active"` after successful provisioning

---

## Step 8: Wait for Account Provisioning

### Purpose
Allow background workers to provision the Stalwart account.

### What to Do
Wait approximately **30 seconds** after creating the address.

### Verify Provisioning
Check your user status:
```
GET http://localhost:8080/v1/me
```

### Expected Status Change
- Initial: `"status": "pending"`
- After provisioning: `"status": "active"`

### Verify Mailbox Ready
Check mail account:
```
GET http://localhost:8080/v1/mail/account
```

---

## Step 9: Create Mail Session

### Purpose
Get JMAP credentials (AppPassword) for mail operations.

### API
```
POST http://localhost:8080/v1/mail/session
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
Nothing - just send the request with the authorization header.

### Expected Response
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "c"
}
```

### What to Save
- **`access_token`**: This is your JMAP AppPassword - **NOT** the same as Norest token
- **`account_id`**: This is the Stalwart account ID for JMAP operations
- **`jmap_session_url`**: JMAP endpoint URL

### Important Notes
- This `access_token` is different from your Norest JWT token
- This token is used for JMAP authentication, not Norest API
- The `account_id` is the Stalwart account ID (not your Norest user ID)

---

## Step 10: Get Mailbox Information

### Purpose
Retrieve your current mailbox details.

### API
```
GET http://localhost:8080/v1/mail/account
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
Nothing - just send the request with the authorization header.

### Expected Response
```json
{
  "id": "xyz98765-abcde-4321-fghi-987654321098",
  "address_id": "abc12345-def67-8901-abcd-ef1234567890",
  "status": "active",
  "stalwart_account_id": "c"
}
```

---

## Step 11: JMAP Session Discovery

### Purpose
Discover JMAP capabilities and endpoints.

### API
```
GET http://localhost:8081/.well-known/jmap
```

### Authentication
**None** - Public endpoint

### What to Enter
Nothing - just send the request.

### Expected Response
```json
{
  "capabilities": {
    "urn:ietf:params:jmap:core": {},
    "urn:ietf:params:jmap:mail": {}
  },
  "apiUrl": "http://localhost:8081/jmap",
  "downloadUrl": "http://localhost:8081/download/{accountId}/{blobId}/{name}?type={type}"
}
```

---

## Step 12: Get Mailboxes (JMAP)

### Purpose
List all mailboxes (Inbox, Drafts, Trash, etc.).

### API
```
POST http://localhost:8081/jmap
```

### Authentication
**Required**: HTTP Basic Auth

### How to Authenticate
Use HTTP Basic Authentication with:
- **Username**: Your email address (e.g., `alice@example.com`)
- **Password**: Your JMAP AppPassword from Step 9

### What to Enter
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
        "accountId": "YOUR_ACCOUNT_ID"
      },
      "0"
    ]
  ]
}
```

Replace `YOUR_ACCOUNT_ID` with the `account_id` from Step 9.

### Expected Response
```json
{
  "methodResponses": [
    [
      "Mailbox/get",
      {
        "list": [
          {
            "id": "inbox_id",
            "name": "Inbox",
            "role": "inbox"
          },
          {
            "id": "drafts_id",
            "name": "Drafts",
            "role": "drafts"
          },
          {
            "id": "trash_id",
            "name": "Trash",
            "role": "trash"
          }
        ]
      },
      "0"
    ]
  ]
}
```

### What to Save
- **Inbox ID**: For querying emails
- **Drafts ID**: For draft operations
- **Trash ID**: For trash operations

---

## Step 13: Query Emails (JMAP)

### Purpose
List emails in a specific mailbox (e.g., Inbox).

### API
```
POST http://localhost:8081/jmap
```

### Authentication
**Required**: HTTP Basic Auth

### How to Authenticate
Same as Step 12:
- **Username**: Your email address
- **Password**: Your JMAP AppPassword

### What to Enter
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:mail"
  ],
  "methodCalls": [
    [
      "Email/query",
      {
        "accountId": "YOUR_ACCOUNT_ID",
        "filter": {
          "inMailbox": "YOUR_INBOX_ID"
        },
        "limit": 10
      },
      "0"
    ]
  ]
}
```

Replace:
- `YOUR_ACCOUNT_ID` with account ID from Step 9
- `YOUR_INBOX_ID` with inbox ID from Step 12

### Expected Response
```json
{
  "methodResponses": [
    [
      "Email/query",
      {
        "ids": ["email_id_1", "email_id_2"],
        "queryState": "state_value"
      },
      "0"
    ]
  ]
}
```

### What to Save
- **Email IDs**: For fetching individual emails

---

## Step 14: Get Email Content (JMAP)

### Purpose
Retrieve full email content.

### API
```
POST http://localhost:8081/jmap
```

### Authentication
**Required**: HTTP Basic Auth

### How to Authenticate
Same as Step 12.

### What to Enter
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:mail"
  ],
  "methodCalls": [
    [
      "Email/get",
      {
        "accountId": "YOUR_ACCOUNT_ID",
        "ids": ["YOUR_EMAIL_ID"],
        "properties": [
          "id",
          "from",
          "to",
          "subject",
          "receivedAt",
          "preview",
          "keywords",
          "bodyValues"
        ]
      },
      "0"
    ]
  ]
}
```

Replace:
- `YOUR_ACCOUNT_ID` with account ID from Step 9
- `YOUR_EMAIL_ID` with email ID from Step 13

### Expected Response
```json
{
  "methodResponses": [
    [
      "Email/get",
      {
        "list": [
          {
            "id": "email_id_1",
            "from": [{ "email": "sender@example.com", "name": "Sender" }],
            "to": [{ "email": "alice@example.com", "name": "Alice" }],
            "subject": "Test Email",
            "receivedAt": "2026-08-14T10:00:00Z",
            "preview": "Email preview text...",
            "keywords": { "$seen": true },
            "bodyValues": {
              "body_id": {
                "value": "Full email body...",
                "isEncodingProblem": false
              }
            }
          }
        ]
      },
      "0"
    ]
  ]
}
```

---

## Step 15: Mark Email as Read (JMAP)

### Purpose
Mark an email as read.

### API
```
POST http://localhost:8081/jmap
```

### Authentication
**Required**: HTTP Basic Auth

### How to Authenticate
Same as Step 12.

### What to Enter
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:mail"
  ],
  "methodCalls": [
    [
      "Email/set",
      {
        "accountId": "YOUR_ACCOUNT_ID",
        "update": {
          "YOUR_EMAIL_ID": {
            "keywords/$seen": true
          }
        }
      },
      "0"
    ]
  ]
}
```

Replace:
- `YOUR_ACCOUNT_ID` with account ID from Step 9
- `YOUR_EMAIL_ID` with email ID from Step 13

---

## Step 16: Get Usage Information

### Purpose
Check your account usage and limits.

### API
```
GET http://localhost:8080/v1/account/usage
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
Nothing - just send the request with the authorization header.

### Expected Response
```json
{
  "storage_used_bytes": 1024000,
  "storage_limit_bytes": 1073741824,
  "domains_used": 1,
  "domains_limit": 10,
  "mailboxes_used": 1,
  "mailboxes_limit": 25
}
```

---

## Step 17: List Domains

### Purpose
List all your domains.

### API
```
GET http://localhost:8080/v1/domains
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
Nothing - just send the request with the authorization header.

### Expected Response
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "example.com",
    "status": "active",
    "verification_status": "verified",
    "created_at": "2026-08-14T10:00:00Z"
  }
]
```

---

## Step 18: List Addresses

### Purpose
List all email addresses for a domain.

### API
```
GET http://localhost:8080/v1/domains/DOMAIN_ID/addresses
```

### Authentication
**Required**: Bearer Token

### How to Authenticate
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

### What to Enter
Replace `DOMAIN_ID` with your domain ID from Step 5.

### Expected Response
```json
[
  {
    "id": "abc12345-def67-8901-abcd-ef1234567890",
    "domain_id": "123e4567-e89b-12d3-a456-426614174000",
    "local_part": "alice",
    "status": "active",
    "created_at": "2026-08-14T10:01:00Z"
  }
]
```

---

## Authentication Summary

### Norest API Authentication
- **Method**: Bearer Token (JWT)
- **Header**: `Authorization: Bearer YOUR_NOREST_TOKEN`
- **How to Get**: Register or Login via `/v1/auth/register` or `/v1/auth/login`
- **Used For**: All Norest API endpoints

### JMAP Authentication
- **Method**: HTTP Basic Auth
- **Username**: Your email address (e.g., `alice@example.com`)
- **Password**: JMAP AppPassword from `/v1/mail/session`
- **How to Get**: Create mail session via `/v1/mail/session`
- **Used For**: All JMAP operations

---

## Quick Reference Card

### Tokens to Save
1. **Norest Access Token**: From registration/login - for Norest API
2. **JMAP AppPassword**: From mail session - for JMAP operations
3. **Account ID**: From mail session - for JMAP operations
4. **Domain ID**: From domain creation - for address management
5. **Inbox ID**: From mailbox list - for email queries

### IDs to Remember
- **User ID**: Your Norest user UUID
- **Domain ID**: Your domain UUID
- **Address ID**: Your email address UUID
- **Stalwart Account ID**: Short ID for JMAP (e.g., "c", "f", "h")

### Email Address Format
- Full email: `local_part@domain_name`
- Example: `alice@example.com`

---

## Common Issues and Solutions

### Issue: User status remains "pending"
**Solution**: Wait 30 seconds after creating address for provisioning to complete.

### Issue: Domain status remains "pending"
**Solution**: Wait 30 seconds for domain provisioning in Stalwart.

### Issue: Mail session creation fails
**Solution**: Ensure user status is "active" and mailbox is fully provisioned.

### Issue: JMAP authentication fails
**Solution**: 
- Verify you're using email address as username
- Verify you're using AppPassword (not Norest token) as password
- Verify account_id is correct from mail session response

### Issue: Invalid credentials error
**Solution**: Check that your Norest token hasn't expired and is valid.

---

## Complete cURL Example

```bash
# 1. Register
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"SecurePassword123!"}'

# Save: access_token, email

# 2. Create domain
export TOKEN="your_access_token"
curl -X POST http://localhost:8080/v1/domains \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'

# Save: domain_id

# 3. Wait 30 seconds for domain provisioning

# 4. Create address
export DOMAIN_ID="your_domain_id"
curl -X POST http://localhost:8080/v1/domains/$DOMAIN_ID/addresses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"local_part":"alice"}'

# 5. Wait 30 seconds for account provisioning

# 6. Check user status (should be "active")
curl -X GET http://localhost:8080/v1/me \
  -H "Authorization: Bearer $TOKEN"

# 7. Create mail session
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer $TOKEN"

# Save: access_token (AppPassword), account_id

# 8. Get mailboxes
export EMAIL="alice@example.com"
export APP_PASSWORD="your_app_password"
export ACCOUNT_ID="your_account_id"
curl -X POST http://localhost:8081/jmap \
  -u "$EMAIL:$APP_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "$ACCOUNT_ID"}, "0"]]
  }'

# Save: inbox_id

# 9. Query emails
curl -X POST http://localhost:8081/jmap \
  -u "$EMAIL:$APP_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {"accountId": "$ACCOUNT_ID", "filter": {"inMailbox": "$inbox_id"}}, "0"]]
  }'
```

---

## Postman Collection Usage

The Postman collection `Norest-Mail.postman_collection.json` is organized with:

1. **01 Health**: Service health checks
2. **02 Authentication**: User registration and login
3. **03 Domains**: Domain management
4. **04 Addresses**: Email address management
5. **05 Mail Session**: JMAP session creation
6. **06 Usage**: Account usage information
7. **07 JMAP - Session**: JMAP discovery
8. **08 JMAP - Mailboxes**: Mailbox operations
9. **09 JMAP - Emails**: Email operations

### Postman Variables
The collection automatically sets these variables:
- `norest_access_token`: Norest JWT token
- `mail_access_token`: JMAP AppPassword
- `account_id`: Stalwart account ID
- `domain_id`: Domain UUID
- `email_address`: Full email address
- `inbox_id`: Inbox mailbox ID

Just run the requests in order and the collection will handle the variable management automatically.