// Mock AsyncStorage before importing stores
jest.mock('@react-native-async-storage/async-storage', () => ({
  default: {
    getItem: jest.fn().mockResolvedValue(null),
    setItem: jest.fn().mockResolvedValue(undefined),
    removeItem: jest.fn().mockResolvedValue(undefined),
    multiGet: jest.fn().mockResolvedValue([]),
    multiSet: jest.fn().mockResolvedValue(undefined),
    multiRemove: jest.fn().mockResolvedValue(undefined),
    getAllKeys: jest.fn().mockResolvedValue([]),
    clear: jest.fn().mockResolvedValue(undefined),
  },
  __esModule: true,
}));

import React from 'react';
import { act, create, type ReactTestRenderer } from 'react-test-renderer';
import type { CacheStats } from '@/services/cache';
import { useCacheStore } from '@/store/useCacheStore';
import { useDownloadStore } from '@/store/useDownloadStore';
import { useCacheManager } from '../useCacheManager';

(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

function createMockCacheManager() {
  return {
    init: jest.fn().mockResolvedValue(undefined) as jest.Mock<Promise<void>>,
    getStats: jest.fn() as jest.Mock<Promise<CacheStats>>,
    clearAll: jest.fn().mockResolvedValue(undefined) as jest.Mock<Promise<void>>,
    clearStories: jest.fn().mockResolvedValue(undefined) as jest.Mock<Promise<void>>,
  };
}

const defaultStats: CacheStats = {
  totalSizeBytes: 5000,
  cachedFileCount: 3,
  maxSizeBytes: 104857600,
};
const emptyStats: CacheStats = { totalSizeBytes: 0, cachedFileCount: 0, maxSizeBytes: 104857600 };

type HookResult = ReturnType<typeof useCacheManager>;

function HookConsumer({ cm }: { cm: ReturnType<typeof createMockCacheManager> | null }) {
  const result = useCacheManager(cm as unknown as Parameters<typeof useCacheManager>[0]);
  // Expose result via ref on the root element
  return React.createElement('View', { testID: 'hook', hookResult: result });
}

function getHookResult(renderer: ReactTestRenderer): HookResult {
  const root = renderer.root.findByProps({ testID: 'hook' });
  return root.props.hookResult;
}

function resetStores() {
  useCacheStore.setState({
    stats: { totalSizeBytes: 0, cachedFileCount: 0, maxSizeBytes: 104857600 },
    isClearing: false,
    initialized: false,
    error: null,
  });
  useDownloadStore.setState({
    downloadsByCityId: {},
    downloadedCities: {},
    _hasHydrated: true,
  });
}

describe('useCacheManager', () => {
  beforeEach(async () => {
    jest.clearAllMocks();
    await act(() => {
      resetStores();
    });
  });

  it('initializes cache manager and loads stats on mount', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    expect(cm.init).toHaveBeenCalledTimes(1);
    expect(cm.getStats).toHaveBeenCalledTimes(1);

    const result = getHookResult(renderer);
    expect(result.initialized).toBe(true);
    expect(result.stats).toEqual(defaultStats);
    expect(result.error).toBeNull();
  });

  it('sets initialized=true and error when init fails', async () => {
    const cm = createMockCacheManager();
    cm.init.mockRejectedValue(new Error('db broken'));

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    const result = getHookResult(renderer);
    expect(result.initialized).toBe(true);
    expect(result.error).toBe('Failed to load cache stats');
  });

  it('handles null cacheManager gracefully', async () => {
    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm: null }));
    });

    const result = getHookResult(renderer);
    expect(result.initialized).toBe(false);
    expect(result.stats.cachedFileCount).toBe(0);
  });

  it('clearCache clears files, resets downloads, and refreshes stats', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    // Seed a downloaded city
    await act(() => {
      useDownloadStore.setState({
        downloadedCities: {
          '1': {
            cityId: 1,
            language: 'en',
            downloadedAt: 1000,
            storyIds: [10, 11],
            totalFiles: 2,
            totalSizeBytes: 2000,
          },
        },
      });
    });

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    // Now clear
    cm.getStats.mockResolvedValue(emptyStats);
    await act(async () => {
      await getHookResult(renderer).clearCache();
    });

    expect(cm.clearAll).toHaveBeenCalledTimes(1);
    const result = getHookResult(renderer);
    expect(result.stats).toEqual(emptyStats);
    expect(result.downloadedCityCount).toBe(0);
    expect(result.isClearing).toBe(false);
  });

  it('clearCache resets isClearing even on error', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    cm.clearAll.mockRejectedValue(new Error('disk error'));
    await act(async () => {
      try {
        await getHookResult(renderer).clearCache();
      } catch {
        // expected
      }
    });

    expect(getHookResult(renderer).isClearing).toBe(false);
  });

  it('removeCityCache removes a single city and refreshes stats', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    await act(() => {
      useDownloadStore.setState({
        downloadedCities: {
          '1': {
            cityId: 1,
            language: 'en',
            downloadedAt: 1000,
            storyIds: [10, 11],
            totalFiles: 2,
            totalSizeBytes: 2000,
          },
          '2': {
            cityId: 2,
            language: 'en',
            downloadedAt: 2000,
            storyIds: [20],
            totalFiles: 1,
            totalSizeBytes: 1000,
          },
        },
      });
    });

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    const reducedStats: CacheStats = {
      totalSizeBytes: 1000,
      cachedFileCount: 1,
      maxSizeBytes: 104857600,
    };
    cm.getStats.mockResolvedValue(reducedStats);

    await act(async () => {
      await getHookResult(renderer).removeCityCache(1);
    });

    expect(cm.clearStories).toHaveBeenCalledWith([10, 11]);
    const result = getHookResult(renderer);
    expect(result.stats).toEqual(reducedStats);
    expect(result.downloadedCityCount).toBe(1);
    expect(result.downloadedCityList[0].cityId).toBe(2);
    expect(result.isClearing).toBe(false);
  });

  it('removeCityCache is a no-op when city is not downloaded', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    await act(async () => {
      await getHookResult(renderer).removeCityCache(999);
    });

    expect(cm.clearStories).not.toHaveBeenCalled();
  });

  it('reports correct downloadedCityCount and downloadedCityList', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    await act(() => {
      useDownloadStore.setState({
        downloadedCities: {
          '1': {
            cityId: 1,
            language: 'en',
            downloadedAt: 1000,
            storyIds: [10],
            totalFiles: 1,
            totalSizeBytes: 500,
          },
          '5': {
            cityId: 5,
            language: 'ru',
            downloadedAt: 2000,
            storyIds: [50, 51],
            totalFiles: 2,
            totalSizeBytes: 3000,
          },
        },
      });
    });

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    const result = getHookResult(renderer);
    expect(result.downloadedCityCount).toBe(2);
    expect(result.downloadedCityList).toHaveLength(2);
  });

  it('refreshStats updates store with fresh stats', async () => {
    const cm = createMockCacheManager();
    cm.getStats.mockResolvedValue(defaultStats);

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(React.createElement(HookConsumer, { cm }));
    });

    const newStats: CacheStats = {
      totalSizeBytes: 9999,
      cachedFileCount: 7,
      maxSizeBytes: 104857600,
    };
    cm.getStats.mockResolvedValue(newStats);

    await act(async () => {
      await getHookResult(renderer).refreshStats();
    });

    expect(getHookResult(renderer).stats).toEqual(newStats);
  });
});
