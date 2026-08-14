# Norest Mail Postman Collection

Complete Postman collection for testing the Norest Mail backend API.

## Installation

### 1. Import Collection

1. Open Postman
2. Click "Import" in the top left
3. Select "Norest-Mail.postman_collection.json"
4. Click "Import"

### 2. Import Environment

1. Click "Manage Environments" (gear icon in top right)
2. Click "Import"
3. Select "Norest-Mail.postman_environment.json"
4. Click "Import"

### 3. Select Environment

1. Click the environment dropdown in the top right
2. Select "Norest Mail Development"

## Environment Variables

The collection uses these environment variables:

### API Configuration
- `api_base`: Norest API base URL (default: `http://localhost:8080/v1`)
- `jmap_base`: Stalwart JMAP base URL (default: `http://localhost:8081/jmap`)

### Authentication
- `norest_access_token`: JWT access token (auto-filled from register/login)
- `mail_access_token`: Stalwart app password (auto-filled from mail session)
- `email_address`: User email address (auto-filled from register/login)

### JMAP Resources
- `account_id`: Stalwart account ID (auto-filled from mail session)
- `inbox_id`: Inbox mailbox ID (auto-filled from Mailbox/get)
- `drafts_id`: Drafts mailbox ID (auto-filled from Mailbox/get)
- `trash_id`: Trash mailbox ID (auto-filled from Mailbox/get)

### Norest Resources
- `domain_id`: Domain UUID (auto-filled from domain creation)
- `address_id`: Address UUID (auto-filled from address creation)
- `email_id`: Email ID (auto-filled from Email/query)
- `identity_id`: Sender identity ID (auto-filled from Identity/get)

### Admin Credentials (Development Only)
- `admin_user`: Stalwart admin username (default: `admin`)
- `admin_password`: Stalwart admin password (default: `change-me-development-only`)

### Test Data
- `test_email`: Test email address (default: `testuser@example.com`)
- `test_password`: Test password (default: `SecurePassword123!`)
- `recipient_email`: Recipient email for sending tests (default: `recipient@example.com`)

## Collection Structure

### 01 Health
Basic health check endpoints:
- Health Check
- Database Health
- Stalwart Health
- Readiness Check

### 02 Authentication
User authentication endpoints:
- Register User (saves `norest_access_token`, `email_address`)
- Login User (saves `norest_access_token`, `email_address`)
- Get Current User

### 03 Domains
Domain management endpoints:
- Create Domain (saves `domain_id`)
- List Domains
- Get Domain
- Start Domain Verification
- Get Domain Verification
- Delete Domain

### 04 Addresses
Address management endpoints:
- Create Address (saves `address_id`)
- List Addresses

### 05 Mail Session
Mail session endpoints:
- Create Mail Session (saves `mail_access_token`, `account_id`)
- Get Mail Account

### 06 Usage
Usage statistics:
- Get Usage

### 07 JMAP - Session
JMAP session discovery:
- JMAP Session Discovery

### 08 JMAP - Mailboxes
JMAP mailbox operations:
- Get Mailboxes (saves `inbox_id`, `drafts_id`, `trash_id`)

### 09 JMAP - Emails
JMAP email operations:
- Query Emails (saves `email_id`)
- Get Email
- Mark as Read
- Star Email
- Move to Trash

### 10 JMAP - Submission
JMAP email submission:
- Get Identity (saves `identity_id`)
- Create and Send Email

### 11 Stalwart Management - DEVELOPMENT ONLY
Stalwart management operations (requires admin credentials):
- List All Domains
- List All Accounts

**SECURITY WARNING**: These endpoints use admin credentials and should only be used in development.

### 12 Billing
Billing webhook endpoint:
- Billing Webhook

### 13 Admin
Admin endpoints:
- Suspend Account
- Reactivate Account

## Usage Guide

### 1. Health Check

Run the health endpoints first to verify services are running:

1. Click "01 Health"
2. Run "Health Check"
3. Run "Database Health"
4. Run "Stalwart Health"
5. Run "Readiness Check"

All should return 200 OK.

### 2. Register/Login

Create a new user or login to existing user:

1. Click "02 Authentication"
2. Run "Register User" or "Login User"
3. The `norest_access_token` and `email_address` variables will be auto-saved

### 3. Create Domain

Create a mail domain:

1. Click "03 Domains"
2. Run "Create Domain"
3. The `domain_id` variable will be auto-saved
4. Run "Start Domain Verification" (optional)
5. Run "Get Domain Verification" (optional)

### 4. Create Address

Create an email address:

1. Click "04 Addresses"
2. Run "Create Address"
3. The `address_id` variable will be auto-saved

### 5. Wait for Provisioning

The worker needs time to provision the mailbox. You can:

1. Wait 10-30 seconds manually
2. Poll "Get Mail Account" until status is "active"

### 6. Create Mail Session

Create a JMAP mail session:

1. Click "05 Mail Session"
2. Run "Create Mail Session"
3. The `mail_access_token` and `account_id` variables will be auto-saved

### 7. Get Mailboxes

Get JMAP mailboxes:

1. Click "08 JMAP - Mailboxes"
2. Run "Get Mailboxes"
3. The `inbox_id`, `drafts_id`, and `trash_id` variables will be auto-saved

### 8. Query Emails

Query for emails in Inbox:

1. Click "09 JMAP - Emails"
2. Run "Query Emails"
3. The `email_id` variable will be auto-saved (if emails exist)

### 9. Get Email Content

