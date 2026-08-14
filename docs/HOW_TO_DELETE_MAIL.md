# How to Delete Mail

Complete guide for deleting emails using JMAP.

## Overview

Deleting emails in JMAP involves moving emails to the Trash folder or permanently removing them.

## Move to Trash (Soft Delete)

Move email to Trash folder:

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

## Permanent Delete

Remove email from all mailboxes:

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

## Empty Trash

Get all emails in Trash and permanently delete them:

```bash
# Get trash emails
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/query", {
      "accountId": "'$ACCOUNT_ID'",
      "filter": {"inMailbox": "'$TRASH_ID'"}
    }, "0"]]
  }'

# Permanently delete all
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Email/set", {
      "accountId": "'$ACCOUNT_ID'",
      "update": {
        "email1": {"mailboxIds": {}},
        "email2": {"mailboxIds": {}}
      }
    }, "0"]]
  }'
```

## Next Steps

- [HOW_TO_MOVE_MAIL.md](HOW_TO_MOVE_MAIL.md) - Move between folders