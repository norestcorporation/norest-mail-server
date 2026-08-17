# Norest Mail API Collections - Environment Variables & Automation

This document describes the environment variables used in the Norest Mail API collections for both Postman and Scalar (OpenAPI/Swagger), along with the automation features included in the Postman collections.

## Automation Features

The Postman collections include powerful automation scripts that:

### 1. **Automatic Token Extraction**
- **Login/Register**: Automatically extracts `access_token` from successful authentication responses
- **User Information**: Auto-populates `account_id` and `email` from user responses
- **No Manual Copy-Paste**: Tokens are automatically stored in environment variables

### 2. **Smart Variable Population**
- **Domain Selection**: Automatically selects the first available domain from list responses
- **Address Reservation**: Auto-captures `reservation_id` and `email_address` when reserving
- **Mailbox Detection**: Automatically identifies and stores mailbox IDs (inbox, sent, drafts, trash, spam)
- **Message Selection**: Auto-selects the first message from message lists for testing
- **Thread Detection**: Auto-captures thread IDs from thread lists

### 3. **Idempotency Key Generation**
- **Auto-Generation**: Automatically generates unique idempotency keys for send operations
- **Pre-request Script**: Generates keys using timestamp + random string format
- **No Manual Entry**: `idempotency_key` is automatically populated before send requests

### 4. **Session Management**
- **Mail Session**: Auto-captures session tokens and account IDs
- **Provisioning Status**: Auto-updates provisioning status from responses
- **Sync State**: Auto-captures sync state tokens for incremental sync

## How to Use the Automation

### Quick Start Guide

1. **Import the Collection and Environment**
   - Import `Norest_Owned_Domain.postman_collection.json` or `Norest_Custom_Domain.postman_collection.json`
   - Import `Norest_Mail_Environment.postman_environment.json`
   - Select the "Norest Mail Environment" in Postman

2. **Set Initial Variables**
   - Set `base_url` to your server URL (default: `http://localhost:8080`)
   - Set `email` to your test email address
   - Set `password` to your test password

3. **Run Authentication Flow**
   - Execute "Register" or "Login" request
   - The script automatically extracts and stores `access_token`
   - `account_id` and `email` are auto-populated

4. **Follow the Domain Flow**
   - For Platform Domain: Run "List Available Domains" → auto-selects first domain
   - For Custom Domain: Set `domain_name` → Run "Add Domain" → auto-captures domain details

5. **Complete Email Setup**
   - Set `username` → Run "Check Username Availability"
   - Run "Reserve Email Address" → auto-captures `reservation_id` and `email_address`
   - Run "Claim Address" to complete the setup

6. **Mail Operations**
   - Run "Create Mail Session" → auto-captures mail account details
   - Run "List Mailboxes" → auto-detects and stores all mailbox IDs
   - Run "List Messages" → auto-selects first message for testing
   - All subsequent requests use auto-populated variables

### Platform Domain Flow (Automated)

1. **Set Initial Variables**
   ```
   base_url = http://localhost:8080
   email = your-email@norestmail.com
   password = your-password
   ```

2. **Authentication**
   - Run: `POST /v1/auth/register` or `POST /v1/auth/login`
   - Auto-sets: `access_token`, `account_id`, `email`

3. **Domain Selection**
   - Run: `GET /v1/domains/platform`
   - Auto-sets: `domain_id`, `domain_name` (first available domain)

4. **Email Address Setup**
   - Set: `username = your-username`
   - Run: `GET /v1/domains/{{domain_id}}/addresses/check/{{username}}`
   - Run: `POST /v1/domains/{{domain_id}}/addresses/reserve`
   - Auto-sets: `reservation_id`, `email_address`
   - Run: `POST /v1/addresses/{{reservation_id}}/claim`

5. **Mail Provisioning**
   - Run: `POST /v1/mail/session`
   - Auto-sets: `mail_account_id`, `session_token`
   - Run: `GET /v1/mail/account`
   - Auto-sets: `mailbox_id`, `mail_account_id`

