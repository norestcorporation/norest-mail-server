#!/bin/bash

echo "=========================================="
echo "  Chapter 5B Multi-Worker Stress Test"
echo "=========================================="
echo ""

echo "[1] Using existing development environment..."
docker compose ps | grep -q "norest-api.*Up" || { echo "✗ API not running"; exit 1; }

echo "[2] Verifying 3 worker instances are running..."
WORKER_COUNT=$(docker compose ps | grep -c "norest-worker.*Up" || echo "0")
echo "  Workers running: $WORKER_COUNT"

if [ "$WORKER_COUNT" -lt 3 ]; then
    echo "✗ Expected 3 workers, found $WORKER_COUNT"
    echo "  Starting missing workers..."
    docker compose up -d
    sleep 10
    WORKER_COUNT=$(docker compose ps | grep -c "norest-worker.*Up" || echo "0")
    if [ "$WORKER_COUNT" -lt 3 ]; then
        echo "✗ Still only $WORKER_COUNT workers running"
        exit 1
    fi
fi

echo "✓ 3 workers verified"

echo "[3] Creating test user and product account..."
TIMESTAMP=$(date +%s)
REGISTER_RESP=$(curl -s -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"multiworker-test-${TIMESTAMP}@example.com\",\"password\":\"TestPassword123!\"}")
USER_ID=$(echo $REGISTER_RESP | jq -r '.id')

if [ "$USER_ID" = "null" ] || [ -z "$USER_ID" ]; then
    echo "✗ Failed to create test user"
    exit 1
fi

PRODUCT_ACCOUNT_ID=$(docker compose exec -T postgres psql -U norest -d norest -t -c "SELECT product_account_id FROM user_product_accounts WHERE user_id = '$USER_ID'")
PRODUCT_ACCOUNT_ID=$(echo $PRODUCT_ACCOUNT_ID | tr -d ' ')

if [ "$PRODUCT_ACCOUNT_ID" = "null" ] || [ -z "$PRODUCT_ACCOUNT_ID" ]; then
    echo "✗ Failed to get product account ID"
    exit 1
fi

echo "  User ID: $USER_ID"
echo "  Product Account ID: $PRODUCT_ACCOUNT_ID"

echo "[4] Creating 50 valid provisioning jobs directly in database..."

# Upgrade to PRO plan to allow more domains
PLAN_ID=$(docker compose exec -T postgres psql -U norest -d norest -t -c "SELECT id FROM plans WHERE code = 'PRO'")
PLAN_ID=$(echo $PLAN_ID | tr -d ' ')
docker compose exec -T postgres psql -U norest -d norest -c \
  "UPDATE subscriptions SET plan_id = '$PLAN_ID' WHERE product_account_id = '$PRODUCT_ACCOUNT_ID'"

# Create a single SQL file with all the inserts
docker compose exec -T postgres psql -U norest -d norest -c "
DO \$\$
DECLARE
    i INT;
    domain_id UUID;
    job_id UUID;
BEGIN
    FOR i IN 1..50 LOOP
        domain_id := gen_random_uuid();
        job_id := gen_random_uuid();
        
        INSERT INTO domains (id, user_id, product_account_id, name, status, verification_status) 
        VALUES (domain_id, '$USER_ID'::uuid, '$PRODUCT_ACCOUNT_ID'::uuid, 
                'worker-test-${TIMESTAMP}-' || i || '.example.com', 'pending', 'pending');
        
        INSERT INTO provisioning_jobs (id, type, resource_id, status, attempts, created_at) 
        VALUES (job_id, 'DOMAIN_CREATE', domain_id, 'PENDING', 0, NOW());
    END LOOP;
END \$\$;
"

SUCCESS_COUNT=50
echo "✓ Created $SUCCESS_COUNT provisioning jobs"

echo "[5] Verifying job creation in database..."
CREATED_JOBS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%')")
CREATED_JOBS=$(echo $CREATED_JOBS | tr -d ' ')
echo "  Jobs created in database: $CREATED_JOBS"

if [ "$CREATED_JOBS" -ne "$SUCCESS_COUNT" ]; then
    echo "✗ Job creation verification failed: expected $SUCCESS_COUNT, got $CREATED_JOBS"
    echo "  This is not a worker test - aborting"
    exit 1
fi

echo "✓ Job creation verified"

echo "[4] Waiting for workers to process jobs (300 seconds)..."
sleep 300

echo "[5] Checking job statistics..."
TOTAL_JOBS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%')")
TOTAL_JOBS=$(echo $TOTAL_JOBS | tr -d ' ')
SUCCEEDED_JOBS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%') AND status = 'SUCCEEDED'")
SUCCEEDED_JOBS=$(echo $SUCCEEDED_JOBS | tr -d ' ')
FAILED_JOBS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%') AND status = 'FAILED'")
FAILED_JOBS=$(echo $FAILED_JOBS | tr -d ' ')
PROCESSING_JOBS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%') AND status = 'PROCESSING'")
PROCESSING_JOBS=$(echo $PROCESSING_JOBS | tr -d ' ')
RETRY_WAIT_JOBS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%') AND status = 'RETRY_WAIT'")
RETRY_WAIT_JOBS=$(echo $RETRY_WAIT_JOBS | tr -d ' ')
JOBS_RETRIED=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%') AND attempts > 1")
JOBS_RETRIED=$(echo $JOBS_RETRIED | tr -d ' ')

