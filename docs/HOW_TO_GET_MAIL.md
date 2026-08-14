# How to Get Mail

Complete guide for retrieving email messages using Norest Mail and JMAP.

## Overview

To retrieve email messages, you need to:
1. Authenticate with Norest
2. Create a mail session
3. Connect to Stalwart JMAP
4. Get mailboxes
5. Query emails
6. Retrieve email content

## Step-by-Step Guide

### Step 1: Register/Login to Norest

**Register new user:**
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123!"
  }'
```

**Or login existing user:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123!"
  }'
```

**Save the access token from response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Step 2: Create Domain

```bash
curl -X POST http://localhost:8080/v1/domains \
  -H "Authorization: Bearer {{norest_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "example.com"
  }'
```

**Save the domain ID from response:**
```json
{
  "id": "domain-uuid",
  "name": "example.com",
  "status": "pending"
}
```

### Step 3: Create Address/Mailbox

```bash
curl -X POST http://localhost:8080/v1/domains/{{domain_id}}/addresses \
  -H "Authorization: Bearer {{norest_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "local_part": "alice"
  }'
```

This creates the address `alice@example.com` and triggers mailbox provisioning.

### Step 4: Wait for Provisioning

Poll the mailbox status until it's active:

```bash
curl -X GET http://localhost:8080/v1/mail/account \
  -H "Authorization: Bearer {{norest_access_token}}"
```

**Expected response when ready:**
```json
{
  "address": "alice",
  "status": "active"
}
```

### Step 5: Get Mail Session

```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer {{norest_access_token}}"
```

**Save the session response:**
```json
{
  "provider": "stalwart",
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "stalwart_account_id"
}
```

**Variables to save:**
- `{{mail_access_token}}`: `app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta`
- `{{account_id}}`: `stalwart_account_id`
- `{{jmap_url}}`: `http://localhost:8081/jmap`
- `{{email_address}}`: `alice@example.com`

### Step 6: Discover JMAP Session

```bash
curl http://localhost:8081/.well-known/jmap
```

**Response**:
```json
{
  "accounts": {
    "stalwart_account_id": {
      "name": "alice@example.com",
      "isPersonal": true,
      "isReadOnly": false
    }
  },
  "apiUrl": "http://localhost:8081/jmap",
  "downloadUrl": "http://localhost:8081/download/{blobId}/{accountId}",
  "uploadUrl": "http://localhost:8081/upload/{accountId}"
}
```

### Step 7: Get Mailboxes

```bash
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:mail"
    ],
    "methodCalls": [
      [
        "Mailbox/get",
        {
          "accountId": "{{account_id}}"
        },
        "0"
      ]
    ]
  }'
```

**Response**:
```json
{
  "methodResponses": [
    [
      "Mailbox/get",
      {
        "list": [
          {
            "id": "a",
            "name": "Inbox",
            "role": "inbox",
            "totalEmails": 10,
            "unreadEmails": 3
          },
          {
            "id": "b",
            "name": "Drafts",
            "role": "drafts",
            "totalEmails": 2,
            "unreadEmails": 0
          },
          {
            "id": "c",
            "name": "Sent",
            "role": "sent",
            "totalEmails": 15,
            "unreadEmails": 0
          },
          {
            "id": "d",
            "name": "Trash",
            "role": "trash",
            "totalEmails": 5,
            "unreadEmails": 0
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Save the Inbox ID:**
- `{{inbox_id}}`: `a` (the ID where `role` is `inbox`)

### Step 8: Query Emails in Inbox

```bash
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:mail"
    ],
    "methodCalls": [
      [
        "Email/query",
        {
          "accountId": "{{account_id}}",
          "filter": {
            "inMailbox": "{{inbox_id}}"
          },
          "limit": 20,
          "sort": [
            {
              "property": "receivedAt",
              "isAscendingDescending": "DESC"
            }
          ]
        },
        "0"
      ]
    ]
  }'
