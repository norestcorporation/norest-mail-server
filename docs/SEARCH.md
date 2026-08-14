# Search

Complete guide for searching emails using JMAP.

## Basic Text Search

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
            "text": "search term"
          }
        },
        "0"
      ]
    ]
  }'
```

## Search by Sender

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
            "from": "bob@example.com"
          }
        },
        "0"
      ]
    ]
  }'
```

## Search by Subject

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
            "subject": "meeting"
          }
        },
        "0"
      ]
    ]
  }'
```

## Search in Specific Mailbox

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
            "inMailbox": "{{inbox_id}}",
            "text": "important"
          }
        },
        "0"
      ]
    ]
  }'
```

## Combined Filters

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
            "inMailbox": "{{inbox_id}}",
            "from": "bob@example.com",
            "text": "urgent"
          }
        },
        "0"
      ]
    ]
  }'
```

## Search Unread Emails

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
            "inMailbox": "{{inbox_id}}",
            "hasKeyword": "$seen"
          }
        },
        "0"
      ]
    ]
  }'
```

## Search Flagged Emails

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

## Search by Date Range

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
            "after": "2026-08-01T00:00:00Z",
            "before": "2026-08-31T23:59:59Z"
          }
        },
        "0"
      ]
    ]
  }'
```