6. **Mailbox Setup**
   - Run: `GET /v1/mail/mailboxes`
   - Auto-sets: `inbox_id`, `sent_id`, `drafts_id`, `trash_id`, `spam_id`

7. **Start Using Mail**
   - Run: `GET /v1/mail/messages?mailbox_id={{inbox_id}}`
   - Auto-sets: `message_id`, `thread_id` (first message)
   - All other requests now work with auto-populated variables

### Custom Domain Flow (Automated)

1. **Set Initial Variables**
   ```
   base_url = http://localhost:8080
   email = your-email@yourdomain.com
   password = your-password
   domain_name = yourdomain.com
   ```

2. **Authentication**
   - Run: `POST /v1/auth/register` or `POST /v1/auth/login`
   - Auto-sets: `access_token`, `account_id`, `email`

3. **Domain Setup**
   - Run: `POST /v1/domains`
   - Auto-sets: `domain_id`, `domain_name`
   - Run: `POST /v1/domains/{{domain_id}}/verification/start`
   - Configure DNS records as instructed
   - Run: `GET /v1/domains/{{domain_id}}/verification` (check status)

4. **Email Address Setup**
   - Set: `username = your-username`
   - Run: `GET /v1/domains/{{domain_id}}/addresses/check/{{username}}`
   - Run: `POST /v1/domains/{{domain_id}}/addresses/reserve`
   - Auto-sets: `reservation_id`, `email_address`
   - Run: `POST /v1/addresses/{{reservation_id}}/claim`

5. **Continue with Mail Provisioning**
   - Same as Platform Domain flow from step 5

## Automation Script Details

### Token Extraction Scripts
```javascript
// Automatically runs after successful login/register
if (pm.response.code === 200 || pm.response.code === 201) {
    const jsonData = pm.response.json();
    if (jsonData.access_token) {
        pm.environment.set('access_token', jsonData.access_token);
    }
    if (jsonData.user && jsonData.user.id) {
        pm.environment.set('account_id', jsonData.user.id);
    }
    if (jsonData.user && jsonData.user.email) {
        pm.environment.set('email', jsonData.user.email);
    }
}
```

### Smart Selection Scripts
```javascript
// Automatically selects first item from arrays
if (pm.response.code === 200) {
    const jsonData = pm.response.json();
    if (Array.isArray(jsonData) && jsonData.length > 0) {
        pm.environment.set('domain_id', jsonData[0].id);
        pm.environment.set('domain_name', jsonData[0].name);
    }
}
```

### Mailbox Detection Scripts
```javascript
// Automatically identifies mailbox roles
if (pm.response.code === 200) {
    const jsonData = pm.response.json();
    if (jsonData.data && Array.isArray(jsonData.data)) {
        jsonData.data.forEach(mailbox => {
            if (mailbox.role === 'inbox') {
                pm.environment.set('inbox_id', mailbox.id);
            } else if (mailbox.role === 'sent') {
                pm.environment.set('sent_id', mailbox.id);
            } else if (mailbox.role === 'drafts') {
                pm.environment.set('drafts_id', mailbox.id);
            } else if (mailbox.role === 'trash') {
                pm.environment.set('trash_id', mailbox.id);
            } else if (mailbox.role === 'spam') {
                pm.environment.set('spam_id', mailbox.id);
            }
        });
    }
}
```

### Idempotency Key Generation
```javascript
// Automatically generates unique keys before send requests
const idempotencyKey = 'req_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
pm.environment.set('idempotency_key', idempotencyKey);
```

## Collection Files

### Postman Collections
- **Norest_Owned_Domain.postman_collection.json** - Platform domain flow with automation
- **Norest_Custom_Domain.postman_collection.json** - Custom domain flow with automation
- **Norest_Mail_Environment.postman_environment.json** - Pre-configured environment