Get full email content:

1. Run "Get Email"
2. View the email details in the response

### 10. Mark as Read

Mark email as read:

1. Run "Mark as Read"
2. Email will be marked as read

### 11. Star Email

Star an email:

1. Run "Star Email"
2. Email will be flagged/starred

### 12. Move to Trash

Move email to trash:

1. Run "Move to Trash"
2. Email will be moved to Trash folder

### 13. Send Email

Send a test email:

1. Click "10 JMAP - Submission"
2. Run "Get Identity" (saves `identity_id`)
3. Run "Create and Send Email"
4. Email will be created and submitted for delivery

## Authentication Types

### Norest API Authentication

**Type**: Bearer Token

**Usage**: Norest protected endpoints

**Setup**: Auto-filled from Register/Login

**Example**:
```
Authorization: Bearer {{norest_access_token}}
```

### JMAP Authentication

**Type**: HTTP Basic Auth

**Usage**: JMAP mail operations

**Setup**: Auto-filled from mail session

**Example**:
```
Authorization: Basic {{email_address}}:{{mail_access_token}}
```

### Stalwart Admin Authentication

**Type**: HTTP Basic Auth

**Usage**: Stalwart management operations (development only)

**Setup**: Uses `admin_user` and `admin_password` from environment

**Example**:
```
Authorization: Basic {{admin_user}}:{{admin_password}}
```

**Security**: These credentials are for development only and should never be used in production.

## Test Scripts

The collection includes test scripts that automatically save IDs/tokens:

- Register/Login: Saves `norest_access_token`, `email_address`
- Create Domain: Saves `domain_id`
- Create Address: Saves `address_id`
- Create Mail Session: Saves `mail_access_token`, `account_id`
- Get Mailboxes: Saves `inbox_id`, `drafts_id`, `trash_id`
- Query Emails: Saves `email_id`
- Get Identity: Saves `identity_id`

## Running the End-to-End Flow

To run the complete end-to-end flow:

1. Run "Health Check" (01 Health)
2. Run "Register User" (02 Authentication)
3. Run "Create Domain" (03 Domains)
4. Run "Create Address" (04 Addresses)
5. Wait 30 seconds for provisioning
6. Run "Create Mail Session" (05 Mail Session)
7. Run "Get Mailboxes" (08 JMAP - Mailboxes)
8. Run "Query Emails" (09 JMAP - Emails)
9. Run "Get Email" (09 JMAP - Emails)
10. Run "Get Identity" (10 JMAP - Submission)
11. Run "Create and Send Email" (10 JMAP - Submission)

## Common Issues

### 401 Unauthorized

**Cause**: Invalid or missing access token

**Solution**:
1. Run "Register User" or "Login User" to get a fresh token
2. Check that `norest_access_token` variable is set
3. Verify token hasn't expired (15 minute lifetime)

### 403 Forbidden

**Cause**: Quota exceeded or account suspended

**Solution**:
1. Check plan limits via "Get Usage"
2. Verify account is not suspended
3. Contact admin if needed

### 404 Not Found

**Cause**: Resource ID doesn't exist

**Solution**:
1. Verify resource was created successfully
2. Check that the correct ID variable is set
3. Ensure you own the resource

### 500 Internal Server Error

**Cause**: Server error

**Solution**:
1. Check service logs: `docker-compose logs norest-api`
2. Check worker logs: `docker-compose logs norest-worker`
3. Check database health: "Database Health"
4. Check Stalwart health: "Stalwart Health"

### Mail Session Creation Fails

**Cause**: Mailbox not fully provisioned

**Solution**:
1. Wait longer for provisioning (30-60 seconds)
2. Check "Get Mail Account" status
3. Verify address creation succeeded
4. Check worker logs for provisioning errors

### JMAP Authentication Fails

**Cause**: Invalid app password or email

**Solution**:
1. Verify `mail_access_token` is set
2. Verify `email_address` is set
3. Recreate mail session
4. Check app password hasn't expired

## Prerequisites

Before using this collection, ensure:

1. Norest Mail backend is running
2. PostgreSQL is running and healthy
3. Stalwart Mail Server is running and healthy
4. Database migrations have been run
5. You have network access to localhost:8080 and localhost:8081

## Development Environment

This collection is configured for the development environment:

- API: http://localhost:8080/v1
- JMAP: http://localhost:8081/jmap
- Admin credentials: Development defaults

For production, create a new environment with production URLs and credentials.

## Automation

You can automate the collection using Newman (Postman CLI):

```bash
# Install Newman
npm install -g newman

# Run collection
newman run Norest-Mail.postman_collection.json \
  --environment Norest-Mail.postman_environment.json
```

## Security Notes

1. **Never commit environment files** with real credentials
2. **Use different environments** for development, staging, and production
3. **Rotate credentials regularly** in production
4. **Use secrets management** for production credentials
5. **Never expose admin credentials** to frontend or users
6. **Management APIs are server-side only** and should not be used from client applications

## Additional Resources

- [API_REFERENCE.md](../docs/API_REFERENCE.md) - Complete API documentation
- [AUTHENTICATION.md](../docs/AUTHENTICATION.md) - Authentication details
- [STARTUP.md](../docs/STARTUP.md) - Backend startup guide
- [HOW_TO_GET_MAIL.md](../docs/HOW_TO_GET_MAIL.md) - Getting mail guide
- [HOW_TO_SEND_MAIL.md](../docs/HOW_TO_SEND_MAIL.md) - Sending mail guide
