import { useCallback, useEffect, useRef } from 'react';
import type { StoryCacheManager } from '@/services/cache';
import { useCacheStore } from '@/store/useCacheStore';

/**
 * Hook to interact with the story cache from UI components.
 * Provides cache stats and actions (refresh, clear).
 */
export function useCacheManager(cacheManager: StoryCacheManager | null) {
  const { stats, isClearing, setStats, setIsClearing } = useCacheStore();
  const managerRef = useRef(cacheManager);
  managerRef.current = cacheManager;

  const refreshStats = useCallback(async () => {
    if (!managerRef.current) return;
    try {
      const newStats = await managerRef.current.getStats();
      setStats(newStats);
    } catch {
      // Silently fail — cache may not be initialized yet
    }
  }, [setStats]);

  const clearCache = useCallback(async () => {
    if (!managerRef.current) return;
    setIsClearing(true);
    try {
      await managerRef.current.clearAll();
      await refreshStats();
    } finally {
      setIsClearing(false);
    }
  }, [refreshStats, setIsClearing]);

  // Refresh stats on mount and when cacheManager changes
  useEffect(() => {
    void refreshStats();
  }, [refreshStats, cacheManager]);

  return {
    stats,
    isClearing,
    refreshStats,
    clearCache,
  };
}
