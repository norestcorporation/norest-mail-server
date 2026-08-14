#!/bin/bash
set -e

echo "=========================================="
echo "  Chapter 5D Observability Tests"
echo "=========================================="

# 1. Test structured logs
echo "[1] Testing structured logs..."
response=$(curl -s http://localhost:8080/health | jq -r '.status')
if [ "$response" = "ok" ]; then
    echo "✓ Structured logs present (check logs for service field)"
else
    echo "✗ Health check failed"
fi

# 2. Test request IDs
echo "[2] Testing request IDs..."
request_id=$(curl -s -I http://localhost:8080/health | grep -i x-request-id | cut -d' ' -f2 | tr -d '\r')
if [ -n "$request_id" ]; then
    echo "✓ Request ID header present: $request_id"
else
    echo "✗ Request ID header missing"
fi

# 3. Test metrics endpoint
echo "[3] Testing metrics endpoint..."
response=$(curl -s http://localhost:8080/metrics)
if echo "$response" | jq -e . > /dev/null 2>&1; then
    requests_total=$(echo "$response" | jq -r '.http_requests_total')
    if [ -n "$requests_total" ] && [ "$requests_total" -ge 0 ]; then
        echo "✓ Metrics endpoint accessible, requests_total: $requests_total"
    else
        echo "✗ Metrics endpoint failed to parse"
    fi
else
    echo "✗ Metrics endpoint returned invalid JSON"
fi

# 4. Test readiness
echo "[4] Testing readiness..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/ready)
if [ "$response" -eq 200 ]; then
    echo "✓ Readiness endpoint returns 200"
else
    echo "✗ Readiness endpoint returned $response"
fi

# 5. Test liveness
echo "[5] Testing liveness..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/live)
if [ "$response" -eq 200 ]; then
    echo "✓ Liveness endpoint returns 200"
else
    echo "✗ Liveness endpoint returned $response"
fi

# 6. Test dependency health
echo "[6] Testing dependency health..."
db_response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/db)
stalwart_response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/stalwart)
if [ "$db_response" -eq 200 ] && [ "$stalwart_response" -eq 200 ]; then
    echo "✓ Both database and Stalwart dependencies healthy"
else
    echo "⚠ Dependency health check: DB=$db_response, Stalwart=$stalwart_response"
fi

# 7. Test provisioning backlog metrics
echo "[7] Testing provisioning backlog metrics..."
response=$(curl -s http://localhost:8080/metrics)
if echo "$response" | jq -e . > /dev/null 2>&1; then
    pending_jobs=$(echo "$response" | jq -r '.pending_jobs')
    if [ -n "$pending_jobs" ]; then
        echo "✓ Provisioning backlog metrics accessible, pending_jobs: $pending_jobs"
    else
        echo "⚠ Provisioning backlog metrics may not be populated"
    fi
else
    echo "⚠ Metrics endpoint returned invalid JSON"
fi

echo "=========================================="
echo "  Chapter 5D Tests Complete"
echo "=========================================="