# How to Mark Read/Unread

Complete guide for marking emails as read or unread using JMAP.

## Overview

Marking emails as read/unread uses the JMAP `Email/set` method with the `$seen` keyword.

## Mark as Read

### Single Email

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
        "Email/set",
        {
          "accountId": "{{account_id}}",
          "update": {
            "{{email_id}}": {
              "keywords/$seen": true
            }
          }
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
      "Email/set",
      {
        "updated": {
          "{{email_id}}": {}
        },
        "state": "new_state"
      },
      "0"
    ]
  ]
}
```

### Multiple Emails

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
        "Email/set",
        {
          "accountId": "{{account_id}}",
          "update": {
            "{{email_id_1}}": {
              "keywords/$seen": true
            },
            "{{email_id_2}}": {
              "keywords/$seen": true
            },
            "{{email_id_3}}": {
              "keywords/$seen": true
            }
          }
        },
        "0"
      ]
    ]
  }'
```

### All Emails in Mailbox

```bash
# First get all email IDs in mailbox
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {"inMailbox": "'$INBOX_ID'"}
    }, "0"]]
  }'

# Then mark all as read
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "email1": {"keywords/$seen": true},
        "email2": {"keywords/$seen": true},
        "email3": {"keywords/$seen": true}
      }
    }, "0"]]
  }'
```

## Mark as Unread

### Single Email

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
        "Email/set",
        {
          "accountId": "{{account_id}}",
          "update": {
            "{{email_id}}": {
              "keywords/$seen": false
            }
          }
        },
        "0"
      ]
    ]
  }'
```

### Multiple Emails

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
        "Email/set",
        {
          "accountId": "{{account_id}}",
          "update": {
            "{{email_id_1}}": {
              "keywords/$seen": false
            },
            "{{email_id_2}}": {
              "keywords/$seen": false
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Toggle Read Status

To toggle read status (if unread, mark read; if read, mark unread):

```bash
# First get current status
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/get", {
      "accountId": "'$ACCOUNT_ID'",
      "ids": ["'$EMAIL_ID'"],
      "properties": ["keywords"]
    }, "0"]]
  }'

# Check if $seen is true or false, then toggle
# If $seen is true, set to false. If false, set to true.
```

## Check Read Status

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
          "ids": ["{{email_id}}"],
          "properties": ["keywords"]
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
            "keywords": {
              "$seen": true,
              "$flagged": false
            }
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Check status:**
```javascript
const isRead = email.keywords.$seen === true;
```

## Common Patterns

### Mark as Read on Open

When a user opens an email to read it:

```bash
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {"keywords/$seen": true}
      }
    }, "0"]]
  }'
```

### Mark All as Read in Inbox

Useful for "Mark all as read" button:

```bash
# Get all unread emails
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {
        "inMailbox": "'$INBOX_ID'",
        "hasKeyword": "$seen"
      }
    }, "0"]]
  }'

# Mark all as read
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "email1": {"keywords/$seen": true},
        "email2": {"keywords/$seen": true}
      }
    }, "0"]]
  }'
```

### Mark as Unread for Follow-up

User marks email as unread to follow up later:

```bash
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {"keywords/$seen": false}
      }
    }, "0"]]
  }'
```

## Combined Operations

### Mark as Read and Flag

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
        "Email/set",
        {
          "accountId": "{{account_id}}",
          "update": {
            "{{email_id}}": {
              "keywords/$seen": true,
              "keywords/$flagged": true
            }
          }
        },
        "0"
      ]
    ]
  }'
```

### Mark as Unread and Unflag

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
        "Email/set",
        {
          "accountId": "{{account_id}}",
          "update": {
            "{{email_id}}": {
              "keywords/$seen": false,
              "keywords/$flagged": false
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Error Handling

**Email not found:**
```json
{
  "methodResponses": [
    [
      "Email/set",
      {
        "notUpdated": {
          "{{email_id}}": {
            "type": "notFound"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Handle by:**
- Refreshing email list
- Verifying email ID is correct
- Checking if email was deleted

## Complete Example

```bash
#!/bin/bash

EMAIL="alice@example.com"
MAIL_TOKEN="app_token"
ACCOUNT_ID="account_id"
JMAP_BASE="http://localhost:8081/jmap"
EMAIL_ID="email123"

# Mark as read
echo "Marking email as read..."
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {"keywords/$seen": true}
      }
    }, "0"]]
  }' | jq '.'

# Verify status
echo "Verifying read status..."
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/get", {
      "accountId": "'$ACCOUNT_ID'",
      "ids": ["'$EMAIL_ID'"],
      "properties": ["keywords"]
    }, "0"]]
  }' | jq '.'
```

## Next Steps

- [HOW_TO_STAR.md](HOW_TO_STAR.md) - Star/flag messages
- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move between folders
- [HOW_TO_DELETE_MAIL.md](HOW_TO_DELETE_MAIL.md) - Delete messages