import { useAuthStore } from '@/store/useAuthStore';
import { useDownloadStore } from '@/store/useDownloadStore';
import type { DownloadedCityMeta } from '@/store/useDownloadStore';
import { usePurchaseStore } from '@/store/usePurchaseStore';
import { useSettingsStore } from '@/store/useSettingsStore';
// eslint-disable-next-line import/order
import type { User } from '@/types';
const mockGetItem = jest.fn().mockResolvedValue(null);
const mockSetItem = jest.fn().mockResolvedValue(undefined);
const mockRemoveItem = jest.fn().mockResolvedValue(undefined);

jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: (...args: unknown[]) => mockGetItem(...args),
  setItem: (...args: unknown[]) => mockSetItem(...args),
  removeItem: (...args: unknown[]) => mockRemoveItem(...args),
}));

// Mock expo-sqlite
const mockRunAsync = jest.fn().mockResolvedValue(undefined);
const mockGetAllAsync = jest.fn().mockResolvedValue([]);
const mockExecAsync = jest.fn().mockResolvedValue(undefined);
const mockCloseAsync = jest.fn().mockResolvedValue(undefined);

jest.mock('expo-sqlite', () => ({
  openDatabaseAsync: jest.fn().mockResolvedValue({
    runAsync: (...args: unknown[]) => mockRunAsync(...args),
    getAllAsync: (...args: unknown[]) => mockGetAllAsync(...args),
    execAsync: (...args: unknown[]) => mockExecAsync(...args),
    closeAsync: () => mockCloseAsync(),
  }),
}));

import {
  getStoredCacheVersion,
  stampCacheVersion,
  checkAndMigrateCacheSchema,
  CURRENT_CACHE_SCHEMA_VERSION,
} from '../CacheSchemaVersion';

function makeMeta(cityId: number): DownloadedCityMeta {
  return {
    cityId,
    language: 'en',
    downloadedAt: Date.now(),
    storyIds: [100, 101],
    totalFiles: 2,
    totalSizeBytes: 5000,
  };
}

