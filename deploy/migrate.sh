#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/deploy"

DB_USER="${DB_USER:-comic}"
DB_NAME="${DB_NAME:-comic}"
COMPOSE="docker compose -f docker-compose.prod.yml"

if ! $COMPOSE ps postgres >/dev/null 2>&1; then
  echo "postgres service is not running; start the stack first: $COMPOSE up -d postgres redis" >&2
  exit 1
fi

for f in "$ROOT"/server/migrations/*.up.sql; do
  echo "apply $(basename "$f")"
  $COMPOSE exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -q <"$f"
done

echo "migrations applied to $DB_NAME"
