# JMAP Mail API Reference

Complete reference of JMAP mail methods used by Norest Mail for user mail operations.

## Overview

Norest Mail uses JMAP (JSON Meta Application Protocol) for mail operations. The frontend communicates directly with Stalwart Mail Server using JMAP after obtaining a mail session from Norest.

## Authentication

**Method**: HTTP Basic Authentication

**Credentials**:
- Username: Full email address (e.g., `alice@example.com`)
- Password: App password token from Norest mail session

**Example**:
```bash
curl -X POST http://localhost:8081/jmap \
  -u "alice@example.com:app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta" \
  -H "Content-Type: application/json" \
  -d '{...}'
```

**Session Flow**:
1. User authenticates with Norest (JWT)
2. User requests mail session from Norest
3. Norest creates AppPassword in Stalwart
4. User uses AppPassword for JMAP authentication

## Capabilities

The current implementation uses these JMAP capabilities:

```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:mail",
    "urn:ietf:params:jmap:submission"
  ]
}
```

### Capability Breakdown

- **urn:ietf:params:jmap:core**: Core JMAP functionality (session, errors, etc.)
- **urn:ietf:params:jmap:mail**: Mail data access (mailboxes, emails, threads)
- **urn:ietf:params:jmap:submission**: Email submission (sending messages)

## JMAP Methods Used

### 1. Mailbox/get

**Purpose**: Get mailbox folders (Inbox, Sent, Drafts, Trash, etc.)

**Capability**: `urn:ietf:params:jmap:mail`

**Request**:
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
        "accountId": "stalwart_account_id"
      },
      "0"
    ]
  ]
}
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
            "sortOrder": 0,
            "totalEmails": 10,
            "unreadEmails": 3,
            "totalThreads": 8,
            "unreadThreads": 2,
            "myRights": {
              "mayReadItems": true,
              "mayAddItems": true,
              "mayRemoveItems": true,
              "maySetSeen": true,
              "maySetKeywords": true,
              "mayCreateChild": true,
              "mayDelete": true,
              "mayRename": true,
              "maySubmit": true
            },
            "isSubscribed": true
          },
          {
            "id": "b",
            "name": "Drafts",
            "role": "drafts",
            "sortOrder": 0,
            "totalEmails": 2,
            "unreadEmails": 0,
            "totalThreads": 2,
            "unreadThreads": 0,
            "myRights": {...},
            "isSubscribed": true
          },
          {
            "id": "c",
            "name": "Sent",
            "role": "sent",
            "sortOrder": 0,
            "totalEmails": 15,
            "unreadEmails": 0,
            "totalThreads": 15,
            "unreadThreads": 0,
            "myRights": {...},
            "isSubscribed": true
          },
          {
            "id": "d",
            "name": "Trash",
            "role": "trash",
            "sortOrder": 0,
            "totalEmails": 5,
            "unreadEmails": 0,
            "totalThreads": 5,
            "unreadThreads": 0,
            "myRights": {...},
            "isSubscribed": true
          },
          {
            "id": "e",
            "name": "Junk Mail",
            "role": "junk",
            "sortOrder": 0,
            "totalEmails": 1,
            "unreadEmails": 1,
            "totalThreads": 1,
            "unreadThreads": 1,
            "myRights": {...},
            "isSubscribed": true
          }
        ],
        "notFound": [],
        "state": "a1b2c3d4"
      },
      "0"
    ]
  ]
}
```

**Common Mailbox Roles**:
- `inbox`: Incoming messages
- `drafts`: Draft messages
- `sent`: Sent messages
- `trash`: Deleted messages
- `junk`: Spam messages
- `archive`: Archived messages (if present)

**Usage Notes**:
- Always called with accountId from mail session
- Returns all user mailboxes
- Use `role` to identify special folders
- Use `id` for email operations

### 2. Identity/get

**Purpose**: Get email identities (from addresses for sending)

**Capability**: `urn:ietf:params:jmap:submission`

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:submission"
  ],
  "methodCalls": [
    [
      "Identity/get",
      {
        "accountId": "stalwart_account_id"
      },
      "0"
    ]
  ]
}
```

**Response**:
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
            "email": "alice@example.com",
            "replyTo": null,
            "bcc": null,
            "textSignature": "Sent from Norest Mail",
            "htmlSignature": null,
            "mayDelete": true
          }
        ],
        "notFound": [],
        "state": "e5f6g7h8"
      },
      "0"
    ]
  ]
}
```

**Usage Notes**:
- Returns identities for sending messages
- Use `id` for EmailSubmission operations
- Typically one identity per email address

### 3. Email/set (Create)

**Purpose**: Create email drafts

**Capability**: `urn:ietf:params:jmap:mail`

**Request**:
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
        "accountId": "stalwart_account_id",
        "create": {
          "msg1": {
            "mailboxIds": {
              "b": true
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
                "value": "This is the email body content.",
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
      },
      "0"
    ]
  ]
}
```

