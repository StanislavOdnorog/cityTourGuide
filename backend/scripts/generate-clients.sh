#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SPEC="$ROOT_DIR/backend/api/openapi.yaml"

if [ ! -f "$SPEC" ]; then
  echo "ERROR: OpenAPI spec not found at $SPEC" >&2
  exit 1
fi

echo "Generating TypeScript API types from $SPEC..."

# Generate for mobile
npm --prefix "$ROOT_DIR/mobile" exec openapi-typescript -- \
  "$SPEC" \
  -o "$ROOT_DIR/mobile/src/api/generated/schema.ts"
echo "  -> mobile/src/api/generated/schema.ts"

# Generate for admin
npm --prefix "$ROOT_DIR/admin" exec openapi-typescript -- \
  "$SPEC" \
  -o "$ROOT_DIR/admin/src/api/generated/schema.ts"
echo "  -> admin/src/api/generated/schema.ts"

echo "Done."
