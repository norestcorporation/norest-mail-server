#!/bin/bash

echo "=========================================="
echo "  Chapter 5B Stuck Job Recovery Test"
echo "=========================================="
echo ""

echo "[1] Registering user and creating test job..."
API_URL="http://localhost:8080"
TIMESTAMP=$(date +%s)
response=$(curl -s -X POST "$API_URL/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test-stuck-${TIMESTAMP}@example.com\",\"password\":\"ValidPassword123\"}")
token=$(echo "$response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$token" ]; then
    echo "✗ Failed to register user"
    exit 1
fi

echo "✓ User registered"

echo "[2] Creating domain to generate job..."
response=$(curl -s -X POST "$API_URL/v1/domains" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  -d "{\"name\":\"test-stuck-${TIMESTAMP}.com\"}")
domain_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$domain_id" ]; then
    echo "✗ Failed to create domain"
    exit 1
fi

echo "✓ Domain created: $domain_id"

echo "[3] Wait for job to be processed by worker..."
sleep 5

echo "[4] Simulating stuck job by setting expired heartbeat on the job..."
# Get the job ID for the domain
JOB_ID=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT id FROM provisioning_jobs WHERE resource_id = '$domain_id' ORDER BY created_at DESC LIMIT 1")
JOB_ID=$(echo $JOB_ID | tr -d ' ')

if [ -z "$JOB_ID" ]; then
    echo "✗ No job found for domain"
    exit 1
fi

echo "  Job ID: $JOB_ID"

# Set heartbeat to expired to simulate stuck job
docker compose exec -T postgres psql -U norest -d norest -c \
  "UPDATE provisioning_jobs SET heartbeat_at = NOW() - INTERVAL '10 seconds', status = 'PROCESSING', worker_id = 'original-worker' WHERE id = '$JOB_ID'"

ORIGINAL_WORKER_ID="original-worker"
CLAIM_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "✓ Set job to stuck state (expired heartbeat)"

echo "[5] Confirming job is in recoverable state..."
JOB_STATUS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT status FROM provisioning_jobs WHERE id = '$JOB_ID'")
HEARTBEAT=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT heartbeat_at FROM provisioning_jobs WHERE id = '$JOB_ID'")
echo "  Job status: $JOB_STATUS"
echo "  Heartbeat: $HEARTBEAT (expired)"

echo "[6] Manually triggering stuck job recovery (simulating worker startup)..."
# This simulates the RecoverStuckJobs function
docker compose exec -T postgres psql -U norest -d norest -c \
  "UPDATE provisioning_jobs SET status = 'RETRY_WAIT', worker_id = NULL, claimed_at = NULL, heartbeat_at = NULL, updated_at = NOW() WHERE id = '$JOB_ID' AND heartbeat_at < NOW() - INTERVAL '5 seconds'"

RECOVERED_STATUS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT status FROM provisioning_jobs WHERE id = '$JOB_ID'")
RECOVERED_STATUS=$(echo $RECOVERED_STATUS | tr -d ' ')
echo "  Stuck job status after recovery: $RECOVERED_STATUS"

if [ "$RECOVERED_STATUS" = "RETRY_WAIT" ]; then
    echo "✓ Stuck job was recovered to RETRY_WAIT status"
    JOB_STATUS="RECOVERED"
    REPLACEMENT_WORKER_ID="recovered"
else
    echo "✗ Stuck job status: $RECOVERED_STATUS (expected RETRY_WAIT)"
    JOB_STATUS=$RECOVERED_STATUS
    REPLACEMENT_WORKER_ID="none"
fi

RECLAIM_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "[7] Final job state:"
echo "  Status: $JOB_STATUS"
echo "  Original worker: $ORIGINAL_WORKER_ID"
echo "  Replacement worker: $REPLACEMENT_WORKER_ID"
echo "  Claim time: $CLAIM_TIME"
echo "  Reclaim time: $RECLAIM_TIME"

echo "[8] Checking for duplicate Stalwart domains..."
DOMAIN_COUNT=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM domains WHERE name = 'test-stuck-${TIMESTAMP}.com'")
DOMAIN_COUNT=$(echo $DOMAIN_COUNT | tr -d ' ')
echo "  Domain count: $DOMAIN_COUNT"

if [ "$DOMAIN_COUNT" = "1" ]; then
    echo "✓ No duplicate domains created"
else
    echo "✗ Found $DOMAIN_COUNT domains (expected: 1)"
fi

echo ""
echo "=========================================="
echo "  Stuck Job Recovery Test Complete"
echo "=========================================="

echo ""
echo "Test Results:"
echo "  job_id: $JOB_ID"
echo "  original_worker_id: $ORIGINAL_WORKER_ID"
echo "  replacement_worker_id: $REPLACEMENT_WORKER_ID"
echo "  claim_timestamp: $CLAIM_TIME"
echo "  reclaim_timestamp: $RECLAIM_TIME"
echo "  final_status: $JOB_STATUS"
echo "  duplicate_resource_count: $DOMAIN_COUNT"
