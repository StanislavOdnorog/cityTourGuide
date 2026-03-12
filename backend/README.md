# Backend

Go API server and background worker for City Stories Guide.

## Local Setup

> **Tip:** Run `make doctor` from the repo root first to verify all required tools are installed and configuration files are in place.

1. Start PostgreSQL and MinIO via Docker Compose (from the repo root):

   ```bash
   cd infra && docker compose up -d
   ```

2. Create a local env file and edit as needed:

   ```bash
   test -f .env || cp .env.example .env
   ```

   > **Note:** `.env.example` defaults `DATABASE_URL` to port `5433`. If you use `infra/docker-compose.yml` without overriding `POSTGRES_PORT`, the container listens on port `5432` — update the port in your `.env` accordingly.

3. Install Go tooling (if not already present):

   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
   go install golang.org/x/tools/cmd/goimports@latest
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

4. Apply database migrations and build:

   ```bash
   make migrate-up
   make build
   ```

5. Run the API server or worker:

   ```bash
   ./bin/api
   ./bin/worker
   ```

## Environment Variables

Copy `backend/.env.example` to `.env` for local development. Duration values accept Go duration strings (`30s`, `5m`, `1h`) or plain integer seconds (`30`).

A parity test (`TestEnvExampleParity`) ensures `.env.example` stays in sync with `config.go` — it will fail CI if a new env var is added in code but not documented.

### Provider Mode

| Variable | Default | Description |
|----------|---------|-------------|
| `PROVIDER_MODE` | `real` | `real` or `mock` — controls external integration clients |

Setting `PROVIDER_MODE=mock` replaces the Claude, ElevenLabs, and S3 clients with deterministic in-process fakes. This lets contributors run the full stack (including the worker) without API credentials. Mock mode is logged at startup and returns placeholder story text, silent MP3 audio, and in-memory object storage. **Never use mock mode in production** — real mode fails fast when required credentials are missing.

### Required

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `JWT_SECRET` | — | Secret for signing JWT tokens |

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `GIN_MODE` | `debug` | Gin mode (`debug`, `release`, `test`) |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated CORS origins (scheme + host required) |
| `SERVER_READ_TIMEOUT` | `10s` | HTTP read timeout (duration) |
| `SERVER_WRITE_TIMEOUT` | `30s` | HTTP write timeout (duration) |
| `SERVER_IDLE_TIMEOUT` | `120s` | HTTP idle timeout (duration) |
| `SERVER_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout (duration) |

### Database Pool

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_MAX_CONNS` | `25` | Maximum open connections |
| `DB_MIN_CONNS` | `2` | Minimum idle connections |
| `DB_MAX_CONN_LIFETIME` | `1h` | Max connection lifetime (duration) |
| `DB_MAX_CONN_IDLE_TIME` | `5m` | Max idle time before close (duration) |
| `DB_HEALTH_CHECK_PERIOD` | `30s` | Connection health check interval (duration) |

### JWT

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime (duration) |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (duration) |

### OAuth (optional groups)

Google Sign-In requires only `GOOGLE_CLIENT_ID`. Apple Sign-In requires all four Apple variables together. If any variable in a group is set, all must be set.

| Variable | Description |
|----------|-------------|
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `APPLE_CLIENT_ID` | App bundle ID |
| `APPLE_TEAM_ID` | Apple developer team ID |
| `APPLE_KEY_ID` | Sign-In with Apple key ID |
| `APPLE_PRIVATE_KEY` | PEM-encoded ECDSA private key |

### S3 Storage (required for worker, optional group for API)

All four must be set together. Required when running the worker; optional for the API.
Admin story updates/deletes and city or POI deletes enqueue orphaned audio cleanup jobs in Postgres. `bin/worker` processes those jobs asynchronously and retries transient storage deletion failures; already-missing objects are treated as successfully cleaned up.

