#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "============================================"
echo "  Norest Mail development environment"
echo "============================================"
echo ""

# 1. Verify Docker is installed
if ! command -v docker &>/dev/null; then
    echo "ERROR: Docker is not installed."
    exit 1
fi
echo "✓ Docker found"

# 2. Verify Docker Compose is available
if ! docker compose version &>/dev/null; then
    echo "ERROR: Docker Compose is not available."
    exit 1
fi
echo "✓ Docker Compose found"

# 3. Create .env from .env.example if missing
if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo "✓ .env created"
else
    echo "✓ .env exists"
fi

# 4. Build and start all services
echo ""
echo "Starting services..."
docker compose build --quiet
docker compose up -d

# 5. Wait for PostgreSQL
echo -n "Waiting for PostgreSQL..."
for i in $(seq 1 30); do
    if docker compose exec -T postgres pg_isready -U norest -d norest &>/dev/null; then
        echo " OK"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo " FAILED"
        echo "ERROR: PostgreSQL did not become ready in time."
        docker compose logs postgres
        exit 1
    fi
    sleep 1
    echo -n "."
done

# 6. Run migrations
echo "Running migrations..."
for migration in migrations/*.sql; do
    echo "  Applying $(basename "$migration")..."
    docker compose exec -T postgres psql -U norest -d norest -f /dev/stdin < "$migration" 2>/dev/null || true
done
echo "✓ Migrations applied"

# 7. Wait for Stalwart HTTP
echo -n "Waiting for Stalwart..."
for i in $(seq 1 60); do
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/ 2>/dev/null | grep -qE "^[2345]"; then
        echo " OK"
        break
    fi
    if [ "$i" -eq 60 ]; then
        echo " WAITING (Stalwart may need initial setup at http://localhost:8081/admin)"
        break
    fi
    sleep 2
    echo -n "."
done

# Bootstrap Stalwart programmatically using the Admin Secret if it is in bootstrap mode
# DANGER: THIS IS STRICTLY DEVELOPMENT ONLY.
# In a production environment, bootstrap must be performed manually or via secure orchestrator.
APP_ENV=${APP_ENV:-development}
if [ "$APP_ENV" = "development" ]; then
    echo "Checking Stalwart Bootstrap State (DEVELOPMENT ONLY)..."
    curl -s -u admin:change-me-development-only -X POST http://localhost:8081/jmap -H "Content-Type: application/json" -d '{
      "using": ["urn:ietf:params:jmap:core", "urn:stalwart:jmap"],
      "methodCalls": [
        ["x:Bootstrap/set", {
          "accountId": "admin",
          "update": {
            "singleton": {
              "serverHostname": "localhost",
              "defaultDomain": "localhost"
            }
          }
        }, "0"]
      ]
    }' > /dev/null || true
    echo "Bootstrapped!"
    docker compose restart stalwart || true
else
    echo "WARNING: Non-development environment detected. Skipping automatic Stalwart bootstrap."
fi

# 8. Wait for Norest API
echo -n "Waiting for Norest API..."
for i in $(seq 1 30); do
    if curl -s http://localhost:8080/health 2>/dev/null | grep -q '"ok"'; then
        echo " OK"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo " FAILED"
        echo "ERROR: Norest API did not become ready in time."
        docker compose logs norest-api
        exit 1
    fi
    sleep 2
    echo -n "."
done

# 9. Print service URLs
echo ""
echo "============================================"
echo "  Norest Mail is running."
echo "============================================"
echo ""
echo "PostgreSQL:"
echo "  Internal: postgres:5432 (Docker network only)"
echo ""
echo "Stalwart Admin:"
echo "  http://localhost:8081/admin"
echo ""
echo "Stalwart JMAP:"
echo "  http://localhost:8081/jmap"
echo ""
echo "Norest API:"
echo "  http://localhost:8080"
echo ""
echo "Health:"
echo "  http://localhost:8080/health"
echo ""
