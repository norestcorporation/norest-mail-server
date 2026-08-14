#!/bin/bash
set -e

echo "============================================"
echo "  Norest Mail Chapter 2 E2E Test"
echo "============================================"

# Wait for API to be ready
echo "Waiting for API..."
until curl -s http://localhost:8080/health > /dev/null; do
    sleep 1
done

# 1. Register User
echo "1. Registering User..."
RES=$(curl -s -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "securepassword"}')
echo $RES | jq .

# 2. Login User
echo "2. Logging In..."
LOGIN_RES=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "securepassword"}')
echo $LOGIN_RES | jq .

TOKEN=$(echo $LOGIN_RES | jq -r .access_token)

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "Login failed!"
    exit 1
fi

# 3. Create Domain
echo "3. Creating Domain (norest.test)..."
DOMAIN_RES=$(curl -s -X POST http://localhost:8080/v1/domains \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "norest.test"}')
echo $DOMAIN_RES | jq .

DOMAIN_ID=$(echo $DOMAIN_RES | jq -r .id)

# 4. Create Address
echo "4. Creating Address (alice)..."
ADDRESS_RES=$(curl -s -X POST http://localhost:8080/v1/domains/$DOMAIN_ID/addresses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"local_part": "alice"}')
echo $ADDRESS_RES | jq .

echo "Waiting for worker to provision resources (5s)..."
sleep 5

echo "5. Checking Domain Status..."
curl -s -X GET http://localhost:8080/v1/domains/$DOMAIN_ID \
  -H "Authorization: Bearer $TOKEN" | jq .

echo "6. Checking Address Status..."
curl -s -X GET http://localhost:8080/v1/domains/$DOMAIN_ID/addresses \
  -H "Authorization: Bearer $TOKEN" | jq .

echo "============================================"
echo "  E2E Test Complete"
echo "============================================"
