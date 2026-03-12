# City Stories Guide

Ambient audio guide that tells stories about nearby places while you walk. A "parallel audio layer" over Google Maps — navigation stays in Maps, and City Stories Guide turns any walk into a living documentary.

## Monorepo Structure

```
cityStoriesGuide/
├── backend/       # Go API server (Gin, PostgreSQL + PostGIS, pgx)
│   ├── cmd/       # Entry points (api, worker)
│   ├── internal/  # Business logic (handler, service, repository, domain)
│   ├── migrations/# SQL migrations (golang-migrate)
│   └── scripts/   # Helper scripts
├── mobile/        # React Native (Expo) mobile app
│   ├── app/       # Expo Router file-based routes
│   └── src/       # App source (api, services, store, components, hooks)
├── admin/         # React admin panel (Vite + Ant Design)
│   └── src/       # Admin source (pages, components, api, hooks)
├── infra/         # Infrastructure configs
│   ├── docker-compose.yml
│   ├── Caddyfile
│   └── backup/
├── scripts/       # Repo-level helper/validation scripts
├── .github/       # GitHub Actions CI/CD workflows
├── tasks.json     # Structured task list
├── progress.md    # Agent progress log
└── PRD.md         # Product Requirements Document
```

## Tech Stack

| Layer          | Technology                                      |
|----------------|------------------------------------------------|
| Backend        | Go, Gin, PostgreSQL 16 + PostGIS 3.4, pgx     |
| Mobile         | React Native (Expo), Expo Router, Zustand      |
| Admin Panel    | React, TypeScript, Vite, Ant Design, Leaflet   |
| Infrastructure | Docker Compose, Caddy, GitHub Actions          |
| AI / TTS       | Claude API (story generation), ElevenLabs (audio) |
| Storage        | S3-compatible (MinIO dev, Backblaze B2 prod)   |

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.24+ | Backend |
| Node.js | 22+ | Admin, mobile, API type generation |
| Docker & Docker Compose | latest | Local Postgres, MinIO, Prometheus, Grafana |
| golang-migrate | v4.18+ | Database migrations (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`) |
| golangci-lint | v1.64+ | Backend linting (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8`) |

## Getting Started

### 0. Verify your environment

```bash
make doctor
```

This checks that required tools (`go`, `node`, `npm`, `docker`) are installed, `backend/.env` exists, dependency manifests parse correctly, and key project files are in place. Fix any failures before proceeding.

### 1. Start the full local stack (recommended)

```bash
make dev-up
```

This builds and starts the backend API along with all dependencies (PostgreSQL, MinIO, Prometheus, Grafana) and runs database migrations automatically. The API is available at `http://localhost:8080`.

| Service    | URL                        | Notes                          |
|------------|----------------------------|--------------------------------|
| API        | http://localhost:8080       | Health: `/healthz`, `/readyz`  |
| MinIO      | http://localhost:9001       | Console (minioadmin/minioadmin_secret) |
| Prometheus | http://localhost:9090       |                                |
| Worker     | —                          | Runs in mock provider mode     |
| Grafana    | http://localhost:3000       | admin/admin                    |

To stop the stack: `make dev-down`. To tail API logs: `make dev-logs`.

### 1b. Infrastructure only (run API outside Docker)

If you prefer to run the API binary directly (e.g. for faster iteration with a debugger):

```bash
cd infra && docker compose up -d postgres minio
```

Then start the backend manually:

```bash
test -f backend/.env || cp backend/.env.example backend/.env
cd backend
make migrate-up   # apply database migrations
make build        # compile bin/api and bin/worker
./bin/api         # start the API server (default :8080)
```

See [backend/README.md](./backend/README.md) for the full list of Make targets and environment variables.

### 2. Admin panel

```bash
cd admin
npm ci
npm run dev       # Vite dev server
```

### 3. Mobile app

```bash
cd mobile
npm ci
npx expo start    # or: npm start
```

#### Running on Android Emulator

