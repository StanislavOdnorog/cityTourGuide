#!/usr/bin/env bash
# Validate the OpenAPI spec using Redocly CLI.
# Usage: bash scripts/lint-openapi.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
SPEC="$ROOT_DIR/backend/api/openapi.yaml"

if ! command -v npx >/dev/null 2>&1; then
  echo "ERROR: npx is not installed. Install Node.js 22+ first." >&2
  exit 1
fi

echo "==> Linting OpenAPI spec: $SPEC"
npx --yes @redocly/cli@latest lint "$SPEC" --config "$ROOT_DIR/redocly.yaml"
echo "OpenAPI spec is valid."
