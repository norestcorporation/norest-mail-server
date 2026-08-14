# How to Read Mail

Complete guide for reading and displaying email messages with full content.

## Overview

Reading mail involves:
1. Querying for emails
2. Getting email content with specific properties
3. Rendering the email body
4. Handling different content types (text, HTML, attachments)

## Step-by-Step Guide

### Step 1: Query for Emails

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
          "limit": 20
        },
        "0"
      ]
    ]
  }'
```

### Step 2: Get Email with Preview (List View)

For email list views, request minimal properties:

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
          "ids": ["{{email_id_1}}", "{{email_id_2}}"],
          "properties": [
            "id",
            "from",
            "to",
            "subject",
            "receivedAt",
            "preview",
            "keywords",
            "hasAttachments"
          ]
        },
        "0"
      ]
    ]
  }'
```

**Response:**
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
            "preview": "Hi Alice, just wanted to say hello and check how you're doing...",
            "keywords": {
              "$seen": false,
              "$flagged": false
            },
            "hasAttachments": false
          }
        ]
      },
      "0"
    ]
  ]
}
```

### Step 3: Get Email with Full Content (Detail View)

For reading the full email, request body values:

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
            "blobId",
            "threadId",
            "mailboxIds",
            "from",
            "to",
            "cc",
            "bcc",
            "subject",
            "receivedAt",
            "sentAt",
            "keywords",
            "bodyValues",
            "textBody",
            "htmlBody",
            "attachments",
            "hasAttachments",
            "size"
          ]
        },
        "0"
      ]
    ]
  }'
```

**Response:**
```json
{
  "methodResponses": [
    [
      "Email/get",
      {
        "list": [
          {
            "id": "email123",
            "blobId": "blob456",
            "threadId": "thread789",
            "mailboxIds": {
              "a": true
            },
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
            "cc": [],
            "bcc": [],
            "subject": "Hello from Bob",
            "receivedAt": "2026-08-14T10:30:00Z",
            "sentAt": "2026-08-14T10:30:00Z",
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
                "type": "text/plain",
                "charset": "utf-8"
              }
            ],
            "htmlBody": [],
            "attachments": [],
            "hasAttachments": false,
            "size": 234
          }
        ]
      },
      "0"
    ]
  ]
}
```

## Rendering Email Content

### Text-Only Email

**Display headers:**
```
From: Bob Johnson <bob@example.com>
To: Alice Smith <alice@example.com>
Subject: Hello from Bob
Date: 2026-08-14T10:30:00Z
```

**Display body:**
```
Hi Alice,

Just wanted to say hello and check how you're doing.

Best,
Bob
```

### HTML Email

**Response includes htmlBody:**
```json
{
  "htmlBody": [
    {
      "partId": "b2",
      "type": "text/html",
      "charset": "utf-8"
    }
  ],
  "bodyValues": {
    "b2": {
      "value": "<h1>Hello Alice</h1><p>This is <strong>HTML</strong> content.</p>"
    }
  }
}
```

**Render HTML content safely:**
```html
<div class="email-body">
  <!-- Render b2 value as HTML -->
</div>
```

### Multipart Email (Text + HTML)

**Response includes both:**
```json
{
  "textBody": [
    {
      "partId": "b1",
      "type": "text/plain"
    }
  ],
  "htmlBody": [
    {
      "partId": "b2",
      "type": "text/html"
    }
  ],
  "bodyValues": {
    "b1": {
      "value": "Plain text version"
    },
    "b2": {
      "value": "<p>HTML version</p>"
    }
  }
}
```

**Prefer HTML if available, fallback to text:**
```javascript
if (email.htmlBody && email.htmlBody.length > 0) {
  renderHTML(email.bodyValues[email.htmlBody[0].partId].value);
} else if (email.textBody && email.textBody.length > 0) {
  renderText(email.bodyValues[email.textBody[0].partId].value);
}
```

## Email Headers

### Sender Information

**From:**
```json
"from": [
  {
    "name": "Bob Johnson",
    "email": "bob@example.com"
  }
]
```

**Display as:** `Bob Johnson <bob@example.com>`

### Recipients

**To:**
```json
"to": [
  {
    "name": "Alice Smith",
    "email": "alice@example.com"
  },
  {
    "name": "Carol White",
    "email": "carol@example.com"
  }
]
```

