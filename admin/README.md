# Admin Panel

React admin panel for City Stories Guide, built with Vite, Ant Design, TanStack Query, and Leaflet.

## Local Setup

1. Install dependencies:

   ```bash
   npm ci
   ```

2. Start the Vite dev server:

   ```bash
   npm run dev
   ```

   The backend API must be running for the admin panel to function. See [backend/README.md](../backend/README.md) for setup instructions.

## npm Scripts

All commands run from the `admin/` directory.

| Script | Description |
|--------|-------------|
| `npm run dev` | Start Vite dev server with HMR |
| `npm run build` | TypeScript compile and Vite production build |
| `npm run preview` | Preview the production build locally |
| `npm run lint` | Run ESLint |
| `npm run lint:fix` | Run ESLint with auto-fix |
| `npm run typecheck` | Run TypeScript type checking (`tsc -b`) |
| `npm run format` | Format source files with Prettier |
| `npm run format:check` | Check formatting without writing |
| `npm test` | Run Vitest unit tests |
| `npm run test:e2e` | Run Playwright E2E tests |
| `npm run test:e2e:ui` | Run Playwright E2E tests with interactive UI |
| `npm run generate-api` | Regenerate TypeScript types from the OpenAPI spec |

## CI Checks

The Admin CI workflow (`.github/workflows/admin.yml`) runs on changes to `admin/**`:

- **Lint** — `npm run lint`
- **Typecheck** — `npm run typecheck`
- **Format** — `npm run format:check`
- **Build** — `npm run build`
- **E2E** — `npm run test:e2e` (requires Playwright browsers: `npx playwright install --with-deps chromium`)

## Generated API Types

TypeScript types are generated from `backend/api/openapi.yaml` into `src/api/generated/schema.ts`.

To regenerate after an OpenAPI spec change, run from the **repo root**:

```bash
make generate-api
```

Or from the `admin/` directory:

```bash
npm run generate-api
```

CI will fail if the generated types are out of date. Always commit the regenerated output.

## Authentication

The admin panel uses JWT authentication. The backend must be configured with a `JWT_SECRET` in `backend/.env`. Admin users are created via the backend seed command or API.