describe('CacheSchemaVersion', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useDownloadStore.setState({
      downloadsByCityId: {},
      downloadedCities: {},
      _hasHydrated: false,
    });
    usePurchaseStore.setState({
      status: null,
      isLoading: false,
      paywallVisible: false,
      _hasHydrated: false,
    });
  });

  describe('getStoredCacheVersion', () => {
    it('returns null when no version is stored', async () => {
      mockGetItem.mockResolvedValueOnce(null);
      expect(await getStoredCacheVersion()).toBeNull();
    });

    it('returns the stored version number', async () => {
      mockGetItem.mockResolvedValueOnce('1');
      expect(await getStoredCacheVersion()).toBe(1);
    });

    it('returns null for malformed non-numeric value', async () => {
      mockGetItem.mockResolvedValueOnce('garbage');
      expect(await getStoredCacheVersion()).toBeNull();
    });

    it('returns null for negative version', async () => {
      mockGetItem.mockResolvedValueOnce('-1');
      expect(await getStoredCacheVersion()).toBeNull();
    });

    it('returns null for floating-point version', async () => {
      mockGetItem.mockResolvedValueOnce('1.5');
      expect(await getStoredCacheVersion()).toBeNull();
    });

    it('returns null for empty string', async () => {
      mockGetItem.mockResolvedValueOnce('');
      expect(await getStoredCacheVersion()).toBeNull();
    });

    it('returns null when AsyncStorage throws', async () => {
      mockGetItem.mockRejectedValueOnce(new Error('storage error'));
      expect(await getStoredCacheVersion()).toBeNull();
    });
  });

  describe('stampCacheVersion', () => {
    it('writes the current version to AsyncStorage', async () => {
      await stampCacheVersion();
      expect(mockSetItem).toHaveBeenCalledWith(
        'city-stories-cache-schema-version',
        String(CURRENT_CACHE_SCHEMA_VERSION),
      );
    });
  });

  describe('checkAndMigrateCacheSchema', () => {
    it('returns false and does not invalidate when version matches', async () => {
      mockGetItem.mockResolvedValueOnce(String(CURRENT_CACHE_SCHEMA_VERSION));

      const result = await checkAndMigrateCacheSchema();

      expect(result).toBe(false);
      // Should not have written a cache version key
      expect(mockSetItem).not.toHaveBeenCalledWith(
        'city-stories-cache-schema-version',
        expect.anything(),
      );
    });

    it('invalidates cache when version is missing (fresh install or pre-versioning)', async () => {
      // Seed download store with data
      useDownloadStore.getState().markCityDownloaded(makeMeta(1));
      usePurchaseStore.setState({
        status: {
          has_full_access: false,
          is_lifetime: false,
          free_stories_used: 1,
          free_stories_limit: 5,
          free_stories_left: 4,
          city_packs: [],
        },
      });

      mockGetItem.mockResolvedValueOnce(null);

      const result = await checkAndMigrateCacheSchema();

      expect(result).toBe(true);
      // Downloads cleared
      expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(false);
      // Purchase status cleared
      expect(usePurchaseStore.getState().status).toBeNull();
      // Version stamped
      expect(mockSetItem).toHaveBeenCalledWith(
        'city-stories-cache-schema-version',
        String(CURRENT_CACHE_SCHEMA_VERSION),
      );
    });

    it('invalidates cache when version is outdated', async () => {
      useDownloadStore.getState().markCityDownloaded(makeMeta(2));
      mockGetItem.mockResolvedValueOnce('0');

      const result = await checkAndMigrateCacheSchema();

      expect(result).toBe(true);
      expect(useDownloadStore.getState().isCityDownloaded(2)).toBe(false);
      expect(mockSetItem).toHaveBeenCalledWith(
        'city-stories-cache-schema-version',
        String(CURRENT_CACHE_SCHEMA_VERSION),
      );
    });

    it('invalidates cache when version is malformed', async () => {
      useDownloadStore.getState().markCityDownloaded(makeMeta(3));
      mockGetItem.mockResolvedValueOnce('not-a-number');

      const result = await checkAndMigrateCacheSchema();

      expect(result).toBe(true);
      expect(useDownloadStore.getState().isCityDownloaded(3)).toBe(false);
    });

    it('drops SQLite tables for story_cache.db and sync_queue.db', async () => {
      mockGetItem.mockResolvedValueOnce(null);
      // Return table names when querying sqlite_master
      mockGetAllAsync.mockResolvedValue([{ name: 'cached_stories' }]);

      await checkAndMigrateCacheSchema();

      // Should have opened databases and dropped tables
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { openDatabaseAsync } = require('expo-sqlite') as { openDatabaseAsync: jest.Mock };
      expect(openDatabaseAsync).toHaveBeenCalledWith('story_cache.db');
      expect(openDatabaseAsync).toHaveBeenCalledWith('sync_queue.db');
      expect(mockExecAsync).toHaveBeenCalledWith('DROP TABLE IF EXISTS "cached_stories"');
    });

    it('preserves user settings during invalidation', async () => {
      // Seed settings store with user preferences
      useSettingsStore.setState({
        language: 'ru',
        onboardingCompleted: true,
        deviceId: 'test-device-id-123',
        geoNotifications: false,
        contentNotifications: true,
        pushToken: 'push-token-abc',
        registeredPushUserId: 'user-xyz',
      });

      // Seed download store with data that should be cleared
      useDownloadStore.getState().markCityDownloaded(makeMeta(1));
      usePurchaseStore.setState({
        status: {
          has_full_access: true,
          is_lifetime: true,
          free_stories_used: 5,
          free_stories_limit: 5,
          free_stories_left: 0,
          city_packs: [],
        },
      });

      mockGetItem.mockResolvedValueOnce(null);

      await checkAndMigrateCacheSchema();

      // Download and purchase state should be cleared
      expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(false);
      expect(usePurchaseStore.getState().status).toBeNull();

      // Settings state must be fully preserved
      const settings = useSettingsStore.getState();
      expect(settings.language).toBe('ru');
      expect(settings.onboardingCompleted).toBe(true);
      expect(settings.deviceId).toBe('test-device-id-123');
      expect(settings.geoNotifications).toBe(false);
      expect(settings.contentNotifications).toBe(true);
      expect(settings.pushToken).toBe('push-token-abc');
      expect(settings.registeredPushUserId).toBe('user-xyz');
    });

    it('preserves auth state during invalidation', async () => {
      // Seed auth store with active session
      useAuthStore.setState({
        user: { id: 'u-1', email: 'test@example.com' } as unknown as User,
        userId: 'u-1',
        accessToken: 'access-token-secret',
        refreshToken: 'refresh-token-secret',
        _hasHydrated: true,
        bootstrapStatus: 'ready',
        bootstrapError: null,
      });

      // Seed download store with data that should be cleared
      useDownloadStore.getState().markCityDownloaded(makeMeta(1));

      mockGetItem.mockResolvedValueOnce('0'); // outdated version

      await checkAndMigrateCacheSchema();

      // Download state should be cleared
      expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(false);

      // Auth state must be fully preserved
      const auth = useAuthStore.getState();
      expect(auth.userId).toBe('u-1');
      expect(auth.accessToken).toBe('access-token-secret');
      expect(auth.refreshToken).toBe('refresh-token-secret');
      expect(auth.user).toEqual(expect.objectContaining({ id: 'u-1', email: 'test@example.com' }));
    });

    it('preserves both settings and auth when version is malformed', async () => {
      useSettingsStore.setState({
        language: 'en',
        onboardingCompleted: true,
        deviceId: 'device-42',
      });
      useAuthStore.setState({
        userId: 'u-2',
        accessToken: 'tok-2',
        refreshToken: 'ref-2',
      });
      useDownloadStore.getState().markCityDownloaded(makeMeta(5));

      mockGetItem.mockResolvedValueOnce('garbage');

      await checkAndMigrateCacheSchema();

      // Cache cleared
      expect(useDownloadStore.getState().isCityDownloaded(5)).toBe(false);
      expect(usePurchaseStore.getState().status).toBeNull();

      // Settings and auth intact
      expect(useSettingsStore.getState().deviceId).toBe('device-42');
      expect(useSettingsStore.getState().onboardingCompleted).toBe(true);
      expect(useAuthStore.getState().userId).toBe('u-2');
      expect(useAuthStore.getState().accessToken).toBe('tok-2');
    });

    it('handles SQLite errors gracefully without crashing', async () => {
      mockGetItem.mockResolvedValueOnce(null);
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { openDatabaseAsync } = require('expo-sqlite') as { openDatabaseAsync: jest.Mock };
      openDatabaseAsync.mockRejectedValueOnce(new Error('db error'));

      // Should not throw even if SQLite fails
      const result = await checkAndMigrateCacheSchema();
      expect(result).toBe(true);
      // Version should still be stamped
      expect(mockSetItem).toHaveBeenCalledWith(
        'city-stories-cache-schema-version',
        String(CURRENT_CACHE_SCHEMA_VERSION),
      );
    });

    it('clears multiple downloaded cities at once', async () => {
      useDownloadStore.getState().markCityDownloaded(makeMeta(1));
      useDownloadStore.getState().markCityDownloaded(makeMeta(2));
      useDownloadStore.getState().markCityDownloaded(makeMeta(3));
      mockGetItem.mockResolvedValueOnce(null);

      await checkAndMigrateCacheSchema();

      expect(Object.keys(useDownloadStore.getState().downloadedCities)).toHaveLength(0);
    });
  });
});
