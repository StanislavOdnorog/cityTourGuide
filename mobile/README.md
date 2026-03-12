# Mobile

React Native (Expo) mobile app for City Stories Guide.

## Local Setup

1. Install dependencies:

   ```bash
   npm ci
   ```

2. Start the Expo dev server:

   ```bash
   npx expo start
   ```

   Or use the shorthand scripts:

   ```bash
   npm run ios       # start on iOS simulator
   npm run android   # start on Android emulator
   npm run web       # start web version
   ```

## npm Scripts

All commands run from the `mobile/` directory.

| Script | Description |
|--------|-------------|
| `npm start` | Start Expo dev server (`expo start`) |
| `npm run ios` | Start on iOS simulator |
| `npm run android` | Start on Android emulator |
| `npm run web` | Start web version |
| `npm test` | Run Jest tests |
| `npm run lint` | Run ESLint |
| `npm run lint:fix` | Run ESLint with auto-fix |
| `npm run typecheck` | Run TypeScript type checking (`tsc --noEmit`) |
| `npm run format` | Format source files with Prettier |
| `npm run format:check` | Check formatting without writing |
| `npm run generate-api` | Regenerate TypeScript types from the OpenAPI spec |

## CI Checks

The Mobile CI workflow (`.github/workflows/mobile.yml`) runs on changes to `mobile/**`:

- **Lint** — `npm run lint`
- **Typecheck** — `npm run typecheck`
- **Format** — `npm run format:check`
- **Test** — `npm test`

## Generated API Types

TypeScript types are generated from `backend/api/openapi.yaml` into `src/api/generated/schema.ts`.

To regenerate after an OpenAPI spec change, run from the **repo root**:

```bash
make generate-api
```

Or from the `mobile/` directory:

```bash
npm run generate-api
```

CI will fail if the generated types are out of date. Always commit the regenerated output.
