# Stalwart Management API Reference

**CRITICAL SECURITY WARNING**: These APIs are for **SERVER-SIDE USE ONLY**. They must never be called from the frontend or exposed to normal users. They use administrative credentials that have full control over the mail server.

**DEVELOPMENT ONLY**: The examples in this document use development credentials (`change-me-development-only`). In production, use strong, unique credentials managed via secrets management.

## Overview

The Norest backend uses Stalwart's JMAP-based management API to perform administrative operations. These operations are called by the provisioning worker using admin credentials.

## Authentication

**Method**: HTTP Basic Authentication

**Credentials**:
- Username: `STALWART_ADMIN_USER` environment variable
- Password: `STALWART_ADMIN_PASSWORD` environment variable

**Example**:
```bash
curl -X POST http://localhost:8081/jmap \
  -u "admin:change-me-development-only" \
  -H "Content-Type: application/json" \
  -d '{...}'
```

**Security Notes**:
- Credentials are stored in environment variables
- Never expose these credentials to frontend or users
- Use secrets management in production
- Rotate credentials regularly

## Capabilities

All management operations use these JMAP capabilities:

```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ]
}
```

## Management Operations

### 1. Create Domain

**Purpose**: Create a new mail domain in Stalwart

**JMAP Method**: `x:Domain/set`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (domain creation)

