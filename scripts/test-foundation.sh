#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

NOREST_URL="${NOREST_URL:-http://localhost:8080}"
STALWART_URL="${STALWART_URL:-http://localhost:8081}"

PASS=0
FAIL=0

check() {
    local name="$1"
    local result="$2"
    if [ "$result" = "pass" ]; then
        echo "  ✓ $name"
        PASS=$((PASS + 1))
    else
        echo "  ✗ $name"
        FAIL=$((FAIL + 1))
    fi
}

echo "============================================"
echo "  Norest Mail Foundation Tests"
echo "============================================"
echo ""

# Test 1: /health
echo "Testing health endpoints..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$NOREST_URL/health" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    check "GET /health" "pass"
else
    check "GET /health (HTTP $HTTP_CODE)" "fail"
fi

# Test 2: /health/db
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$NOREST_URL/health/db" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    check "GET /health/db" "pass"
else
    check "GET /health/db (HTTP $HTTP_CODE)" "fail"
fi

# Test 3: /health/stalwart
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$NOREST_URL/health/stalwart" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    check "GET /health/stalwart" "pass"
else
    check "GET /health/stalwart (HTTP $HTTP_CODE)" "fail"
fi

# Test 4: Stalwart JMAP discovery
echo ""
echo "Testing Stalwart JMAP..."
# Stalwart v0.16 redirects /.well-known/jmap → /jmap/session (307).
# Follow the redirect with -L and basic auth to get the session resource.
HTTP_CODE=$(curl -s -L -o /dev/null -w "%{http_code}" -u "admin:change-me-development-only" "$STALWART_URL/.well-known/jmap" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "307" ] || [ "$HTTP_CODE" = "401" ]; then
    check "Stalwart JMAP discovery (/.well-known/jmap)" "pass"
else
    check "Stalwart JMAP discovery (HTTP $HTTP_CODE)" "fail"
fi

# Test 5: Stalwart JMAP endpoint reachability
# /jmap is a POST endpoint; GET returns 404. Test the JMAP session endpoint instead.
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -u "admin:change-me-development-only" "$STALWART_URL/jmap/session" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ]; then
    check "Stalwart JMAP session (/jmap/session)" "pass"
else
    check "Stalwart JMAP session (HTTP $HTTP_CODE)" "fail"
fi

# Summary
echo ""
echo "============================================"
TOTAL=$((PASS + FAIL))
echo "  Results: $PASS/$TOTAL passed"
if [ "$FAIL" -gt 0 ]; then
    echo "  SOME TESTS FAILED"
    echo "============================================"
    exit 1
fi
echo "  ALL TESTS PASSED"
echo "============================================"