### Scalar/OpenAPI Collections
- **Norest_Owned_Domain.openapi.json** - OpenAPI spec for Platform Domain
- **Norest_Custom_Domain.openapi.json** - OpenAPI spec for Custom Domain

### Documentation
- **ENVIRONMENT_VARIABLES.md** - This comprehensive guide

## Manual vs Automated Mode

### Manual Mode
- Set all variables manually
- Copy tokens from responses
- Full control over variable values
- Better for specific test scenarios

### Automated Mode (Default)
- Set only initial variables (`base_url`, `email`, `password`, `domain_name`)
- Let scripts handle the rest
- Faster workflow for testing
- Ideal for API exploration

### Switching Between Modes
- Automation scripts are non-invasive
- You can manually override any auto-set variable
- Scripts only update variables if they're empty or from successful responses
- No conflicts between manual and automated operation

## Complete Variable Reference

### Base Configuration
- **`base_url`** - The base URL of the Norest Mail API server
  - Default: `http://localhost:8080`
  - Description: Root URL for all API endpoints
  - Example: `https://api.norestmail.com`

### Authentication
- **`email`** - User email address for authentication
  - Type: string
  - Description: Email address used for registration and login
  - Example: `user@norestmail.com`

- **`password`** - User password for authentication
  - Type: string
  - Description: Password used for registration and login
  - Example: `securePassword123`

