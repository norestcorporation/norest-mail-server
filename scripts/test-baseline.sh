#!/bin/bash
set -e

echo "=========================================="
echo "  Norest Mail Chapter 4 Baseline Test"
echo "=========================================="
echo ""

echo "[1] Running Go tests..."
go test ./...
echo "✓ Go tests passed"
echo ""

echo "[2] Running Go build..."
go build ./...
echo "✓ Go build passed"
echo ""

echo "[3] Running foundation test..."
./scripts/test-foundation.sh
echo "✓ Foundation test passed"
echo ""

echo "[4] Running DB clean verification..."
./scripts/verify-db-clean.sh
echo "✓ DB clean verification passed"
echo ""

echo "[5] Running Chapter 2 test..."
./scripts/test-chapter2.sh
echo "✓ Chapter 2 test passed"
echo ""

echo "[6] Running Chapter 2 verification..."
go run scripts/verify-chapter2/main.go
echo "✓ Chapter 2 verification passed"
echo ""

echo "[7] Running Chapter 3 verification..."
go run scripts/verify-chapter3/main.go
echo "✓ Chapter 3 verification passed"
echo ""

echo "[8] Running Chapter 4 full test..."
./scripts/test-chapter4-full.sh
echo "✓ Chapter 4 full test passed"
echo ""

echo "=========================================="
echo "  BASELINE = PASS"
echo "=========================================="
