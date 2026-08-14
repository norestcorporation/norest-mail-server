#!/bin/bash
set -e

echo "Verifying PostgreSQL has no mail data..."
TABLES=$(docker exec server-postgres-1 psql -U norest -d norest -t -c "\dt")
echo "$TABLES"
echo "Checking for any table containing 'message', 'email', 'folder', 'attachment'..."
if echo "$TABLES" | grep -iE 'message|email|folder|attachment'; then
  echo "FAILED: Found mail data tables!"
  exit 1
fi
echo "SUCCESS: No mail data tables found in PostgreSQL."
