#!/bin/bash

echo "=========================================="
echo "  Chapter 5B Account Lifecycle Test"
echo "=========================================="
echo ""

API_URL="http://localhost:8080"

echo "[1] Registering user and creating account..."
TIMESTAMP=$(date +%s)
response=$(curl -s -X POST "$API_URL/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test-lifecycle-${TIMESTAMP}@example.com\",\"password\":\"ValidPassword123\"}")
token=$(echo "$response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$token" ]; then
    echo "✗ Failed to register user"
    exit 1
fi

echo "✓ User registered"

echo "[2] Creating domain..."
response=$(curl -s -X POST "$API_URL/v1/domains" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  -d "{\"name\":\"test-lifecycle-${TIMESTAMP}.com\"}")
domain_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$domain_id" ]; then
    echo "✗ Failed to create domain"
    exit 1
fi

echo "✓ Domain created: $domain_id"

echo "[3] Waiting for domain provisioning..."
# Poll for domain to become active
MAX_ATTEMPTS=60
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    STATUS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
      "SELECT status FROM domains WHERE id = '$domain_id'")
    if [ "$STATUS" = "active" ]; then
        echo "✓ Domain is active"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    sleep 1
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo "✗ Domain did not become active in time"
    exit 1
fi

echo "[4] Creating address/mailbox..."
response=$(curl -s -X POST "$API_URL/v1/addresses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  -d "{\"domain_id\":\"$domain_id\",\"local_part\":\"lifecycle-test\"}")
address_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$address_id" ]; then
    echo "✗ Failed to create address"
    exit 1
fi

echo "✓ Address created: $address_id"

echo "[5] Waiting for account provisioning..."
sleep 5

echo "[6] Getting mailbox ID and stalwart_account_id..."
# Query database for mailbox ID and stalwart_account_id
MAILBOX_INFO=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT id, stalwart_account_id FROM mailboxes WHERE address_id = '$address_id'")

MAILBOX_ID=$(echo "$MAILBOX_INFO" | awk '{print $1}')
STALWART_ACCOUNT_ID=$(echo "$MAILBOX_INFO" | awk '{print $2}')

if [ -z "$MAILBOX_ID" ] || [ "$MAILBOX_ID" == "(no" ]; then
    echo "✗ Failed to get mailbox ID"
    exit 1
fi

echo "✓ Mailbox ID: $MAILBOX_ID, Stalwart Account ID: $STALWART_ACCOUNT_ID"

echo "[7] Testing ACCOUNT_DISABLE idempotency..."
# Disable account twice via database job insertion
for i in 1 2; do
    echo "  Disable attempt $i..."
    docker compose exec -T postgres psql -U norest -d norest -c \
      "INSERT INTO provisioning_jobs (type, resource_id, status) VALUES ('ACCOUNT_DISABLE', '$MAILBOX_ID', 'PENDING')" > /dev/null
    
    # Wait for worker to process
    sleep 3
    
    # Check mailbox status
    STATUS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
      "SELECT status FROM mailboxes WHERE id = '$MAILBOX_ID'")
    
    if [ "$i" = "1" ]; then
        if [ "$STATUS" = "disabled" ]; then
            echo "    ✓ Account disabled (first attempt)"
        else
            echo "    ⚠ Account status: $STATUS (expected: disabled)"
        fi
    else
        if [ "$STATUS" = "disabled" ]; then
            echo "    ✓ Account disable idempotent (second attempt - same state)"
        else
            echo "    ⚠ Account status: $STATUS (expected: disabled)"
        fi
    fi
done

echo "[8] Testing ACCOUNT_REACTIVATE idempotency..."
# Reactivate account twice via database job insertion
for i in 1 2; do
    echo "  Reactivate attempt $i..."
    docker compose exec -T postgres psql -U norest -d norest -c \
      "INSERT INTO provisioning_jobs (type, resource_id, status) VALUES ('ACCOUNT_ENABLE', '$MAILBOX_ID', 'PENDING')" > /dev/null
    
    # Wait for worker to process
    sleep 3
    
    # Check mailbox status
    STATUS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
      "SELECT status FROM mailboxes WHERE id = '$MAILBOX_ID'")
    
    if [ "$i" = "1" ]; then
        if [ "$STATUS" = "active" ]; then
            echo "    ✓ Account reactivated (first attempt)"
        else
            echo "    ⚠ Account status: $STATUS (expected: active)"
        fi
    else
        if [ "$STATUS" = "active" ]; then
            echo "    ✓ Account reactivate idempotent (second attempt - same state)"
        else
            echo "    ⚠ Account status: $STATUS (expected: active)"
        fi
    fi
done

echo ""
echo "=========================================="
echo "  Account Lifecycle Test Complete"
echo "=========================================="
