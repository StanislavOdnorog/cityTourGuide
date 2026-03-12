import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SQLite from 'expo-sqlite';
import { useDownloadStore } from '@/store/useDownloadStore';
import { usePurchaseStore } from '@/store/usePurchaseStore';

const CACHE_VERSION_KEY = 'city-stories-cache-schema-version';

/**
 * Bump this number whenever the persisted cache/download schema changes
 * in a way that is incompatible with previously stored data.
 *
 * Version history:
 *   1 — initial versioned schema (downloads, audio cache, purchases)
 */
export const CURRENT_CACHE_SCHEMA_VERSION = 1;

/**
 * Read the stored cache schema version.
 * Returns null if no version has been stamped yet.
 */
export async function getStoredCacheVersion(): Promise<number | null> {
  try {
    const raw = await AsyncStorage.getItem(CACHE_VERSION_KEY);
    if (raw == null || raw === '') return null;

    const parsed = Number(raw);
    if (!Number.isFinite(parsed) || parsed < 0 || parsed !== Math.floor(parsed)) {
      return null; // malformed
    }
    return parsed;
  } catch {
    return null;
  }
}

/**
 * Persist the current cache schema version marker.
 */
export async function stampCacheVersion(): Promise<void> {
  await AsyncStorage.setItem(CACHE_VERSION_KEY, String(CURRENT_CACHE_SCHEMA_VERSION));
}

/**
 * Check the stored cache schema version against the current version.
 * If the version is missing, outdated, or malformed, clear incompatible
 * cached data (downloads, purchases, audio cache) while preserving
 * user settings and auth state, then stamp the current version.
 *
 * Must be called after Zustand stores have hydrated but before screens
 * rely on cached downloads.
 *
 * Returns true if invalidation was performed.
 */
export async function checkAndMigrateCacheSchema(): Promise<boolean> {
  const storedVersion = await getStoredCacheVersion();

  if (storedVersion === CURRENT_CACHE_SCHEMA_VERSION) {
    return false; // up to date
  }

  // Version is missing, outdated, or malformed — invalidate cache state.
  // Future migrations can be added here for specific version transitions
  // (e.g. if storedVersion === 1, migrate to 2 without full clear).

  await invalidateOfflineCache();
  await stampCacheVersion();
  return true;
}

/**
 * Clear all offline/cache state that depends on the cache schema,
 * preserving user settings (language, deviceId, onboarding, notifications)
 * and auth state (tokens, user).
 */
async function invalidateOfflineCache(): Promise<void> {
  // 1. Clear Zustand download store (persisted city metadata)
  useDownloadStore.getState().clearAllDownloads();

  // 2. Clear Zustand purchase cache (will be re-fetched from server)
  usePurchaseStore.setState({ status: null });

  // 3. Clear SQLite audio cache database
  await clearSQLiteDatabase('story_cache.db');

  // 4. Clear SQLite sync queue database
  await clearSQLiteDatabase('sync_queue.db');
}

async function clearSQLiteDatabase(dbName: string): Promise<void> {
  let db: SQLite.SQLiteDatabase | null = null;
  try {
    db = await SQLite.openDatabaseAsync(dbName);
    // Get all user-created tables and drop them
    const tables = await db.getAllAsync<{ name: string }>(
      "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'",
    );
    for (const table of tables) {
      await db.execAsync(`DROP TABLE IF EXISTS "${table.name}"`);
    }
  } catch {
    // Database may not exist yet — that's fine
  } finally {
    if (db) {
      try {
        await db.closeAsync();
      } catch {
        // Ignore close errors
      }
    }
  }
}