**Idempotent**: No (uses unique create key to avoid conflicts)

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Domain/set",
      {
        "create": {
          "domain_example.com_1234567890": {
            "name": "example.com"
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
      "x:Domain/set",
      {
        "created": {
          "domain_example.com_1234567890": {
            "id": "abc123",
            "name": "example.com"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Error Response**:
```json
{
  "methodResponses": [
    [
      "x:Domain/set",
      {
        "notCreated": {
          "domain_example.com_1234567890": {
            "type": "alreadyExists"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Uses timestamp-based unique create key
- Returns Stalwart domain ID in response
- ID is stored in Norest `domains.stalwart_domain_id`

### 2. Create Account

**Purpose**: Create a new mail account in Stalwart

**JMAP Method**: `x:Account/set`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (account creation)

**Idempotent**: No (uses unique create key to avoid conflicts)

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/set",
      {
        "create": {
          "account_alice_1234567890": {
            "@type": "User",
            "name": "alice",
            "domainId": "abc123",
            "description": "alice@example.com",
            "credentials": {
              "0": {
                "@type": "Password",
                "secret": "random_password_32_chars"
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

**Response**:
```json
{
  "methodResponses": [
    [
      "x:Account/set",
      {
        "created": {
          "account_alice_1234567890": {
            "id": "def456",
            "name": "alice",
            "domainId": "abc123"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Notes**:
- `@type` is always "User"
- `name` is the local part (before @)
- `domainId` is the Stalwart domain ID
- Generates random 32-character password
- ID is stored in Norest `mailboxes.stalwart_account_id`

### 3. Create App Password

**Purpose**: Create an application password for JMAP authentication

**JMAP Method**: `x:AppPassword/set`

**Capability**: `urn:stalwart:jmap`

**Caller**: Mail service (session creation)

**Idempotent**: No

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:AppPassword/set",
      {
        "accountId": "def456",
        "create": {
          "app_password_1234567890": {
            "description": "Norest Session Token - user-uuid"
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
      "x:AppPassword/set",
      {
        "created": {
          "app_password_1234567890": {
            "secret": "app_aaaaaaiafjblhhlm0ftsgjzamqbjcp0zdzta"
          }
        }
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Used for mail session tokens
- Secret is returned in response
- Secret is used as JMAP access token
- Description includes user ID for tracking

### 4. Update Account Quota

**Purpose**: Update storage quota for an account

**JMAP Method**: `x:Account/set`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (quota sync)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/set",
      {
        "update": {
          "def456": {
            "maxDiskQuota": 10737418240
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
      "x:Account/set",
      {
        "updated": {
          "def456": {}
        }
      },
      "0"
    ]
  ]
}
```

**Notes**:
- `maxDiskQuota` is in bytes
- Called when plan changes
- 10737418240 bytes = 10 GB

### 5. Domain Exists

**Purpose**: Check if a domain exists by ID

**JMAP Method**: `x:Domain/get`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (idempotency check)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Domain/get",
      {
        "ids": ["abc123"]
      },
      "0"
    ]
  ]
}
```

**Response** (exists):
```json
{
  "methodResponses": [
    [
      "x:Domain/get",
      {
        "list": [
          {
            "id": "abc123",
            "name": "example.com"
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Response** (not found):
```json
{
  "methodResponses": [
    [
      "x:Domain/get",
      {
        "list": []
      },
      "0"
    ]
  ]
}
```

### 6. Domain Exists And Matches

**Purpose**: Check if a domain exists by ID AND matches expected name

**JMAP Method**: `x:Domain/get`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (idempotency check with name verification)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Domain/get",
      {
        "ids": ["abc123"]
      },
      "0"
    ]
  ]
}
```

**Response** (matches):
```json
{
  "methodResponses": [
    [
      "x:Domain/get",
      {
        "list": [
          {
            "id": "abc123",
            "name": "example.com"
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Norest compares returned name with expected name
- Used to prevent ID reuse across different domains

### 7. Find Domain By Name

**Purpose**: Find a domain ID by name (gets all domains and filters)

**JMAP Method**: `x:Domain/get`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (duplicate detection)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Domain/get",
      {},
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
      "x:Domain/get",
      {
        "list": [
          {
            "id": "abc123",
            "name": "example.com"
          },
          {
            "id": "def456",
            "name": "test.com"
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Gets all domains without ID filter
- Norest filters by name in application code
- Used to prevent duplicate domain creation

### 8. Account Exists

**Purpose**: Check if an account exists by ID

**JMAP Method**: `x:Account/get`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (idempotency check)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/get",
      {
        "ids": ["def456"]
      },
      "0"
    ]
  ]
}
```

**Response** (exists):
```json
{
  "methodResponses": [
    [
      "x:Account/get",
      {
        "list": [
          {
            "id": "def456",
            "name": "alice",
            "domainId": "abc123"
          }
        ]
      },
      "0"
    ]
  ]
}
```

### 9. Account Exists And Matches

**Purpose**: Check if an account exists by ID AND matches expected localPart and domainId

**JMAP Method**: `x:Account/get`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (idempotency check with verification)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/get",
      {
        "ids": ["def456"]
      },
      "0"
    ]
  ]
}
```

**Response** (matches):
```json
{
  "methodResponses": [
    [
      "x:Account/get",
      {
        "list": [
          {
            "id": "def456",
            "name": "alice",
            "domainId": "abc123"
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Norest compares name and domainId with expected values
- Used to prevent ID reuse across different accounts

### 10. Find Account By Name

**Purpose**: Find an account ID by name (gets all accounts and filters)

**JMAP Method**: `x:Account/get`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (duplicate detection)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/get",
      {},
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
      "x:Account/get",
      {
        "list": [
          {
            "id": "def456",
            "name": "alice",
            "domainId": "abc123"
          },
          {
            "id": "ghi789",
            "name": "bob",
            "domainId": "abc123"
          }
        ]
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Gets all accounts without ID filter
- Norest filters by name in application code
- Used to prevent duplicate account creation

### 11. Disable Account

**Purpose**: Disable a Stalwart account (suspension)

**JMAP Method**: `x:Account/set`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (account suspension)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/set",
      {
        "update": {
          "def456": {
            "active": false
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
      "x:Account/set",
      {
        "updated": {
          "def456": {}
        }
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Sets `active` field to false
- Account cannot authenticate when disabled
- Used for admin suspension

### 12. Enable Account

**Purpose**: Enable a Stalwart account (reactivation)

**JMAP Method**: `x:Account/set`

**Capability**: `urn:stalwart:jmap`

**Caller**: Provisioning worker (account reactivation)

**Idempotent**: Yes

**Development Only**: No

**Request**:
```json
{
  "using": [
    "urn:ietf:params:jmap:core",
    "urn:stalwart:jmap"
  ],
  "methodCalls": [
    [
      "x:Account/set",
      {
        "update": {
          "def456": {
            "active": true
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
      "x:Account/set",
      {
        "updated": {
          "def456": {}
        }
      },
      "0"
    ]
  ]
}
```

**Notes**:
- Sets `active` field to true
- Account can authenticate when enabled
- Used for admin reactivation

## Error Handling

All management operations can return JMAP-level errors:

**Error Response Format**:
```json
{
  "methodResponses": [
    [
      "error",
      {
        "type": "invalidArguments",
        "description": "Invalid domain name"
      },
      "0"
    ]
  ]
}
```

**Common Error Types**:
- `invalidArguments`: Invalid request parameters
- `alreadyExists`: Resource already exists
- `notFound`: Resource not found
- `serverFail`: Internal server error

## Security Guidelines

### Server-Side Only

These APIs must never be called from:
- Frontend JavaScript
- Mobile applications
- Public API endpoints
- User-facing code

### Credential Protection

- Store admin credentials in environment variables
- Use secrets management in production
- Never log credentials
- Rotate credentials regularly
- Audit access to credentials

### Rate Limiting

Consider rate limiting management operations to prevent abuse, even though they're server-side.

### Audit Logging

Log all management operations for security auditing:
- Operation type
- Target resource
- Timestamp
- Result (success/failure)

## Development vs Production

### Development

- Use default admin credentials from `.env.example`
- HTTP allowed
- Detailed error messages
- Test domains/accounts allowed

### Production

- Use strong, unique admin credentials
- HTTPS required
- Generic error messages
- Strict validation
- Audit logging enabled

## Summary

**Total Management Operations**: 12

**JMAP Methods Used**:
- `x:Domain/set` (create, update)
- `x:Domain/get` (query, existence check)
- `x:Account/set` (create, update)
- `x:Account/get` (query, existence check)
- `x:AppPassword/set` (create)

**Capabilities Required**:
- `urn:ietf:params:jmap:core`
- `urn:stalwart:jmap`

**Caller Components**:
- Provisioning worker (domain/account operations)
- Mail service (app password creation)

**Idempotent Operations**: 8
**Non-Idempotent Operations**: 4

**Development Only**: 0 (all used in production)