**Prerequisites:**
- [Android Studio](https://developer.android.com/studio) installed with Android SDK
- An AVD (Android Virtual Device) created via Android Studio → Device Manager
- `ANDROID_HOME` environment variable set (usually `~/Android/Sdk` on Linux, `~/Library/Android/sdk` on macOS)

```bash
# 1. Start the Android emulator (or open it from Android Studio → Device Manager)
emulator -avd <your_avd_name>

# 2. Start the app on the emulator
cd mobile
npm ci
npx expo start --android
# or from the repo root:
make mobile-android
```

The Expo dev server builds the app and installs it on the running emulator. If no emulator is running, Expo will attempt to launch the default AVD automatically.

> **Tip:** The backend API runs on `localhost:8080` inside Docker. The Android emulator accesses the host machine at `10.0.2.2`, so set `API_URL=http://10.0.2.2:8080` in the mobile environment if needed.

#### Running on iOS Simulator (macOS only)

**Prerequisites:**
- macOS with [Xcode](https://developer.apple.com/xcode/) installed (latest stable recommended)
- Xcode Command Line Tools: `xcode-select --install`
- CocoaPods: `sudo gem install cocoapods` (or `brew install cocoapods`)
- An iOS Simulator available (Xcode → Settings → Platforms → download an iOS runtime if needed)

```bash
# 1. Install dependencies and CocoaPods
cd mobile
npm ci
npx expo run:ios
# or from the repo root:
make mobile-ios
```

`expo run:ios` creates a native build (first run takes several minutes) and launches the iOS Simulator. Subsequent runs are fast.

> **Tip:** To pick a specific simulator: `npx expo run:ios --device "iPhone 16 Pro"`

#### Running on a Physical Device

```bash
# Android: connect via USB with USB debugging enabled, then:
npx expo run:android --device

# iOS: connect via USB, then:
npx expo run:ios --device
```

For iOS physical devices you need an Apple Developer account and a valid provisioning profile configured in Xcode.

#### Mobile Tests

```bash
cd mobile
npm test                    # unit tests (Jest)
npm run lint                # ESLint
npm run typecheck           # TypeScript check
npm run format:check        # Prettier check

# or from the repo root:
make verify-mobile          # runs lint + typecheck + test
```

### 4. Demo data (optional but recommended)

Seed the database with a predictable local dataset so admin pages and nearby-story flows work without manual data entry:

```bash
# One command — starts Docker, runs migrations, seeds demo data:
make demo-setup
```

Or step by step if infrastructure is already running:

```bash
cd backend
make migrate-up    # apply migrations
make seed          # seed demo data (idempotent)
```

To wipe and reseed:

```bash
cd backend
make demo-reset    # truncate all tables (local dev only)
make seed          # repopulate
```

**Demo credentials:** `admin@demo.local` / `demodemo` (admin), `user@demo.local` / `demodemo` (regular user).

The dataset includes 1 city (Tbilisi), 55 POIs, 110 stories (EN+RU), 5 reports, and listening history. See [backend/README.md](./backend/README.md#demo-data) for details.

### 5. Validate the OpenAPI spec

Lint the spec for structural errors, invalid references, and schema issues:

```bash
make lint-api
```

This runs [@redocly/cli](https://redocly.com/docs/cli/) against `backend/api/openapi.yaml` using the `redocly.yaml` config at the repo root. CI will fail if the spec contains errors.

### 6. Generated API types

Both `admin` and `mobile` share TypeScript types generated from `backend/api/openapi.yaml`. After changing the OpenAPI spec, regenerate from the **repo root**:

```bash
make generate-api
```

To verify that generated schemas are up to date (exits non-zero on drift):

```bash
make check-generated-api
```

This compares `admin/src/api/generated/schema.ts` and `mobile/src/api/generated/schema.ts` against the current spec. Always run generation and commit the output after spec changes.

## Verification (mirrors CI)

### Quick: verify everything from the repo root

```bash
make verify          # runs backend, admin, mobile checks + API drift detection
```

Or target a single layer:

```bash
make verify-planning # validate tasks.json structure and task IDs/statuses
make verify-backend  # lint, test, build
make verify-admin    # lint, typecheck, build
make verify-mobile   # lint, typecheck, test
make verify-api      # regenerate API types and check for uncommitted drift
```

Planning artifacts live at the repo root: `tasks.json` is the structured source of truth for task state, `scripts/validate_tasks.py` enforces its schema, and `progress.md` records per-task progress entries against that file.

### Per-package commands

Run these from the listed working directory to match what GitHub Actions checks:

| Check | Directory | Command |
|-------|-----------|---------|
| Backend lint | `backend/` | `make lint` |
| Backend test | `backend/` | `make test` |
| Backend integration test | `backend/` | `make test-integration` |
| Backend build | `backend/` | `make build` |
| Migration roundtrip | `backend/` | `make test-migrations` |
| Admin lint | `admin/` | `npm run lint` |
| Admin typecheck | `admin/` | `npm run typecheck` |
| Admin format | `admin/` | `npm run format:check` |
| Admin build | `admin/` | `npm run build` |
| Admin E2E | `admin/` | `npm run test:e2e` |
| Mobile lint | `mobile/` | `npm run lint` |
| Mobile typecheck | `mobile/` | `npm run typecheck` |
| Mobile format | `mobile/` | `npm run format:check` |
| Mobile test | `mobile/` | `npm test` |
| OpenAPI spec lint | repo root | `make lint-api` |
| API types up-to-date | repo root | `make check-generated-api` |

## Troubleshooting

**First step** — run `make doctor` from the repo root. It checks for missing tools, configuration files, and project structure issues with actionable fix instructions.

**Database connection refused** — `backend/.env.example` defaults to port `5433`, but `infra/docker-compose.yml` exposes Postgres on port `5432`. Make sure the port in your `DATABASE_URL` matches the running container.

**Generated API types out of date** — If CI fails on the `api-types` job, run `make generate-api` from the repo root and commit the changes. Both `admin/` and `mobile/` must have `npm ci` run first so `openapi-typescript` is available.

**`golangci-lint` not found** — Install it with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8`. The binary must be on your `PATH` (usually `$(go env GOPATH)/bin`).

**`migrate` not found** — Install with `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`.

**MinIO bucket not created** — The `city-stories` bucket must exist before the backend can upload files. Create it via the MinIO console at `http://localhost:9001` or with `mc mb local/city-stories`.

**Admin E2E tests fail locally** — Install Playwright browsers first: `cd admin && npx playwright install --with-deps chromium`.

## License

Private — All rights reserved.
