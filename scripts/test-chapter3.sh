#!/bin/bash
set -e

API_URL="http://localhost:8080/v1"

echo "============================================"
echo "  Norest Mail Chapter 3 - Session Auth Test"
echo "============================================"

# Wait for API to be ready
echo "Waiting for API..."
until curl -s $API_URL/health > /dev/null; do
    sleep 1
done

TIMESTAMP=$(date +%s%N)
ALICE_EMAIL="alice_${TIMESTAMP}@example.com"
PASSWORD="SecurePassword123!"

echo "1. Registering Alice ($ALICE_EMAIL)..."
RES=$(curl -s -X POST $API_URL/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ALICE_EMAIL\", \"password\":\"$PASSWORD\"}")

# Extract token
ACCESS_TOKEN=$(echo $RES | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
    echo "Failed to register/get access token"
    echo $RES
    exit 1
fi

echo "2. Creating domain..."
DOMAIN_RES=$(curl -s -X POST $API_URL/domains \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"norest-${TIMESTAMP}.test\"}")

DOMAIN_ID=$(echo $DOMAIN_RES | grep -o '"id":"[^"]*' | cut -d'"' -f4)

echo "Waiting for domain to be active..."
for i in {1..30}; do
    DOMAIN_GET=$(curl -s -X GET $API_URL/domains/$DOMAIN_ID -H "Authorization: Bearer $ACCESS_TOKEN")
    STATUS=$(echo $DOMAIN_GET | grep -o '"status":"[^"]*"' || true)
    if [[ "$STATUS" == '"status":"active"' ]]; then
        echo "Domain is active!"
        break
    fi
    sleep 1
done

echo "3. Creating address..."
ADDR_RES=$(curl -s -X POST $API_URL/domains/$DOMAIN_ID/addresses \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"local_part\":\"alice\"}")

echo "Waiting for worker to provision resources..."
for i in {1..30}; do
    ACCOUNT_RES=$(curl -s -X GET $API_URL/mail/account -H "Authorization: Bearer $ACCESS_TOKEN")
    STATUS=$(echo $ACCOUNT_RES | grep -o '"status":"[^"]*"' || true)
    if [[ "$STATUS" == '"status":"active"' ]]; then
        echo "Mailbox is active!"
        break
    fi
    sleep 1
done

echo "4. Getting mail account..."
echo $ACCOUNT_RES | grep -o '"status":"[^"]*"' || true

echo "5. Creating mail session..."
SESSION_RES=$(curl -s -X POST $API_URL/mail/session \
  -H "Authorization: Bearer $ACCESS_TOKEN")

echo "Session response:"
echo $SESSION_RES | jq . || echo $SESSION_RES

SESSION_TOKEN=$(echo $SESSION_RES | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
ACCOUNT_ID=$(echo $SESSION_RES | grep -o '"account_id":"[^"]*' | cut -d'"' -f4)
JMAP_URL=$(echo $SESSION_RES | grep -o '"jmap_session_url":"[^"]*' | cut -d'"' -f4)

if [ -z "$SESSION_TOKEN" ]; then
    echo "Failed to get session token"
    exit 1
fi

echo "6. Testing direct JMAP access with session token..."
echo "Using Bearer $SESSION_TOKEN to $JMAP_URL..."

# The JMAP session is fetched
curl -s -v -H "Authorization: Bearer $SESSION_TOKEN" $JMAP_URL > /tmp/jmap_out 2>&1
if grep -q "307" /tmp/jmap_out || grep -q "200" /tmp/jmap_out; then
    echo "JMAP authorization accepted!"
else
    echo "JMAP authorization failed."
    cat /tmp/jmap_out
    exit 1
fi

echo "============================================"
echo "  Success!"
echo "============================================"