| Variable | Default | Description |
|----------|---------|-------------|
| `S3_ENDPOINT` | — | S3-compatible endpoint (`http://localhost:9000` for MinIO) |
| `S3_ACCESS_KEY` | — | S3 access key |
| `S3_SECRET_KEY` | — | S3 secret key |
| `S3_BUCKET` | `city-stories` | S3 bucket name |

### External Services (optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAUDE_API_KEY` | — | Anthropic API key (story generation) |
| `CLAUDE_HTTP_TIMEOUT` | `60s` | Claude API HTTP timeout (duration) |
| `ELEVENLABS_API_KEY` | — | ElevenLabs API key (TTS) |
| `ELEVENLABS_HTTP_TIMEOUT` | `120s` | ElevenLabs API HTTP timeout (duration) |
| `FCM_CREDENTIALS_JSON` | — | Firebase service account JSON (single-line) |
| `FCM_HTTP_TIMEOUT` | `30s` | FCM API HTTP timeout (duration) |

### Logging (not part of config.go, consumed by logger package)

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Make Targets

All commands run from the `backend/` directory.

| Target | Description |
|--------|-------------|
| `make build` | Compile `bin/api` and `bin/worker` |
| `make test` | Run unit tests with race detector and coverage |
| `make test-integration` | Run integration tests (requires a running test database) |
| `make test-migrations` | Run migration roundtrip (up → down → up) |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code with `gofmt` and `goimports` |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Roll back the last migration |
| `make migrate-create` | Create a new migration pair (prompts for a name) |
| `make seed` | Seed the database with demo data (idempotent) |
| `make demo-reset` | Truncate all tables (local dev only) |
| `make generate-api` | Regenerate TypeScript API clients for admin and mobile |
| `make clean` | Remove compiled binaries |

## Database

The project uses **PostgreSQL 16 with PostGIS 3.4**. The Docker image is `postgis/postgis:16-3.4`.

Default credentials (local dev):

- User: `citystories`
- Password: `citystories_secret`
- Database: `citystories`

Integration tests use a separate database (`citystories_test` by default, controlled by `TEST_DATABASE_URL`).

## Migrations

Managed by [golang-migrate](https://github.com/golang-migrate/migrate). Migration files live in `backend/migrations/`.

```bash
make migrate-up      # apply all pending
make migrate-down    # roll back one step
make migrate-create  # create a new pair (interactive)
make test-migrations # verify up → down → up roundtrip
```

## Demo Data

The seed command (`make seed`) populates the database with a reproducible local dataset:

| Data | Count | Details |
|------|-------|---------|
| Cities | 1 | Tbilisi, Georgia |
| POIs | 55 | Real landmarks with accurate coordinates |
| Stories | 110 | 55 EN + 55 RU, with narrative text and fake audio URLs |
| Users | 2 | Admin + regular user |
| Reports | 5 | Mixed statuses (new, reviewed, resolved) |
| Listenings | 8 | Demo user history |

**Demo credentials:**

| Role | Email | Password |
|------|-------|----------|
| Admin | `admin@demo.local` | `demodemo` |
| User | `user@demo.local` | `demodemo` |

The seed is idempotent — running it again skips existing records. To start fresh:

```bash
make demo-reset   # truncate all tables
make seed         # repopulate
```

Both commands refuse to run if `DATABASE_URL` contains `prod`, `production`, or cloud provider hostnames.

**Verifying the seed worked:**

```bash
# Check POI count
curl -s http://localhost:8080/api/v1/cities | jq '.data | length'
# Should return 1

# Check nearby stories (Tbilisi center coordinates)
curl -s 'http://localhost:8080/api/v1/nearby?lat=41.7151&lng=44.8271&radius=2000' | jq '.data | length'
# Should return multiple stories
```

## Generated API Types

TypeScript types for the admin and mobile apps are generated from `backend/api/openapi.yaml`. After modifying the spec, run from the **repo root**:

```bash
make generate-api
```

This runs `openapi-typescript` inside both `admin/` and `mobile/`. They must have dependencies installed (`npm ci`) first. CI will reject PRs where generated types are stale.
