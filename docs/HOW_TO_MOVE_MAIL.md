# How to Move Mail

Complete guide for moving emails between mailboxes/folders using JMAP.

## Overview

Moving emails between folders uses the JMAP `Email/set` method by updating the `mailboxIds` property.

## Move Email to Different Mailbox

### Move to Trash

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
              "mailboxIds": {
                "{{trash_id}}": true,
                "{{inbox_id}}": false
              }
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

### Move to Archive

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
              "mailboxIds": {
                "{{archive_id}}": true,
                "{{inbox_id}}": false
              }
            }
          }
        },
        "0"
      ]
    ]
  }'
```

### Move to Custom Folder

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
              "mailboxIds": {
                "{{custom_folder_id}}": true,
                "{{inbox_id}}": false
              }
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Move Multiple Emails

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
              "mailboxIds": {
                "{{trash_id}}": true,
                "{{inbox_id}}": false
              }
            },
            "{{email_id_2}}": {
              "mailboxIds": {
                "{{trash_id}}": true,
                "{{inbox_id}}": false
              }
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Copy to Multiple Mailboxes

JMAP allows emails to exist in multiple mailboxes simultaneously:

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
              "mailboxIds": {
                "{{inbox_id}}": true,
                "{{archive_id}}": true
              }
            }
          }
        },
        "0"
      ]
    ]
  }'
```

**Note:** This keeps the email in both Inbox and Archive.

## Remove from All Mailboxes (Delete)

To remove an email from all mailboxes (effectively delete):

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
              "mailboxIds": {}
            }
          }
        },
        "0"
      ]
    ]
  }'
```

## Common Patterns

### Move from Inbox to Trash

```bash
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {
          "mailboxIds": {
            "'$TRASH_ID'": true,
            "'$INBOX_ID'": false
          }
        }
      }
    }, "0"]]
  }'
```

### Restore from Trash

```bash
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {
          "mailboxIds": {
            "'$INBOX_ID'": true,
            "'$TRASH_ID'": false
          }
        }
      }
    }, "0"]]
  }'
```

### Archive All Emails in Inbox

```bash
# First get all email IDs in inbox
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {"inMailbox": "'$INBOX_ID'"}
    }, "0"]]
  }'

# Then move all to archive
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "email1": {
          "mailboxIds": {
            "'$ARCHIVE_ID'": true,
            "'$INBOX_ID'": false
          }
        },
        "email2": {
          "mailboxIds": {
            "'$ARCHIVE_ID'": true,
            "'$INBOX_ID'": false
          }
        }
      }
    }, "0"]]
  }'
```

## Error Handling

**Mailbox not found:**
```json
{
  "methodResponses": [
    [
      "Email/set",
      {
        "notUpdated": {
          "{{email_id}}": {
            "type": "invalidProperties"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Handle by:**
- Verifying mailbox IDs are correct
- Checking mailbox exists via Mailbox/get
- Refreshing mailbox list

## Complete Example

```bash
#!/bin/bash

EMAIL="alice@example.com"
MAIL_TOKEN="app_token"
ACCOUNT_ID="account_id"
JMAP_BASE="http://localhost:8081/jmap"
EMAIL_ID="email123"
INBOX_ID="a"
TRASH_ID="d"

# Move to trash
echo "Moving email to trash..."
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "'$EMAIL_ID'": {
          "mailboxIds": {
            "'$TRASH_ID'": true,
            "'$INBOX_ID'": false
          }
        }
      }
    }, "0"]]
  }' | jq '.'
```

## Next Steps

- [HOW_TO_DELETE_MAIL.md](HOW_TO_DELETE_MAIL.md) - Permanent deletion
- [HOW_TO_MARK_READ.md](HOW_TO_MARK_READ.md) - Mark read/unread