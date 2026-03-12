#!/usr/bin/env bash
# Ensures that endpoint-specific API wrappers in admin/src/api and mobile/src/api
# use the generated OpenAPI client (generatedApiClient) rather than raw axios calls.
#
# Allowed exceptions:
#   - client.ts files (auth refresh, interceptors, base client setup)
#   - retry.ts / errors.ts (shared plumbing)
#   - generated/ directory
#   - test files
set -euo pipefail

EXIT_CODE=0

# Pattern: direct HTTP method calls on the raw axios client (apiClient.get, apiClient.post, etc.)
# These indicate endpoint-specific wrappers bypassing the generated client.
PATTERN='apiClient\.(get|post|put|patch|delete|head|options|request)\b'

# Files to exclude: client plumbing, generated code, tests
EXCLUDE_PATTERN='(client\.ts|retry\.ts|errors\.ts|generated/|\.test\.|\.spec\.)'

for dir in admin/src/api mobile/src/api; do
  if [ ! -d "$dir" ]; then
    continue
  fi

  violations=$(grep -rEn "$PATTERN" "$dir" \
    | grep -Ev "$EXCLUDE_PATTERN" || true)

  if [ -n "$violations" ]; then
    echo "ERROR: Found raw apiClient HTTP calls in $dir (use generatedApiClient instead):"
    echo "$violations"
    echo ""
    EXIT_CODE=1
  fi
done

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "OK: All endpoint wrappers use generatedApiClient."
fi

exit $EXIT_CODE
