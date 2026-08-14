# How to Send Mail

Complete guide for sending email messages using Norest Mail and JMAP.

## Overview

To send email messages, you need to:
1. Authenticate with Norest
2. Create a mail session
3. Get sender identity
4. Create email draft
5. Submit email for delivery

## Step-by-Step Guide

### Prerequisites

You should already have:
- Norest access token (from login/register)
- Mail session with JMAP access token
- Valid sender identity

### Step 1: Get Mail Session

```bash
curl -X POST http://localhost:8080/v1/mail/session \
  -H "Authorization: Bearer {{norest_access_token}}"
```

**Save session data:**
```json
{
  "access_token": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta",
  "account_id": "stalwart_account_id"
}
```

Variables:
- `{{mail_access_token}}`: `app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta`
- `{{account_id}}`: `stalwart_account_id`
- `{{email_address}}`: `alice@example.com`

### Step 2: Get Mailboxes

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

**Save the Drafts ID:**
- `{{drafts_id}}`: ID where `role` is `drafts`

### Step 3: Get Sender Identity

```bash
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:submission"
    ],
    "methodCalls": [
      [
        "Identity/get",
        {
          "accountId": "{{account_id}}"
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
      "Identity/get",
      {
        "list": [
          {
            "id": "id1",
            "name": "Alice Smith",
            "email": "alice@example.com"
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Save the identity ID:**
- `{{identity_id}}`: `id1`

### Step 4: Create Email Draft

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
              "from": [
                {
                  "name": "Alice Smith",
                  "email": "alice@example.com"
                }
              ],
              "to": [
                {
                  "name": "Bob Johnson",
                  "email": "bob@example.com"
                }
              ],
              "subject": "Hello from Norest Mail",
              "keywords": {
                "$seen": true
              },
              "bodyValues": {
                "b1": {
                  "value": "Hi Bob,\n\nThis is a test email sent from Norest Mail.\n\nBest regards,\nAlice",
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
          }
        }
      },
      "0"
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
        "created": {
          "msg1": {
            "id": "email123",
            "blobId": "blob456",
            "threadId": "thread789"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Note the email ID:**
- `{{email_id}}`: `email123`
- The create key `msg1` is used to reference this email

### Step 5: Submit Email for Delivery

```bash
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:submission"
    ],
    "methodCalls": [
      [
        "EmailSubmission/set",
        {
          "accountId": "{{account_id}}",
          "create": {
            "sub1": {
              "emailId": "#msg1",
              "identityId": "{{identity_id}}"
            }
          }
        },
        "1"
      ]
    ]
  }'
```

**Important:** Use `#msg1` (with `#` prefix) to reference the email created in the same request.

**Response:**
```json
{
  "methodResponses": [
    [
      "EmailSubmission/set",
      {
        "created": {
          "sub1": {
            "id": "sub123",
            "emailId": "email123",
            "threadId": "thread789"
          }
        }
      },
      "1"
    ]
  ]
}
```

### Step 6: Combined Create and Submit

You can combine draft creation and submission in a single request:

```bash
curl -X POST http://localhost:8081/jmap \
  -u "{{email_address}}:{{mail_access_token}}" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:mail",
      "urn:ietf:params:jmap:submission"
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
              "from": [
                {
                  "name": "Alice Smith",
                  "email": "alice@example.com"
                }
              ],
              "to": [
                {
                  "name": "Bob Johnson",
                  "email": "bob@example.com"
                }
              ],
              "subject": "Hello from Norest Mail",
              "keywords": {
                "$seen": true
              },
              "bodyValues": {
                "b1": {
                  "value": "Hi Bob,\n\nThis is a test email sent from Norest Mail.\n\nBest regards,\nAlice",
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
          }
        }
      },
      "0"
    ],
    [
      "EmailSubmission/set",
      {
        "accountId": "{{account_id}}",
        "create": {
          "sub1": {
            "emailId": "#msg1",
            "identityId": "{{identity_id}}"
          }
        }
      },
      "1"
    ]
  }'
```

## Email Structure

### Basic Email Fields

- `mailboxIds`: Which folders contain the email
- `from`: Sender information
- `to`: Primary recipients
- `cc`: Carbon copy recipients
- `bcc`: Blind carbon copy recipients
- `subject`: Email subject line
- `keywords`: Email flags ($seen, $flagged, etc.)

### Body Structure

- `bodyValues`: Map of body part IDs to content
- `textBody`: Array of text body parts
- `htmlBody`: Array of HTML body parts (optional)

### Recipients

**Single recipient:**
```json
"to": [
  {
    "name": "Bob Johnson",
    "email": "bob@example.com"
  }
]
```

**Multiple recipients:**
```json
"to": [
  {
    "name": "Bob Johnson",
    "email": "bob@example.com"
  },
  {
    "name": "Carol White",
    "email": "carol@example.com"
  }
]
```

**CC recipients:**
```json
"cc": [
  {
    "name": "Dave Brown",
    "email": "dave@example.com"
  }
]
```

**BCC recipients:**
```json
"bcc": [
  {
    "name": "Eve Davis",
    "email": "eve@example.com"
  }
]
```

## HTML Email

To send HTML email:

