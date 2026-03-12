import { create } from 'zustand';
import type { CacheStats } from '@/services/cache';

interface CacheState {
  stats: CacheStats;
  isClearing: boolean;
  initialized: boolean;
  error: string | null;
}

interface CacheActions {
  setStats: (stats: CacheStats) => void;
  setIsClearing: (clearing: boolean) => void;
  setInitialized: (initialized: boolean) => void;
  setError: (error: string | null) => void;
}

const initialStats: CacheStats = {
  totalSizeBytes: 0,
  cachedFileCount: 0,
  maxSizeBytes: 100 * 1024 * 1024,
};

export const useCacheStore = create<CacheState & CacheActions>((set) => ({
  stats: initialStats,
  isClearing: false,
  initialized: false,
  error: null,

  setStats: (stats) => set({ stats }),
  setIsClearing: (clearing) => set({ isClearing: clearing }),
  setInitialized: (initialized) => set({ initialized }),
  setError: (error) => set({ error }),
}));
