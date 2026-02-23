import { create } from 'zustand';
import type { CacheStats } from '@/services/cache';

interface CacheState {
  stats: CacheStats;
  isClearing: boolean;
}

interface CacheActions {
  setStats: (stats: CacheStats) => void;
  setIsClearing: (clearing: boolean) => void;
}

const initialStats: CacheStats = {
  totalSizeBytes: 0,
  cachedFileCount: 0,
  maxSizeBytes: 100 * 1024 * 1024,
};

export const useCacheStore = create<CacheState & CacheActions>((set) => ({
  stats: initialStats,
  isClearing: false,

  setStats: (stats) => set({ stats }),
  setIsClearing: (clearing) => set({ isClearing: clearing }),
}));
