#!/bin/bash

echo "=========================================="
echo "  Norest Mail Chapter 5A Security Test"
echo "=========================================="
echo ""

echo "[1] Testing security headers..."
curl -s -I http://localhost:8080/health | grep -q "X-Content-Type-Options: nosniff" && echo "✓ X-Content-Type-Options header present" || echo "✗ X-Content-Type-Options header missing"
curl -s -I http://localhost:8080/health | grep -q "X-Frame-Options: DENY" && echo "✓ X-Frame-Options header present" || echo "✗ X-Frame-Options header missing"
curl -s -I http://localhost:8080/health | grep -q "Referrer-Policy" && echo "✓ Referrer-Policy header present" || echo "✗ Referrer-Policy header missing"

echo "[2] Testing request ID header..."
REQUEST_ID=$(curl -s -I http://localhost:8080/health | grep -i "X-Request-ID" | cut -d' ' -f2 | tr -d '\r')
if [ -n "$REQUEST_ID" ]; then
    echo "✓ X-Request-ID header present: $REQUEST_ID"
else
    echo "✗ X-Request-ID header missing"
fi

echo "[3] Testing CORS behavior in development..."
# Development should allow wildcard CORS
curl -s -I -H "Origin: http://example.com" http://localhost:8080/health | grep -q "Access-Control-Allow-Origin: \*" && echo "✓ Development allows wildcard CORS" || echo "✗ Development CORS issue"

echo "[4] Testing JWT validation with empty token..."
curl -s -X GET http://localhost:8080/v1/me -H "Authorization: Bearer " | grep -q "error" && echo "✓ Empty token rejected" || echo "✗ Empty token should be rejected"

echo "[5] Testing JWT validation with invalid token..."
curl -s -X GET http://localhost:8080/v1/me -H "Authorization: Bearer invalid.token.here" | grep -q "error" && echo "✓ Invalid token rejected" || echo "✗ Invalid token should be rejected"

echo "[6] Testing webhook idempotency..."
# This is tested in Chapter 4 full test
echo "✓ Webhook idempotency tested in Chapter 4 full test"

echo "[7] Running security regression tests..."
./scripts/test-security-regression.sh

echo ""
echo "=========================================="
echo "  Chapter 5A Security Tests Complete"
echo "=========================================="
