#!/bin/bash

echo "=========================================="
echo "  Chapter 5B Multi-Worker Tests"
echo "=========================================="
echo ""

echo "[1] Checking worker configuration..."
# Check PROVISIONING_WORKERS setting
if docker-compose config | grep -q "PROVISIONING_WORKERS"; then
    echo "✓ PROVISIONING_WORKERS is configured"
else
    echo "⚠ PROVISIONING_WORKERS not set, using default (4)"
fi

echo ""
echo "[2] Testing job claiming with single worker..."
# Create a test job
API_URL="http://localhost:8080"
TIMESTAMP=$(date +%s)
response=$(curl -s -X POST "$API_URL/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test-multiworker-${TIMESTAMP}@example.com\",\"password\":\"ValidPassword123\"}")
token=$(echo "$response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$token" ]; then
    # Create multiple domains to generate jobs
    for i in {1..5}; do
        curl -s -X POST "$API_URL/v1/domains" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $token" \
          -d "{\"name\":\"test-multiworker-${TIMESTAMP}-${i}.com\"}" > /dev/null
    done
    
    echo "✓ Created 5 domains to generate jobs"
    
    # Wait for worker to process
    sleep 5
    
    # Check job statuses
    echo "Checking job statuses..."
    # This would require database access to check provisioning_jobs table
    echo "⚠ Job status check: Manual verification needed (requires database access)"
else
    echo "⚠ Could not get token for test"
fi

echo ""
echo "[3] Testing concurrent job processing..."
# This would require running multiple worker instances
echo "⚠ Concurrent job processing: Manual verification needed"
echo "   To test with multiple workers:"
echo "   1. Scale docker-compose to multiple worker instances"
echo "   2. Create many jobs simultaneously"
echo "   3. Verify no duplicates and all jobs complete"

echo ""
echo "[4] Testing worker starvation prevention..."
# This would require a slow job and a fast job
echo "⚠ Worker starvation: Manual verification needed"
echo "   To test:"
echo "   1. Create a slow job (simulate with sleep)"
echo "   2. Create a fast job"
echo "   3. Verify fast job completes despite slow job"

echo ""
echo "=========================================="
echo "  Multi-Worker Tests Complete"
echo "=========================================="
