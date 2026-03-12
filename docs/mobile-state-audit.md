# Mobile Persisted Store Audit

Audit date: 2026-03-12

## Store Inventory

### 1. `useAuthStore` (persisted)

| Field | Persisted | User-scoped | Reset on logout |
|-------|-----------|-------------|-----------------|
| `user` | Yes | Yes | Yes (`clearSession`) |
| `userId` | Yes | Yes | Yes |
| `accessToken` | Yes | Yes | Yes |
| `refreshToken` | Yes | Yes | Yes |
| `bootstrapStatus` | No | No | No (transient) |
| `bootstrapError` | No | No | No (transient) |
| `_hasHydrated` | No | No | No (transient) |

- **Storage key**: `city-stories-auth`
- **Hydration**: Custom `merge` function with `sanitizePersistedAuthState` validates types on rehydrate. Drops corrupted payloads gracefully.
- **Reset**: `clearSession()` nulls all persisted fields. `resetAuthStore()` resets to full `DEFAULT_STATE`.
- **Risks**: None significant. Hydration sanitization is solid. `clearSession` does not reset `bootstrapStatus`/`bootstrapError`, which is correct since bootstrap re-runs after clear.

### 2. `useSettingsStore` (persisted)

| Field | Persisted | User-scoped | Reset on logout |
|-------|-----------|-------------|-----------------|
| `language` | Yes | No (device) | No |
| `onboardingCompleted` | Yes | No (device) | No |
| `deviceId` | Yes | No (device) | No |
| `geoNotifications` | Yes | No (device) | No |
| `contentNotifications` | Yes | No (device) | No |
| `pushToken` | Yes | **Partially** | Yes (via boundary reset) |
| `registeredPushUserId` | Yes | **Yes** | Yes (via boundary reset) |

- **Storage key**: `city-stories-settings`
- **Hydration**: Default merge (spread). No validation of persisted shape.
- **Reset**: `clearPushRegistration()` clears push fields. Called by `authBoundaryReset`.
- **Risk: No hydration validation** (MEDIUM) -- Unlike `useAuthStore`, there is no `merge` override to validate persisted data. A corrupted `deviceId` (e.g. `null` or non-string) would silently propagate into API calls. Similarly, `language` could be an invalid value if storage is tampered with.
- **Risk: `registeredPushUserId` survives app restart without session** (LOW) -- If the app crashes after push registration but before the auth boundary fires, the stale userId remains. In practice, reconcile on next bootstrap covers this.

### 3. `useDownloadStore` (persisted)

| Field | Persisted | User-scoped | Reset on logout |
|-------|-----------|-------------|-----------------|
| `downloadedCities` | Yes | **Yes** | Yes (via boundary reset) |
| `downloadsByCityId` | No (partialized out) | Yes | Yes (via boundary reset) |

- **Storage key**: `city-stories-downloads`
- **Hydration**: Default merge. `onRehydrateStorage` sets `_hasHydrated`.
- **Reset**: `clearAllDownloads()` called by `authBoundaryReset` on session clear.
- **Risk: Download metadata can diverge from disk** (MEDIUM) -- `downloadedCities` records persist even if the OS evicts cached audio files. Mitigated by `reconcileDownloadState()` in `_layout.tsx`, which checks at least one story file per city. However, if reconciliation fails (e.g. SQLite init error), stale records remain.
- **Risk: In-progress download state persists across crash** (LOW) -- `downloadsByCityId` is NOT persisted (correctly partialized out), so transient progress is lost on restart. This is correct behavior.

### 4. `usePurchaseStore` (persisted)

| Field | Persisted | User-scoped | Reset on logout |
|-------|-----------|-------------|-----------------|
| `status` | Yes | **Yes** | Yes (via boundary reset) |
| `isLoading` | No | No | No (transient) |
| `paywallVisible` | No | No | No (transient) |

- **Storage key**: `city-stories-purchases`
- **Hydration**: Default merge. No validation of persisted shape.
- **Reset**: `setState({ status: null })` called by `authBoundaryReset`.
- **Risk: Stale purchase status after token refresh with new user identity** (MEDIUM) -- The boundary reset only fires when `accessToken` goes from non-null to null. A token refresh that returns a *different* userId (edge case in device auth) would not trigger a purchase reset, leaving the old user's purchase status cached.
- **Risk: `free_stories_left` decrement is local-only** (LOW) -- `decrementFreeStories` reduces the counter client-side without server round-trip. If the app crashes mid-listen, the decrement is lost and the user gets a "free" retry. This is arguably user-friendly but means the persisted count can drift from server truth.
- **Risk: No shape validation on hydration** (LOW) -- If the `PurchaseStatus` type evolves (new required fields), stale persisted data could have missing fields. `hasFullAccess`/`hasCityAccess`/`canListenFree` use optional chaining (`??`) so this is partially mitigated.

### 5. `useCacheStore` (NOT persisted)

| Field | Persisted | User-scoped | Reset on logout |
|-------|-----------|-------------|-----------------|
| `stats` | No | No | N/A |
| `isClearing` | No | No | N/A |
| `initialized` | No | No | N/A |
| `error` | No | No | N/A |