**Display as:** `Alice Smith <alice@example.com>, Carol White <carol@example.com>`

**CC/BCC:** Same format as To

### Date

**Received date:**
```json
"receivedAt": "2026-08-14T10:30:00Z"
```

**Format for display:** `August 14, 2026 at 10:30 AM`

### Subject

```json
"subject": "Hello from Bob"
```

**Display as-is** (may need HTML entity decoding)

## Email Keywords (Flags)

**Available keywords:**
- `$seen`: Read/unread
- `$flagged`: Starred/flagged
- `$answered`: Replied
- `$draft`: Draft
- `$forwarded`: Forwarded
- `$junk`: Spam
- `$phishing`: Phishing

**Check flags:**
```json
"keywords": {
  "$seen": true,
  "$flagged": false,
  "$answered": true
}
```

**Display indicators:**
- Unread: Show envelope icon
- Flagged: Show star icon
- Answered: Show reply icon

## Thread Information

**Thread ID:**
```json
"threadId": "thread789"
```

**Use for:**
- Grouping related emails
- Thread view
- Reply operations

## Attachment Handling

**Check for attachments:**
```json
"hasAttachments": true
```

**Get attachment list:**
```json
"attachments": [
  {
    "blobId": "blob123",
    "name": "document.pdf",
    "size": 12345,
    "type": "application/pdf",
    "cid": null
  }
]
```

**Download attachment:**
```
{{download_url}}/blob123/{{account_id}}
```

## Performance Optimization

### Batch Requests

Get multiple emails in one request:

```json
{
  "methodCalls": [
    [
      "Email/get",
      {
        "accountId": "{{account_id}}",
        "ids": ["email1", "email2", "email3", "email4", "email5"]
      },
      "0"
    ]
  ]
}
```

### Property Selection

Only request properties you need:

**List view (minimal):**
```json
["id", "from", "to", "subject", "receivedAt", "preview", "keywords"]
```

**Detail view (full):**
```json
["id", "from", "to", "cc", "bcc", "subject", "receivedAt", "bodyValues", "textBody", "htmlBody", "attachments"]
```

### Preview vs Full Body

Use `preview` for list views:
- Faster retrieval
- Less bandwidth
- Sufficient for scanning

Use `bodyValues` for detail views:
- Full content
- Complete formatting
- Necessary for reading

## Common Patterns

### Get Single Email

```json
{
  "methodCalls": [
    [
      "Email/get",
      {
        "accountId": "{{account_id}}",
        "ids": ["{{email_id}}"]
      },
      "0"
    ]
  ]
}
```

### Get Thread Emails

```bash
# First get thread ID from an email
THREAD_ID=$(echo $EMAIL | jq -r '.threadId')

# Then query for emails in thread
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {"inMailbox": "'$INBOX_ID'", "hasKeyword": "$threadId:'$THREAD_ID'"}
    }, "0"]]
  }'
```

### Get Unread Emails

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

## Error Handling

**Not found:**
```json
{
  "methodResponses": [
    [
      "Email/get",
      {
        "notFound": ["email123"]
      },
      "0"
    ]
  ]
}
```

**Handle by:**
- Showing "Email not found" message
- Refreshing email list
- Checking if email was deleted

**Encoding problems:**
```json
{
  "bodyValues": {
    "b1": {
      "value": "...",
      "isEncodingProblem": true
    }
  }
}
```

**Handle by:**
- Showing "Encoding error" message
- Offering to view raw source
- Logging the error

## Complete Example

```bash
#!/bin/bash

EMAIL="alice@example.com"
MAIL_TOKEN="app_token"
ACCOUNT_ID="account_id"
JMAP_BASE="http://localhost:8081/jmap"
EMAIL_ID="email123"

# Get email with full content
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/get", {
      "accountId": "'$ACCOUNT_ID'",
      "ids": ["'$EMAIL_ID'"],
      "properties": [
        "id", "from", "to", "subject", "receivedAt",
        "keywords", "bodyValues", "textBody", "htmlBody",
        "attachments", "hasAttachments"
      ]
    }, "0"]]
  }' | jq '.'
```

## Next Steps

- [HOW_TO_MARK_READ.md](HOW_TO_MARK_READ.md) - Mark read/unread
- [HOW_TO_STAR.md](HOW_TO_STAR.md) - Star/flag messages
- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move between folders
- [SEARCH.md](SEARCH.md) - Search emails