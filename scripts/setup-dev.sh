#!/usr/bin/env bash
set -euo pipefail

# setup-dev.sh — One-command dev environment setup for City Stories Guide.
# Usage: ./scripts/setup-dev.sh

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INFRA_DIR="$ROOT_DIR/infra"
BACKEND_DIR="$ROOT_DIR/backend"

# Detect docker compose command (v2 plugin vs v1 standalone).
if docker compose version > /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Colors for output.
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ── Step 1: Start Docker Compose services ───────────────────────────────────
info "Starting Docker Compose services (PostgreSQL + MinIO)..."
$DOCKER_COMPOSE -f "$INFRA_DIR/docker-compose.yml" --env-file "$INFRA_DIR/.env" up -d

# ── Step 2: Wait for PostgreSQL to be ready ─────────────────────────────────
info "Waiting for PostgreSQL to be ready..."
MAX_RETRIES=30
RETRY=0
until docker exec csg-postgres pg_isready -U citystories -d citystories > /dev/null 2>&1; do
    RETRY=$((RETRY + 1))
    if [ "$RETRY" -ge "$MAX_RETRIES" ]; then
        error "PostgreSQL did not become ready in time."
        exit 1
    fi
    sleep 1
done
info "PostgreSQL is ready."

# ── Step 3: Copy .env.example → .env if needed ─────────────────────────────
if [ ! -f "$BACKEND_DIR/.env" ]; then
    warn "No backend/.env found — copying from .env.example"
    cp "$BACKEND_DIR/.env.example" "$BACKEND_DIR/.env"
    info "Created backend/.env (review and update API keys as needed)"
fi

# ── Step 4: Run database migrations ────────────────────────────────────────
info "Running database migrations..."
cd "$BACKEND_DIR"
make migrate-up
info "Migrations applied."

# ── Step 5: Run seed data ──────────────────────────────────────────────────
info "Seeding database with test data..."
cd "$BACKEND_DIR"
go run ./scripts/seed/
info "Seed data inserted."

# ── Step 6: Summary ───────────────────────────────────────────────────────
echo ""
info "========================================="
info "  Dev environment is ready!"
info "========================================="
echo ""
echo "  PostgreSQL: localhost:$(grep POSTGRES_PORT "$INFRA_DIR/.env" 2>/dev/null | cut -d= -f2 || echo 5432)"
echo "  MinIO API:  localhost:$(grep MINIO_API_PORT "$INFRA_DIR/.env" 2>/dev/null | cut -d= -f2 || echo 9000)"
echo "  MinIO UI:   localhost:$(grep MINIO_CONSOLE_PORT "$INFRA_DIR/.env" 2>/dev/null | cut -d= -f2 || echo 9001)"
echo ""
echo "  Start backend: cd backend && go run ./cmd/api/"
echo "  Health check:  curl http://localhost:8080/healthz"
echo "  Nearby stories: curl 'http://localhost:8080/api/v1/nearby-stories?lat=41.7151&lng=44.8271&language=en'"
echo ""