- **No persistence risk.** This is a runtime-only store populated by `useCacheManager`. Correctly not persisted.

### 6. Non-persisted stores (no hydration concerns)

| Store | Persisted | Notes |
|-------|-----------|-------|
| `usePlayerStore` | No | Runtime playback state. Has `reset()`. Not user-scoped. |
| `useWalkStore` | No | Runtime walk/GPS state. No reset needed. |
| `useCityStore` | No | Runtime city selection. Has `reset()`. |
| `useSyncStatus` | No | Runtime sync queue counter. |

## Hydration Ordering Analysis

The startup sequence in `_layout.tsx`:

1. Zustand's `persist` middleware auto-hydrates all persisted stores on import (async, via AsyncStorage).
2. `_layout.tsx` gates on `settingsHydrated && authHydrated` before calling `bootstrapAnonymousAuth()`.
3. `_layout.tsx` gates on `downloadHydrated` before calling `reconcileDownloadState()`.
4. `index.tsx` gates on `settingsHydrated && authHydrated && bootstrapStatus not idle/loading` before rendering routes.

**Risk: Purchase store hydration is not gated** (LOW) -- `_layout.tsx` does not wait for `usePurchaseStore._hasHydrated`. Since purchase status is only used in settings and paywall (not in the routing guard), this is acceptable. However, `canListenFree()` returning `true` when `status` is null (pre-hydration) means a brief window where the paywall is bypassed. This is the intended "allow by default" behavior per the code comment.

**Risk: `authBoundaryReset` subscription races with hydration** (LOW) -- `subscribeAuthBoundaryReset()` captures `previousAccessToken` from the *current* store state at subscription time. It's called in a `useEffect` (after first render), so persisted stores may or may not have hydrated yet. If the auth store hydrates *after* the subscription starts, `previousAccessToken` is `null`, and if hydration restores a non-null token that is then cleared, the boundary reset fires correctly. If hydration restores `null`, the subscription correctly has `previousAccessToken = null` and won't fire. This is safe.

## Auth Boundary Reset Coverage

`authBoundaryReset.ts` detects `accessToken` going from non-null to null and resets:
- `useDownloadStore.clearAllDownloads()` -- clears persisted download metadata
- `usePurchaseStore.setState({ status: null })` -- clears persisted purchase status
- `useSettingsStore.clearPushRegistration()` -- clears push token and registered user ID
- `notificationManager.unregister()` -- deregisters push notifications

**Not reset on logout:**
- `usePlayerStore` -- Not reset. If a story is playing during logout, playback state persists. (LOW risk, unlikely UX scenario)
- `useCityStore` -- Not reset. Selected city persists across sessions. (Acceptable -- city selection is not user-scoped)
- `useSyncStatus` -- Not reset. Pending count could be stale. (LOW risk, counter is re-derived from SQLite on next `initSyncQueue`)
- `useSettingsStore.language/onboardingCompleted/deviceId` -- Not reset. (Correct -- these are device-scoped, not user-scoped)

**Missing: SyncQueue is not flushed or cleared on logout.** If there are queued requests (e.g. listening events) from the previous session, they will be replayed with the new session's auth token after re-authentication. This could attribute actions to the wrong user. The sync queue database (`sync_queue.db`) is only cleared on cache schema version bump, not on auth boundary reset.

## Summary of Findings

### HIGH risk

1. **SyncQueue not cleared on logout** -- Queued offline requests (listening events, reports) survive session clear and will be replayed with the next session's credentials, potentially attributing actions to the wrong anonymous user. The `authBoundaryReset` should clear the sync queue.

### MEDIUM risk

2. **No hydration validation on `useSettingsStore`** -- Unlike `useAuthStore`, there is no `merge` override. Corrupted `deviceId`, `language`, or notification preferences would silently propagate. Should add a sanitizing `merge` function.

3. **Purchase status not reset on user identity change (non-logout)** -- The boundary reset only fires on token null-out. A device re-auth that changes `userId` without going through null (e.g. server reassigns identity) would leave stale purchase data.

### LOW risk

4. **`usePlayerStore` not reset on logout** -- Playback state from previous session could leak. Unlikely to be triggered in practice.

5. **`free_stories_left` local decrement drifts from server** -- Acceptable UX trade-off but the persisted value can be wrong after crash.

6. **Download reconciliation failure leaves stale metadata** -- If SQLite init fails in `reconcileDownloadState`, `downloadedCities` records persist for cities whose audio is gone. User sees "downloaded" badge but playback would fail.

## Recommended Follow-ups

- **Fix #1 (HIGH)**: Clear sync queue in `authBoundaryReset` to prevent cross-session request replay.
- **Fix #2 (MEDIUM)**: Add `merge` validation to `useSettingsStore` similar to `useAuthStore`'s `sanitizePersistedAuthState`.
- **Fix #3 (MEDIUM)**: Detect userId change in boundary reset (not just token null-out) to clear purchase/download state on identity switch.
- **Fix #4 (LOW)**: Reset `usePlayerStore` in auth boundary reset.
