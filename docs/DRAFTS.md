# Drafts

Complete guide for managing email drafts using JMAP.

## Create Draft

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
          "create": {
            "msg1": {
              "mailboxIds": {
                "{{drafts_id}}": true
              },
              "subject": "Draft Email",
              "bodyValues": {
                "b1": {
                  "value": "Draft content..."
                }
              },
              "textBody": [
                {
                  "partId": "b1",
                  "type": "text/plain"
                }
              ]
            }
          }
        }
      },
      "0"
    ]
  }'
```

## Update Draft

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
              "subject": "Updated Subject",
              "bodyValues": {
                "b1": {
                  "value": "Updated content..."
                }
              }
            }
          }
        }
      },
      "0"
    ]
  }'
```

## Send Draft

See [HOW_TO_SEND_MAIL.md](HOW_TO_SEND_MAIL.md) for details.