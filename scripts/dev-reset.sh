#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

if [ "${1:-}" = "--force" ] || [ "${1:-}" = "-f" ]; then
    echo "Force reset requested."
else
    echo "This deletes development database and Stalwart data."
    echo -n "Type RESET to continue: "
    read -r confirm
    if [ "$confirm" != "RESET" ]; then
        echo "Aborted."
        exit 0
    fi
fi

echo "Destroying development environment..."
docker compose down -v
echo "Done. All volumes removed."
echo ""
echo "Run ./scripts/dev-up.sh to start fresh."
