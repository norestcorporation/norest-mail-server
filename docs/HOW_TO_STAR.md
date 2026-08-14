# How to Star/Flag Messages

Complete guide for starring or flagging emails using JMAP.

## Overview

Starring/flagging emails uses the JMAP `Email/set` method with the `$flagged` keyword.

## Star/Flag Email

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
              "keywords/$flagged": true
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
              "keywords/$flagged": true
            },
            "{{email_id_2}}": {
              "keywords/$flagged": true
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Unstar/Unflag Email

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
              "keywords/$flagged": false
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Toggle Star Status

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

# Toggle based on current status
```

## Query Flagged Emails

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
            "hasKeyword": "$flagged"
          }
        },
        "0"
      ]
    ]
  }'
```

## Common Patterns

### Star for Follow-up

User stars email to follow up later:

```bash
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {"keywords/$flagged": true}
      }
    }, "0"]]
  }'
```

### View Starred Emails

Show all starred emails:

```bash
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {"hasKeyword": "$flagged"}
    }, "0"]]
  }'
```

## Complete Example

```bash
#!/bin/bash

EMAIL="alice@example.com"
MAIL_TOKEN="app_token"
ACCOUNT_ID="account_id"
JMAP_BASE="http://localhost:8081/jmap"
EMAIL_ID="email123"

# Star email
echo "Starring email..."
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {"keywords/$flagged": true}
      }
    }, "0"]]
  }' | jq '.'
```

## Next Steps

- [HOW_TO_MARK_READ.md](HOW_TO_MARK_READ.md) - Mark read/unread
- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move between folders