**Response**:
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
            "threadId": "thread789",
            "size": 1234
          }
        },
        "notCreated": {},
        "updated": {},
        "notUpdated": {},
        "destroyed": {},
        "notDestroyed": {},
        "state": "i9j0k1l2"
      },
      "0"
    ]
  ]
}
```

**Usage Notes**:
- `mailboxIds` specifies which folders contain the email
- `from` is the sender identity
- `to`, `cc`, `bcc` are recipients
- `bodyValues` contains email body content
- `textBody` references body value parts
- Returns email ID for submission
- `#msg1` reference used in EmailSubmission

### 4. EmailSubmission/set (Create)

**Purpose**: Submit email for delivery (send message)

**Capability**: `urn:ietf:params:jmap:submission`

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:ietf:params:jmap:submission"
  ],
  "methodCalls": [
    [
      "EmailSubmission/set",
      {
        "accountId": "stalwart_account_id",
        "create": {
          "sub1": {
            "emailId": "#msg1",
            "identityId": "id1"
          }
        }
      },
      "1"
    ]
  ]
}
```

**Response**:
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
        },
        "notCreated": {},
        "updated": {},
        "notUpdated": {},
        "destroyed": {},
        "notDestroyed": {},
        "state": "m3n4o5p6"
      },
      "1"
    ]
  ]
}
```

**Usage Notes**:
- `emailId` references draft email (using `#` prefix)
- `identityId` references sender identity
- Stalwart handles SMTP delivery
- Email is moved from Drafts to Sent after delivery

### 5. Email/query

**Purpose**: Query/search for emails

**Capability**: `urn:ietf:params:jmap:mail`

**Request** (basic):
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
        "accountId": "stalwart_account_id",
        "limit": 10
      },
      "0"
    ]
  ]
}
```

**Request** (with filter):
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
        "accountId": "stalwart_account_id",
        "filter": {
          "inMailbox": "a",
          "text": "search term"
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
}
```

**Response**:
```json
{
  "methodResponses": [
    [
      "Email/query",
      {
        "accountId": "stalwart_account_id",
        "queryState": "q7r8s9t0",
        "canCalculateChanges": true,
        "position": 0,
        "ids": [
          "email123",
          "email456",
          "email789"
        ],
        "total": 150,
        "limit": 20
      },
      "0"
    ]
  ]
}
```

**Filter Options**:
- `inMailbox`: Filter by mailbox ID
- `text`: Full-text search
- `from`: Filter by sender
- `to`: Filter by recipient
- `subject`: Filter by subject
- `hasKeyword`: Filter by keyword (e.g., `$seen`, `$flagged`)

**Sort Options**:
- `receivedAt`: Date received
- `sentAt`: Date sent
- `size`: Email size
- `subject`: Subject line

**Usage Notes**:
- Returns list of email IDs (not full content)
- Use Email/get to retrieve full email data
- Supports pagination with `limit` and `position`
- Can calculate changes for synchronization

### 6. Email/get

**Purpose**: Get full email content

**Capability**: `urn:ietf:params:jmap:mail`

**Request**:
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
        "accountId": "stalwart_account_id",
        "ids": ["email123", "email456"],
        "properties": [
          "id",
          "blobId",
          "threadId",
          "mailboxIds",
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
            "subject": "Re: Hello from Norest Mail",
            "receivedAt": "2026-08-14T10:30:00Z",
            "preview": "This is a preview of the email content...",
            "keywords": {
              "$seen": true,
              "$flagged": false
            },
            "bodyValues": {
              "b1": {
                "value": "This is the full email body content.",
                "isEncodingProblem": false,
                "isTruncated": false
              }
            },
            "textBody": [
              {
                "partId": "b1",
                "type": "text/plain"
              }
            ],
            "hasAttachments": false,
            "size": 1234
          }
        ],
        "notFound": [],
        "state": "u1v2w3x4"
      },
      "0"
    ]
  ]
}
```

**Properties**:
- `id`: Email ID
- `blobId`: Binary blob ID for attachments
- `threadId`: Thread ID
- `mailboxIds`: Mailboxes containing this email
- `from`: Sender
- `to`, `cc`, `bcc`: Recipients
- `subject`: Subject line
- `receivedAt`: Timestamp received
- `sentAt`: Timestamp sent
- `preview`: Short preview text
- `keywords`: Flags (`$seen`, `$flagged`, `$answered`, etc.)
- `bodyValues`: Email body content
- `textBody`, `htmlBody`: Body structure
- `attachments`: Attachment list
- `hasAttachments`: Boolean flag
- `size`: Email size in bytes

**Usage Notes**:
- Can request specific properties only
- Use for displaying email content
- Preview property for list views
- Body values for full content

### 7. Email/set (Update)

**Purpose**: Update email properties (read/unread, flag, move)

**Capability**: `urn:ietf:params:jmap:mail`

**Request** (mark as read):
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
        "accountId": "stalwart_account_id",
        "update": {
          "email123": {
            "keywords/$seen": true
          }
        }
      },
      "0"
    ]
  ]
}
```

