#!/usr/bin/env bash
set -euo pipefail

# doctor.sh — Validate that a developer machine is ready to work on the monorepo.
# Usage: ./scripts/doctor.sh   (or: make doctor)
#
# Read-only: does not start containers, install packages, or mutate the tree.

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Colors for output.
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

pass()  { echo -e "  ${GREEN}✔${NC} $*"; }
fail()  { echo -e "  ${RED}✘${NC} $*"; FAILURES=$((FAILURES + 1)); }
warn()  { echo -e "  ${YELLOW}!${NC} $*"; }

FAILURES=0

echo -e "${BOLD}City Stories Guide — Doctor${NC}"
echo ""

# ── Required CLI tools ───────────────────────────────────────────────────────
echo -e "${BOLD}Required tools${NC}"

check_tool() {
    local cmd="$1"
    local install_hint="$2"
    if command -v "$cmd" > /dev/null 2>&1; then
        local version
        version=$("$cmd" --version 2>&1 | head -1 || true)
        pass "$cmd  ($version)"
    else
        fail "$cmd not found — $install_hint"
    fi
}

# Go uses 'go version' instead of 'go --version'.
if command -v go > /dev/null 2>&1; then
    pass "go  ($(go version 2>&1 || true))"
else
    fail "go not found — install from https://go.dev/dl/"
fi

check_tool "node"   "install Node.js 22+ from https://nodejs.org/"
check_tool "npm"    "installed with Node.js"
check_tool "docker" "install Docker Desktop or Docker Engine"

# Docker daemon running
if docker info > /dev/null 2>&1; then
    pass "Docker daemon is running"
else
    fail "Docker daemon is not running — start Docker and try again"
fi

echo ""

# ── Optional but recommended tools ──────────────────────────────────────────
echo -e "${BOLD}Optional tools (used by make targets)${NC}"

if command -v golangci-lint > /dev/null 2>&1; then
    pass "golangci-lint  ($(golangci-lint --version 2>&1 | head -1 || true))"
else
    warn "golangci-lint not found — needed for 'make lint'; install with:"
    warn "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
fi

if command -v migrate > /dev/null 2>&1; then
    pass "migrate  ($(migrate --version 2>&1 | head -1 || true))"
else
    warn "migrate not found — needed for 'make migrate-up'; install with:"
    warn "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
fi

echo ""

# ── Configuration files ─────────────────────────────────────────────────────
echo -e "${BOLD}Configuration${NC}"

if [ -f "$ROOT_DIR/backend/.env" ]; then
    pass "backend/.env exists"
else
    fail "backend/.env is missing — create it with: cp backend/.env.example backend/.env"
fi

if [ -f "$ROOT_DIR/backend/.env.example" ]; then
    pass "backend/.env.example exists"
else
    fail "backend/.env.example is missing — this file should be in the repo"
fi

echo ""

# ── Dependency manifests ────────────────────────────────────────────────────
echo -e "${BOLD}Dependency manifests${NC}"

# Backend: go.mod must parse
if [ -f "$ROOT_DIR/backend/go.mod" ]; then
    if (cd "$ROOT_DIR/backend" && go mod verify > /dev/null 2>&1); then
        pass "backend/go.mod is valid"
    else
        # go mod verify may fail if deps aren't downloaded yet; just check parse
        if grep -q '^module ' "$ROOT_DIR/backend/go.mod"; then
            pass "backend/go.mod is parseable (run 'go mod download' to fetch deps)"
        else
            fail "backend/go.mod cannot be parsed"
        fi
    fi
else
    fail "backend/go.mod is missing"
fi

# Admin: package.json must be valid JSON
if [ -f "$ROOT_DIR/admin/package.json" ]; then
    if node -e "JSON.parse(require('fs').readFileSync('$ROOT_DIR/admin/package.json','utf8'))" 2>/dev/null; then
        pass "admin/package.json is valid JSON"
    else
        fail "admin/package.json is not valid JSON"
    fi
else
    fail "admin/package.json is missing"
fi

# Mobile: package.json must be valid JSON
if [ -f "$ROOT_DIR/mobile/package.json" ]; then
    if node -e "JSON.parse(require('fs').readFileSync('$ROOT_DIR/mobile/package.json','utf8'))" 2>/dev/null; then
        pass "mobile/package.json is valid JSON"
    else
        fail "mobile/package.json is not valid JSON"
    fi
else
    fail "mobile/package.json is missing"
fi

echo ""

# ── Key project files ───────────────────────────────────────────────────────
echo -e "${BOLD}Project structure${NC}"

# Migration directory
if [ -d "$ROOT_DIR/backend/migrations" ]; then
    migration_count=$(find "$ROOT_DIR/backend/migrations" -name '*.sql' | wc -l)
    if [ "$migration_count" -gt 0 ]; then
        pass "backend/migrations/ contains $migration_count SQL files"
    else
        fail "backend/migrations/ exists but contains no .sql files"
    fi
else
    fail "backend/migrations/ directory is missing"
fi

# Generate-clients script
if [ -f "$ROOT_DIR/backend/scripts/generate-clients.sh" ]; then
    if [ -x "$ROOT_DIR/backend/scripts/generate-clients.sh" ]; then
        pass "backend/scripts/generate-clients.sh is present and executable"
    else
        fail "backend/scripts/generate-clients.sh exists but is not executable — run: chmod +x backend/scripts/generate-clients.sh"
    fi
else
    fail "backend/scripts/generate-clients.sh is missing — needed by 'make generate-api'"
fi

# OpenAPI spec
if [ -f "$ROOT_DIR/backend/api/openapi.yaml" ]; then
    pass "backend/api/openapi.yaml exists"
else
    fail "backend/api/openapi.yaml is missing"
fi

echo ""

# ── Summary ─────────────────────────────────────────────────────────────────
if [ "$FAILURES" -eq 0 ]; then
    echo -e "${GREEN}${BOLD}All checks passed!${NC} Your machine is ready to work on the monorepo."
    exit 0
else
    echo -e "${RED}${BOLD}$FAILURES check(s) failed.${NC} Fix the issues above and re-run: make doctor"
    exit 1
fi
