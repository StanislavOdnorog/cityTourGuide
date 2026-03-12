import { useCallback, useEffect, useRef } from 'react';
import type { StoryCacheManager } from '@/services/cache';
import { useCacheStore } from '@/store/useCacheStore';
import { useDownloadStore, type DownloadedCityMeta } from '@/store/useDownloadStore';

/**
 * Hook to interact with the story cache from UI components.
 * Handles initialization, provides cache stats, and actions (refresh, clear, remove city).
 */
export function useCacheManager(cacheManager: StoryCacheManager | null) {
  const stats = useCacheStore((s) => s.stats);
  const isClearing = useCacheStore((s) => s.isClearing);
  const initialized = useCacheStore((s) => s.initialized);
  const error = useCacheStore((s) => s.error);
  const setStats = useCacheStore((s) => s.setStats);
  const setIsClearing = useCacheStore((s) => s.setIsClearing);
  const setInitialized = useCacheStore((s) => s.setInitialized);
  const setError = useCacheStore((s) => s.setError);

  const clearAllDownloads = useDownloadStore((s) => s.clearAllDownloads);
  const removeCityDownloaded = useDownloadStore((s) => s.removeCityDownloaded);
  const resetCityDownload = useDownloadStore((s) => s.resetCityDownload);
  const downloadedCities = useDownloadStore((s) => s.downloadedCities);

  const managerRef = useRef(cacheManager);
  managerRef.current = cacheManager;

  const refreshStats = useCallback(async () => {
    if (!managerRef.current) return;
    try {
      const newStats = await managerRef.current.getStats();
      setStats(newStats);
      setError(null);
    } catch {
      // Silently fail — cache may not be initialized yet
    }
  }, [setStats, setError]);

  // Initialize cache manager and load stats on mount
  useEffect(() => {
    if (!cacheManager) return;
    let cancelled = false;

    void (async () => {
      try {
        await cacheManager.init();
        if (cancelled) return;
        const newStats = await cacheManager.getStats();
        if (cancelled) return;
        setStats(newStats);
        setError(null);
        setInitialized(true);
      } catch {
        if (!cancelled) {
          setError('Failed to load cache stats');
          setInitialized(true);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [cacheManager, setStats, setInitialized, setError]);

  const clearCache = useCallback(async () => {
    if (!managerRef.current) return;
    setIsClearing(true);
    try {
      await managerRef.current.clearAll();
      clearAllDownloads();
      await refreshStats();
    } finally {
      setIsClearing(false);
    }
  }, [refreshStats, setIsClearing, clearAllDownloads]);

  const removeCityCache = useCallback(
    async (cityId: number) => {
      if (!managerRef.current) return;
      const meta: DownloadedCityMeta | undefined = downloadedCities[String(cityId)];
      if (!meta) return;

      setIsClearing(true);
      try {
        await managerRef.current.clearStories(meta.storyIds);
        removeCityDownloaded(cityId);
        resetCityDownload(cityId);
        await refreshStats();
      } finally {
        setIsClearing(false);
      }
    },
    [refreshStats, setIsClearing, removeCityDownloaded, resetCityDownload, downloadedCities],
  );

  const downloadedCityList = Object.values(downloadedCities);
  const downloadedCityCount = downloadedCityList.length;

  return {
    stats,
    isClearing,
    initialized,
    error,
    refreshStats,
    clearCache,
    removeCityCache,
    downloadedCityCount,
    downloadedCityList,
  };
}
