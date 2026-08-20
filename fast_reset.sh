cd "/home/ripun/Norest Mail/server"

set -e

echo "=================================================="
echo " NOREST MAIL — PART 1: RESET + DATABASE + STALWART"
echo "=================================================="

echo ""
echo "=== 1. STOP AND REMOVE EXISTING STACK ==="
docker compose -f docker-compose.multi-worker.yml down -v --remove-orphans --rmi local

echo ""
echo "=== 2. VERIFY NOREST CONTAINERS ARE RUNNING ==="
docker compose -f docker-compose.multi-worker.yml ps

echo ""
echo "=== 3. VERIFY PROJECT VOLUMES ARE REMOVED ==="
docker volume ls

echo ""
echo "=== 4. REBUILD ALL NOREST IMAGES FROM SCRATCH ==="
docker compose -f docker-compose.multi-worker.yml build 

echo ""
echo "=== 5. START POSTGRES + STALWART ==="
docker compose -f docker-compose.multi-worker.yml up -d postgres stalwart

echo ""
echo "=== 6. WAIT FOR POSTGRESQL ==="

until docker compose -f docker-compose.multi-worker.yml exec -T postgres \
    pg_isready -U norest -d norest >/dev/null 2>&1
do
    echo "Waiting for PostgreSQL..."
    sleep 2
done

echo "PostgreSQL is READY."

echo ""
echo "=== 7. VERIFY POSTGRES DATABASE ==="

docker compose -f docker-compose.multi-worker.yml exec -T postgres \
    psql -U norest -d norest \
    -c "SELECT current_database(), current_user;"

echo ""
echo "=== 8. WAIT FOR STALWART ==="

until curl -fsS http://localhost:8081/ >/dev/null 2>&1
do
    echo "Waiting for Stalwart..."
    sleep 2
done

echo "Stalwart is READY."

echo ""
echo "=== 9. APPLY DATABASE MIGRATIONS ==="

for migration in migrations/*.sql; do
    echo ""
    echo "----------------------------------------------"
    echo "Applying $(basename "$migration")"
    echo "----------------------------------------------"

    docker compose -f docker-compose.multi-worker.yml exec -T postgres \
        psql \
        -v ON_ERROR_STOP=1 \
        -U norest \
        -d norest \
        -f /dev/stdin < "$migration"

    echo "SUCCESS: $(basename "$migration")"
done

echo ""
echo "=================================================="
echo " ALL MIGRATIONS APPLIED SUCCESSFULLY"
echo "=================================================="

echo ""
echo "=== 10. VERIFY DATABASE TABLES ==="

docker compose -f docker-compose.multi-worker.yml exec -T postgres \
    psql -U norest -d norest \
    -c "\dt"

echo ""
echo "=== 11. BOOTSTRAP FRESH STALWART INSTANCE ==="

curl -fsS \
  -u admin:change-me-development-only \
  -X POST http://localhost:8081/jmap \
  -H "Content-Type: application/json" \
  -d '{
    "using": [
      "urn:ietf:params:jmap:core",
      "urn:stalwart:jmap"
    ],
    "methodCalls": [
      [
        "x:Bootstrap/set",
        {
          "accountId": "admin",
          "update": {
            "singleton": {
              "serverHostname": "localhost",
              "defaultDomain": "localhost"
            }
          }
        },
        "0"
      ] 
    ]
  }'

echo ""
echo "Stalwart bootstrap completed."

echo ""
echo "=== 12. RESTART STALWART AFTER BOOTSTRAP ==="

docker compose -f docker-compose.multi-worker.yml restart stalwart

echo ""
echo "=== 13. WAIT FOR STALWART AFTER RESTART ==="

until curl -fsS http://localhost:8081/ >/dev/null 2>&1
do
    echo "Waiting for Stalwart to restart..."
    sleep 2
done

echo "Stalwart is READY after restart."

echo ""
echo "=================================================="
echo " PART 1 COMPLETE"
echo " PostgreSQL + migrations + Stalwart are READY"
echo "=================================================="















cd "/home/ripun/Norest Mail/server"

set -e

echo "=================================================="
echo " NOREST MAIL — PART 2: START API + WORKERS"
echo "=================================================="

echo ""
echo "=== 1. START NOREST API + ALL WORKERS ==="

docker compose -f docker-compose.multi-worker.yml up -d \
    norest-api \
    norest-worker-1 \
    norest-worker-2 \
    norest-worker-3

echo ""
echo "=== 2. SHOW FINAL STACK ==="

docker compose -f docker-compose.multi-worker.yml ps

echo ""
echo "=== 3. VERIFY NOREST API ==="

curl -fsS http://localhost:8080/health

echo ""
echo ""
echo "=== 4. VERIFY POSTGRES ==="

docker compose -f docker-compose.multi-worker.yml exec -T postgres \
    pg_isready -U norest -d norest

echo ""
echo "=== 5. VERIFY STALWART ==="

curl -fsS http://localhost:8081/ >/dev/null

echo "Stalwart is READY."

echo ""
echo "=================================================="
echo " NOREST MAIL RESET COMPLETE"
echo "=================================================="