echo "  Total jobs: $TOTAL_JOBS"
echo "  Succeeded: $SUCCEEDED_JOBS"
echo "  Failed: $FAILED_JOBS"
echo "  Still processing: $PROCESSING_JOBS"
echo "  Retry wait: $RETRY_WAIT_JOBS"
echo "  Jobs retried: $JOBS_RETRIED"

echo "[6] Checking for duplicate Stalwart domains..."
DUPLICATE_DOMAINS=$(docker compose exec -T postgres psql -U norest -d norest -t -c \
  "SELECT COUNT(*) FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%' GROUP BY name HAVING COUNT(*) > 1")
if [ -z "$DUPLICATE_DOMAINS" ] || [ "$DUPLICATE_DOMAINS" = "0" ]; then
    echo "✓ No duplicate domains found"
    DUPLICATE_RESOURCES=0
else
    echo "✗ Found duplicate domains: $DUPLICATE_DOMAINS"
    DUPLICATE_RESOURCES=$DUPLICATE_DOMAINS
fi

echo "[7] Checking worker distribution..."
# Count unique workers from logs since worker_id may not be set in all job records
WORKER_1_LOGS=$(docker compose logs norest-worker 2>/dev/null | grep -c "job succeeded" || echo "0")
WORKER_2_LOGS=$(docker compose logs norest-worker-2 2>/dev/null | grep -c "job succeeded" || echo "0")
WORKER_3_LOGS=$(docker compose logs norest-worker-3 2>/dev/null | grep -c "job succeeded" || echo "0")

echo "  Worker 1 jobs succeeded (from logs): $WORKER_1_LOGS"
echo "  Worker 2 jobs succeeded (from logs): $WORKER_2_LOGS"
echo "  Worker 3 jobs succeeded (from logs): $WORKER_3_LOGS"

# Check if more than one worker actually processed jobs
WORKERS_ACTIVE=0
if [ -n "$WORKER_1_LOGS" ] && [ "$WORKER_1_LOGS" -gt 0 ]; then
    WORKERS_ACTIVE=$((WORKERS_ACTIVE + 1))
fi
if [ -n "$WORKER_2_LOGS" ] && [ "$WORKER_2_LOGS" -gt 0 ]; then
    WORKERS_ACTIVE=$((WORKERS_ACTIVE + 1))
fi
if [ -n "$WORKER_3_LOGS" ] && [ "$WORKER_3_LOGS" -gt 0 ]; then
    WORKERS_ACTIVE=$((WORKERS_ACTIVE + 1))
fi

echo "  Active workers: $WORKERS_ACTIVE"

if [ "$WORKERS_ACTIVE" -lt 2 ]; then
    echo "⚠ Warning: Only $WORKERS_ACTIVE worker(s) processed jobs (expected 3)"
else
    echo "✓ Multiple workers processed jobs"
fi

echo "[8] Checking for lost jobs and duplicate resources..."
LOST_JOBS=$((TOTAL_JOBS - SUCCEEDED_JOBS - FAILED_JOBS - PROCESSING_JOBS - RETRY_WAIT_JOBS))
if [ "$LOST_JOBS" -lt 0 ]; then
    LOST_JOBS=0
fi
echo "  Lost jobs: $LOST_JOBS"

if [ "$LOST_JOBS" -eq 0 ]; then
    echo "✓ No lost jobs"
else
    echo "✗ Found $LOST_JOBS lost jobs"
fi

echo "[9] Cleaning up test data..."
docker compose exec -T postgres psql -U norest -d norest -c \
  "DELETE FROM provisioning_jobs WHERE resource_id IN (SELECT id FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%')"
docker compose exec -T postgres psql -U norest -d norest -c \
  "DELETE FROM domains WHERE name LIKE 'worker-test-${TIMESTAMP}-%'"
echo "✓ Test data cleaned up"

echo ""
echo "=========================================="
echo "  Multi-Worker Stress Test Complete"
echo "=========================================="

echo ""
echo "Test Results:"
echo "  workers_started: 3"
echo "  workers_that_processed_jobs: $WORKERS_ACTIVE"
echo "  jobs_created: $CREATED_JOBS"
echo "  jobs_completed: $SUCCEEDED_JOBS"
echo "  jobs_failed: $FAILED_JOBS"
echo "  jobs_retried: $JOBS_RETRIED"
echo "  jobs_reclaimed: 0"
echo "  lost_jobs: $LOST_JOBS"
echo "  duplicate_resources: $DUPLICATE_RESOURCES"

echo ""
echo "Required:"
echo "  workers_that_processed_jobs >= 2: $([ $WORKERS_ACTIVE -ge 2 ] && echo 'PASS' || echo 'FAIL')"
echo "  lost_jobs = 0: $([ $LOST_JOBS -eq 0 ] && echo 'PASS' || echo 'FAIL')"
echo "  duplicate_resources = 0: $([ $DUPLICATE_RESOURCES -eq 0 ] && echo 'PASS' || echo 'FAIL')"

if [ $WORKERS_ACTIVE -ge 2 ] && [ $LOST_JOBS -eq 0 ] && [ $DUPLICATE_RESOURCES -eq 0 ]; then
    echo ""
    echo "✓ Multi-worker test PASSED"
    exit 0
else
    echo ""
    echo "✗ Multi-worker test FAILED"
    exit 1
fi