**Request** (mark as unread):
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
        "accountId": "stalwart_account_id",
        "update": {
          "email123": {
            "keywords/$seen": false
          }
        }
      },
      "0"
    ]
  ]
}
```

**Request** (flag/star):
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
        "accountId": "stalwart_account_id",
        "update": {
          "email123": {
            "keywords/$flagged": true
          }
        }
      },
      "0"
    ]
  ]
}
```

**Request** (move to different mailbox):
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
        "accountId": "stalwart_account_id",
        "update": {
          "email123": {
            "mailboxIds": {
              "d": true,
              "a": false
            }
          }
        }
      },
      "0"
    ]
  ]
}
```

**Response**:
```json
{
  "methodResponses": [
    [
      "Email/set",
      {
        "updated": {
          "email123": {}
        },
        "notUpdated": {},
        "state": "y5z6a7b8"
      },
      "0"
    ]
  ]
}
```

**Usage Notes**:
- `keywords/$seen`: Read/unread status
- `keywords/$flagged`: Starred/flagged status
- `keywords/$answered`: Replied status
- `mailboxIds`: Move between folders
- `keywords/$draft`: Draft status

## Email Keywords

Standard JMAP keywords used:

- `$seen`: Message has been read
- `$flagged`: Message is flagged/starred
- `$answered`: Message has been answered
- `$draft`: Message is a draft
- `$forwarded`: Message has been forwarded
- `$junk`: Message is junk/spam
- `$phishing`: Message is phishing

## Error Handling

**JMAP Error Response**:
```json
{
  "methodResponses": [
    [
      "error",
      {
        "type": "invalidArguments",
        "description": "Invalid email ID"
      },
      "0"
    ]
  ]
}
```

**Common Error Types**:
- `invalidArguments`: Invalid request parameters
- `notFound`: Resource not found
- `permissionDenied`: Insufficient permissions
- `serverFail`: Internal server error
- `rateLimit`: Rate limit exceeded

## Session Discovery

Before making JMAP calls, discover the session endpoint:

**Request**:
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
      "isReadOnly": false,
      "accountCapabilities": {
        "urn:ietf:params:jmap:mail": {
          "maxMailboxesPerEmail": null,
          "maxEmailsPerMailbox": null,
          "maxAttachmentsPerEmail": null
        },
        "urn:ietf:params:jmap:submission": {
          "maxDelayedSend": 0,
          "maxMaySend": null
        }
      }
    }
  },
  "primaryAccounts": {
    "urn:ietf:params:jmap:mail": "stalwart_account_id",
    "urn:ietf:params:jmap:submission": "stalwart_account_id"
  },
  "apiUrl": "http://localhost:8081/jmap",
  "downloadUrl": "http://localhost:8081/download/{blobId}/{accountId}",
  "uploadUrl": "http://localhost:8081/upload/{accountId}",
  "eventSourceUrl": "http://localhost:8081/events/{accountId}",
  "state": "c1d2e3f4"
}
```

**Norest Integration**:
Norest provides the JMAP session URL in the mail session response:
```json
{
  "jmap_session_url": "http://localhost:8081/.well-known/jmap",
  "account_id": "stalwart_account_id"
}
```

## Summary

**Total JMAP Methods Used**: 7

**Methods**:
1. Mailbox/get - Get mailbox folders
2. Identity/get - Get sender identities
3. Email/set (create) - Create email drafts
4. EmailSubmission/set (create) - Send messages
5. Email/query - Search/query emails
6. Email/get - Get email content
7. Email/set (update) - Update email properties

**Capabilities Required**:
- urn:ietf:params:jmap:core
- urn:ietf:params:jmap:mail
- urn:ietf:params:jmap:submission

**Authentication**: HTTP Basic with app password from Norest session

**Current Limitations**:
- No synchronization methods (changes, queryChanges) currently used
- No thread operations (Thread/query, Thread/get) currently used
- No push/event source currently used
- No attachment upload/download currently documented

## Additional Resources

- [JMAP RFC 8621](https://datatracker.ietf.org/doc/html/rfc8621) - Core JMAP specification
- [JMAP Mail RFC 8621](https://datatracker.ietf.org/doc/html/rfc8621#section-5) - Mail data model
- [JMAP Submission RFC 8621](https://datatracker.ietf.org/doc/html/rfc8621#section-6) - Email submission
- [Stalwart JMAP Documentation](https://stalwartlabs.com/docs/jmap/)