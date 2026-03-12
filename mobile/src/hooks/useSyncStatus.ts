import { create } from 'zustand';

interface SyncStatusState {
  pendingCount: number;
  setPendingCount: (count: number) => void;
}

export const useSyncStatus = create<SyncStatusState>((set) => ({
  pendingCount: 0,
  setPendingCount: (count: number) => set({ pendingCount: count }),
}));
