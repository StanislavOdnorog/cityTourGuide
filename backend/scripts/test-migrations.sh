#!/usr/bin/env bash
set -euo pipefail

# Test migration roundtrip: up → down → up
# Requires: migrate CLI, DATABASE_URL env var

MIGRATIONS_DIR="${MIGRATIONS_DIR:-migrations}"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL must be set}"

echo "==> Running all migrations UP..."
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
echo "    ✓ All migrations applied"

echo "==> Running all migrations DOWN..."
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down -all
echo "    ✓ All migrations reverted"

echo "==> Running all migrations UP again..."
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
echo "    ✓ All migrations re-applied"

echo "==> Migration roundtrip passed"
