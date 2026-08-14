#!/bin/bash
set -e

echo "=========================================="
echo "  Production Configuration Check"
echo "=========================================="

# Check if running in production environment
if [ "$APP_ENV" != "production" ]; then
    echo "⚠ Not running in production environment (APP_ENV=$APP_ENV)"
    echo "This check is designed for production environments"
    exit 0
fi

echo "[1] Checking production environment..."
if [ "$APP_ENV" = "production" ]; then
    echo "✓ Production environment set"
else
    echo "✗ APP_ENV not set to production"
    exit 1
fi

echo "[2] Checking required secrets..."
secrets_ok=true
if [ -z "$JWT_SECRET" ] || [ "$JWT_SECRET" = "change-me-development-only" ]; then
    echo "✗ JWT_SECRET not set or using development default"
    secrets_ok=false
fi
if [ -z "$STALWART_ADMIN_PASSWORD" ] || [ "$STALwart_ADMIN_PASSWORD" = "change-me-development-only" ]; then
    echo "✗ STALWART_ADMIN_PASSWORD not set or using development default"
    secrets_ok=false
fi
if [ -z "$DATABASE_URL" ] || [[ "$DATABASE_URL" == "postgres://norest:norest@"* ]]; then
    echo "✗ DATABASE_URL not set or using development default"
    secrets_ok=false
fi
if [ "$secrets_ok" = true ]; then
    echo "✓ Required secrets configured"
else
    exit 1
fi

echo "[3] Checking CORS configuration..."
if [ -z "$ALLOWED_ORIGINS" ]; then
    echo "✗ ALLOWED_ORIGINS not set in production"
    exit 1
fi
if [[ "$ALLOWED_ORIGINS" == *"*"* ]]; then
    echo "✗ Wildcard CORS not allowed in production"
    exit 1
fi
echo "✓ CORS configured with explicit origins"

echo "[4] Checking Stalwart bootstrap..."
if [[ "$STALWART_RECOVERY_ADMIN" == *"change-me-development-only"* ]]; then
    echo "✗ Stalwart using development bootstrap credentials"
    exit 1
fi
echo "✓ Stalwart bootstrap configured"

echo "[5] Checking database configuration..."
if [[ "$DATABASE_URL" != *"sslmode=require"* ]] && [[ "$DATABASE_URL" != *"sslmode=verify-full"* ]]; then
    echo "⚠ Database URL does not require SSL (consider adding sslmode=require)"
fi
echo "✓ Database URL configured"

echo "[6] Checking required configuration variables..."
required_vars=("DATABASE_URL" "STALWART_BASE_URL" "STALWART_ADMIN_USER" "STALWART_ADMIN_PASSWORD" "JWT_SECRET")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "✗ Required variable $var not set"
        exit 1
    fi
done
echo "✓ All required configuration variables set"

echo "=========================================="
echo "  Production Configuration Check Passed"
echo "=========================================="