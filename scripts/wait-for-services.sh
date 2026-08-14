#!/usr/bin/env bash
set -euo pipefail

# wait-for-services.sh — utility for waiting on service readiness

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

TIMEOUT="${1:-60}"

echo "Waiting for services (timeout: ${TIMEOUT}s)..."

# Wait for PostgreSQL
echo -n "  PostgreSQL..."
for i in $(seq 1 "$TIMEOUT"); do
    if docker compose exec -T postgres pg_isready -U norest -d norest &>/dev/null; then
        echo " OK"
        break
    fi
    if [ "$i" -eq "$TIMEOUT" ]; then
        echo " TIMEOUT"
        exit 1
    fi
    sleep 1
done

# Wait for Stalwart
echo -n "  Stalwart..."
for i in $(seq 1 "$TIMEOUT"); do
    if curl -s -o /dev/null http://localhost:8081/ 2>/dev/null; then
        echo " OK"
        break
    fi
    if [ "$i" -eq "$TIMEOUT" ]; then
        echo " TIMEOUT"
        exit 1
    fi
    sleep 1
done

# Wait for Norest API
echo -n "  Norest API..."
for i in $(seq 1 "$TIMEOUT"); do
    if curl -s http://localhost:8080/health 2>/dev/null | grep -q '"ok"'; then
        echo " OK"
        break
    fi
    if [ "$i" -eq "$TIMEOUT" ]; then
        echo " TIMEOUT"
        exit 1
    fi
    sleep 1
done

echo "All services ready."