```json
{
  "create": {
    "msg1": {
      "mailboxIds": {
        "{{drafts_id}}": true
      },
      "from": [
        {
          "name": "Alice Smith",
          "email": "alice@example.com"
        }
      ],
      "to": [
        {
          "name": "Bob Johnson",
          "email": "bob@example.com"
        }
      ],
      "subject": "HTML Email Test",
      "bodyValues": {
        "b1": {
          "value": "Plain text version for email clients that don't support HTML"
        },
        "b2": {
          "value": "<h1>HTML Email</h1><p>This is <strong>HTML</strong> content.</p>"
        }
      },
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
      ]
    }
  }
}
```

## Reply to Email

To reply to an existing email:

1. Get the original email to extract details
2. Create reply with appropriate subject and body

```json
{
  "create": {
    "msg1": {
      "mailboxIds": {
        "{{drafts_id}}": true
      },
      "from": [
        {
          "name": "Alice Smith",
          "email": "alice@example.com"
        }
      ],
      "to": [
        {
          "name": "Bob Johnson",
          "email": "bob@example.com"
        }
      ],
      "subject": "Re: Original Subject",
      "bodyValues": {
        "b1": {
          "value": "On 2026-08-14, Bob Johnson wrote:\n\n> Original message\n\nMy reply here."
        }
      },
      "textBody": [
        {
          "partId": "b1",
          "type": "text/plain"
        }
      ],
      "keywords": {
        "$answered": true
      }
    }
  }
}
```

## Draft Management

### Save Draft (Don't Send)

Create the email in Drafts folder but don't submit:

```json
{
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
      },
      "0"
    ]
  ]
}
```

### Update Draft

To update an existing draft:

```json
{
  "methodCalls": [
    [
      "Email/set",
      {
        "accountId": "{{account_id}}",
        "update": {
          "email123": {
            "subject": "Updated Subject",
            "bodyValues": {
              "b1": {
                "value": "Updated content..."
              }
            }
          }
        }
      },
      "0"
    ]
  ]
}
```

### Send Saved Draft

To send a previously saved draft:

```json
{
  "methodCalls": [
    [
      "EmailSubmission/set",
      {
        "accountId": "{{account_id}}",
        "create": {
          "sub1": {
            "emailId": "email123",
            "identityId": "{{identity_id}}"
          }
        }
      },
      "0"
    ]
  ]
}
```

## Error Handling

**Common errors:**

1. **Invalid recipient:**
   - Verify recipient email format
   - Check recipient domain exists

2. **Identity not found:**
   - Verify identity_id from Identity/get
   - Check identity belongs to your account

3. **Email creation failed:**
   - Check mailboxIds is valid
   - Verify bodyValues are not empty
   - Ensure content is properly formatted

4. **Submission failed:**
   - Verify emailId reference is correct
   - Check identityId is valid
   - Ensure Stalwart SMTP is configured

## Complete Example

```bash
#!/bin/bash

# Variables
EMAIL="alice@example.com"
MAIL_TOKEN="app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta"
ACCOUNT_ID="stalwart_account_id"
JMAP_BASE="http://localhost:8081/jmap"

# 1. Get mailboxes
MAILBOXES=$(curl -s -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
    "methodCalls": [["Mailbox/get", {"accountId": "'$ACCOUNT_ID'"}, "0"]]
  }')

DRAFTS_ID=$(echo $MAILBOXES | jq -r '.methodResponses[0][1].list[] | select(.role == "drafts") | .id')

# 2. Get identity
IDENTITY=$(curl -s -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:submission"],
    "methodCalls": [["Identity/get", {"accountId": "'$ACCOUNT_ID'"}, "0"]]
  }')

IDENTITY_ID=$(echo $IDENTITY | jq -r '.methodResponses[0][1].list[0].id')

# 3. Create and send email
curl -X POST $JMAP_BASE \
  -u "$EMAIL:$MAIL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:ietf:params:jmap:mail",
      "urn:ietf:params:jmap:submission"
    ],
    "methodCalls": [
      [
        "Email/set",
        {
          "accountId": "'$ACCOUNT_ID'",
          "create": {
            "msg1": {
              "mailboxIds": {
                "'$DRAFTS_ID'": true
              },
              "from": [
                {
                  "name": "Alice Smith",
                  "email": "'$EMAIL'"
                }
              ],
              "to": [
                {
                  "name": "Bob Johnson",
                  "email": "bob@example.com"
                }
              ],
              "subject": "Test Email",
              "bodyValues": {
                "b1": {
                  "value": "This is a test email."
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
      ],
      [
        "EmailSubmission/set",
        {
          "accountId": "'$ACCOUNT_ID'",
          "create": {
            "sub1": {
              "emailId": "#msg1",
              "identityId": "'$IDENTITY_ID'"
            }
          }
        }
      ]
    ]
  }'
```

## Next Steps

- [HOW_TO_GET_MAIL.md](HOW_TO_GET_MAIL.md) - Retrieve messages
- [HOW_TO_READ_MAIL.md](HOW_TO_READ_MAIL.md) - Read message details
- [SEARCH.md](SEARCH.md) - Search emails
- [DRAFTS.md](DRAFTS.md) - Draft management