- **`access_token`** - JWT access token
  - Type: string
  - Description: Bearer token obtained from login, used for authenticated requests
  - Example: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`

### Domain Information
- **`domain_id`** - Unique identifier for a domain
  - Type: string (UUID)
  - Description: ID of the domain (platform or custom)
  - Example: `123e4567-e89b-12d3-a456-426614174000`

- **`domain_name`** - Domain name
  - Type: string
  - Description: The actual domain name (e.g., for custom domains)
  - Example: `example.com`

### Address Configuration
- **`username`** - Local part of email address
  - Type: string
  - Description: Username part before the @ symbol
  - Example: `john.doe`

- **`email_address`** - Complete email address
  - Type: string
  - Description: Full email address (username@domain)
  - Example: `john.doe@norestmail.com`

- **`reservation_id`** - Address reservation identifier
  - Type: string (UUID)
  - Description: ID of a reserved email address
  - Example: `123e4567-e89b-12d3-a456-426614174001`

### Account Information
- **`account_id`** - User account identifier
  - Type: string (UUID)
  - Description: ID of the user account
  - Example: `123e4567-e89b-12d3-a456-426614174002`

- **`mail_account_id`** - Mail account identifier
  - Type: string
  - Description: ID of the mail account in the mail system
  - Example: `acc_1234567890`

- **`provisioning_status`** - Mail account provisioning status
  - Type: string
  - Description: Current status of mail account provisioning
  - Values: `pending`, `provisioning`, `active`, `failed`
  - Example: `active`

### Mailbox Information
- **`mailbox_id`** - Mailbox identifier
  - Type: string (UUID)
  - Description: ID of a specific mailbox
  - Example: `123e4567-e89b-12d3-a456-426614174003`

### Standard Mailboxes
- **`inbox_id`** - Inbox mailbox ID
  - Type: string (UUID)
  - Description: ID of the inbox mailbox
  - Example: `123e4567-e89b-12d3-a456-426614174004`

- **`sent_id`** - Sent mailbox ID
  - Type: string (UUID)
  - Description: ID of the sent items mailbox
  - Example: `123e4567-e89b-12d3-a456-426614174005`

- **`drafts_id`** - Drafts mailbox ID
  - Type: string (UUID)
  - Description: ID of the drafts mailbox
  - Example: `123e4567-e89b-12d3-a456-426614174006`

- **`trash_id`** - Trash mailbox ID
  - Type: string (UUID)
  - Description: ID of the trash mailbox
  - Example: `123e4567-e89b-12d3-a456-426614174007`

- **`spam_id`** - Spam mailbox ID
  - Type: string (UUID)
  - Description: ID of the spam mailbox
  - Example: `123e4567-e89b-12d3-a456-426614174008`

### Message Information
- **`message_id`** - Message identifier
  - Type: string (UUID)
  - Description: ID of a specific email message
  - Example: `123e4567-e89b-12d3-a456-426614174009`

- **`thread_id`** - Thread identifier
  - Type: string (UUID)
  - Description: ID of a conversation thread
  - Example: `123e4567-e89b-12d3-a456-426614174010`

### Draft Information
- **`draft_id`** - Draft identifier
  - Type: string (UUID)
  - Description: ID of a draft message
  - Example: `123e4567-e89b-12d3-a456-426614174011`

### Attachment Information
- **`attachment_id`** - Attachment identifier
  - Type: string
  - Description: Blob ID of an uploaded attachment
  - Example: `blob_1234567890abcdef`

### Synchronization
- **`sync_state`** - Synchronization state token
  - Type: string
  - Description: Token used for incremental sync operations
  - Example: `sync_state_token_abc123`

### Request Safety
- **`idempotency_key`** - Idempotency key for safe retries
  - Type: string
  - Description: Unique key to ensure request idempotency (prevents duplicate sends)
  - Example: `req_20240817_123456_abc123`

### Platform Domain Collection
The Norest_Owned_Domain collection focuses on platform-provided domains and includes:
- Platform domain listing and selection
- Simplified email address reservation
- No domain verification steps required

### Custom Domain Collection
The Norest_Custom_Domain collection includes additional variables and steps for:
- Custom domain addition
- Domain verification process
- DNS configuration requirements
- Extended registration flow

## Usage Instructions

### Postman Setup
1. Import the collection JSON file into Postman
2. Configure the environment variables in Postman
3. Set initial values for `base_url`, `email`, and `password`
4. Run the authentication requests to obtain `access_token`
5. Use subsequent requests to populate other variables as needed

### Scalar/OpenAPI Setup
1. Import the OpenAPI JSON file into Scalar or other OpenAPI-compatible tools
2. Configure the server URL and authentication
3. Use the provided examples to understand request/response formats
4. Set up environment variables in your API client

## Variable Initialization Flow

### Platform Domain Flow
1. Set `base_url`, `email`, `password`
2. Call Register/Login → get `access_token`
3. Call List Platform Domains → get `domain_id`
4. Set `username` → call Check Availability
5. Call Reserve Address → get `reservation_id`
6. Call Claim Address → get mailbox IDs
7. Call Create Mail Session → get `mail_account_id`
8. Use mail variables for subsequent operations

### Custom Domain Flow
1. Set `base_url`, `email`, `password`
2. Call Register/Login → get `access_token`
3. Set `domain_name` → call Add Domain → get `domain_id`
4. Call Start Verification → configure DNS
5. Call Get Verification Status → wait for verification
6. Set `username` → call Check Availability
7. Call Reserve Address → get `reservation_id`
8. Call Claim Address → get mailbox IDs
9. Call Create Mail Session → get `mail_account_id`
10. Use mail variables for subsequent operations

## Security Notes

- Never commit actual values for `password` or `access_token` to version control
- Use different environment configurations for development, staging, and production
- Rotate `access_token` periodically and implement proper token refresh logic
- Use strong, unique `idempotency_key` values for each send operation
- Keep `base_url` updated to match your deployment environment

## Troubleshooting

### Common Issues
- **401 Unauthorized**: Check that `access_token` is valid and not expired
- **404 Not Found**: Verify that resource IDs (`domain_id`, `message_id`, etc.) are correct
- **400 Bad Request**: Ensure request bodies match the expected schema
- **409 Conflict**: Check for duplicate operations (use `idempotency_key` for sends)
- **503 Service Unavailable**: Verify that the mail system is fully provisioned

### Variable Persistence
- Ensure variables are properly scoped (collection vs environment vs global)
- Check that variable names match exactly (case-sensitive)
- Verify that variable values are properly URL-encoded when used in query parameters