```

**Response**:
```json
{
  "methodResponses": [
    [
      "Email/query",
      {
        "queryState": "q1",
        "ids": [
          "email123",
          "email456",
          "email789"
        ],
        "total": 10,
        "limit": 20
      },
      "0"
    ]
  ]
}
```

**Save email IDs:**
- `{{email_id_1}}`: `email123`
- `{{email_id_2}}`: `email456`
- etc.

### Step 9: Get Email Content

```bash
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:mail"
    ],
    "methodCalls": [
      [
        "Email/get",
        {
          "accountId": "{{account_id}}",
          "ids": ["{{email_id_1}}"],
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
  }'
```

**Response**:
```json
{
  "methodResponses": [
    [
      "Email/get",
      {
        "list": [
          {
            "id": "email123",
            "from": [
              {
                "name": "Bob Johnson",
                "email": "bob@example.com"
              }
            ],
            "to": [
              {
                "name": "Alice Smith",
                "email": "alice@example.com"
              }
            ],
            "subject": "Hello from Bob",
            "receivedAt": "2026-08-14T10:30:00Z",
            "preview": "Hi Alice, just wanted to say hello...",
            "keywords": {
              "$seen": false,
              "$flagged": false
            },
            "bodyValues": {
              "b1": {
                "value": "Hi Alice,\n\nJust wanted to say hello and check how you're doing.\n\nBest,\nBob",
                "isEncodingProblem": false,
                "isTruncated": false
              }
            },
            "textBody": [
              {
                "partId": "b1",
                "type": "text/plain"
              }
            ]
          }
        ]
      },
      "0"
    ]
  ]
}
```

### Step 10: Render the Email

Use the returned data to display the email:

**Display metadata:**
- From: `Bob Johnson <bob@example.com>`
- To: `Alice Smith <alice@example.com>`
- Subject: `Hello from Bob`
- Date: `2026-08-14T10:30:00Z`
- Unread: `true` (from `$seen: false`)

**Display body:**
```
Hi Alice,

Just wanted to say hello and check how you're doing.

Best,
Bob
```

## Complete Example Script

```bash
#!/bin/bash

# Variables
API_BASE="http://localhost:8080/v1"
JMAP_BASE="http://localhost:8081/jmap"
EMAIL="alice@example.com"
PASSWORD="SecurePassword123!"

# 1. Login
echo "Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST $API_BASE/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
echo "Token: $TOKEN"

# 2. Get mail session
echo "Getting mail session..."
SESSION_RESPONSE=$(curl -s -X POST $API_BASE/mail/session \
  -H "Authorization: Bearer $TOKEN")

MAIL_TOKEN=$(echo $SESSION_RESPONSE | jq -r '.access_token')
ACCOUNT_ID=$(echo $SESSION_RESPONSE | jq -r '.account_id')
echo "Mail token: $MAIL_TOKEN"
echo "Account ID: $ACCOUNT_ID"

# 3. Get mailboxes
echo "Getting mailboxes..."
MAILBOXES=$(curl -s -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "'$ACCOUNT_ID'"}, "0"]]
  }')

INBOX_ID=$(echo $MAILBOXES | jq -r '.methodResponses[0][1].list[] | select(.role == "inbox") | .id')
echo "Inbox ID: $INBOX_ID"

# 4. Query emails
echo "Querying emails..."
EMAILS=$(curl -s -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {"inMailbox": "'$INBOX_ID'"},
      "limit": 10
    }, "0"]]
  }')

EMAIL_IDS=$(echo $EMAILS | jq -r '.methodResponses[0][1].ids[]')
echo "Email IDs: $EMAIL_IDS"

# 5. Get first email
FIRST_EMAIL_ID=$(echo $EMAIL_IDS | head -n 1)
echo "Getting email: $FIRST_EMAIL_ID"

EMAIL_CONTENT=$(curl -s -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/get", {
      "accountId": "'$ACCOUNT_ID'",
      "ids": ["'$FIRST_EMAIL_ID'"]
    }, "0"]]
  }')

echo "Email content:"
echo $EMAIL_CONTENT | jq '.'
```

## Common Patterns

### Get All Emails (All Mailboxes)

```json
{
  "methodCalls": [
    [
      "Email/query",
      {
        "accountId": "{{account_id}}",
        "limit": 50
      },
      "0"
    ]
  ]
}
```

### Get Unread Emails Only

```json
{
  "methodCalls": [
    [
      "Email/query",
      {
        "accountId": "{{account_id}}",
        "filter": {
          "inMailbox": "{{inbox_id}}",
          "hasKeyword": "$seen"
        }
      },
      "0"
    ]
  ]
}
```

### Get Flagged Emails Only

```json
{
  "methodCalls": [
    [
      "Email/query",
      {
        "accountId": "{{account_id}}",
        "filter": {
          "hasKeyword": "$flagged"
        }
      },
      "0"
    ]
  ]
}
```

### Get Emails from Specific Sender

```json
{
  "methodCalls": [
    [
      "Email/query",
      {
        "accountId": "{{account_id}}",
        "filter": {
          "from": "bob@example.com"
        }
      },
      "0"
    ]
  ]
}
```

## Pagination

For large mailboxes, use pagination:

```json
{
  "methodCalls": [
    [
      "Email/query",
      {
        "accountId": "{{account_id}}",
        "filter": {
          "inMailbox": "{{inbox_id}}"
        },
        "limit": 20,
        "position": 0
      },
      "0"
    ]
  ]
}
```

To get the next page, increment `position` by the number of results.

## Performance Tips

1. **Use properties parameter**: Only request the properties you need
2. **Limit results**: Use `limit` to avoid fetching too many emails
3. **Use preview**: Use `preview` for list views instead of full body
4. **Batch requests**: Get multiple emails in a single Email/get call
5. **Filter early**: Use JMAP filters instead of client-side filtering

## Error Handling

**Common errors:**

1. **Authentication failed**:
   - Check mail access token is valid
   - Verify email address is correct
   - Ensure account is not suspended

2. **Account not found**:
   - Verify account_id from mail session
   - Check mailbox provisioning completed

3. **Mailbox not found**:
   - Verify mailbox ID from Mailbox/get
   - Check mailbox exists for account

4. **Rate limit exceeded**:
   - Reduce request frequency
   - Implement exponential backoff

## Next Steps

- [HOW_TO_SEND_MAIL.md](HOW_TO_SEND_MAIL.md) - Send messages
- [HOW_TO_READ_MAIL.md](HOW_TO_READ_MAIL.md) - Read message details
- [HOW_TO_MARK_READ.md](HOW_TO_MARK_READ.md) - Mark read/unread
- [HOW_TO_STAR.md](HOW_TO_STAR.md) - Star/flag messages
- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move between folders
- [HOW_TO_DELETE_MAIL.md](HOW_TO_DELETE_MAIL.md) - Delete messages