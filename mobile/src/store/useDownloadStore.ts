import { create } from 'zustand';

export type DownloadStatus = 'idle' | 'fetching_manifest' | 'downloading' | 'completed' | 'error';

interface DownloadProgress {
  completedBytes: number;
  totalBytes: number;
  completedFiles: number;
  totalFiles: number;
}

interface DownloadState {
  status: DownloadStatus;
  progress: DownloadProgress;
  error: string | null;
  downloadedCityIds: Set<number>;
}

interface DownloadActions {
  setStatus: (status: DownloadStatus) => void;
  setProgress: (progress: Partial<DownloadProgress>) => void;
  setError: (error: string | null) => void;
  markCityDownloaded: (cityId: number) => void;
  removeCityDownloaded: (cityId: number) => void;
  reset: () => void;
}

const initialProgress: DownloadProgress = {
  completedBytes: 0,
  totalBytes: 0,
  completedFiles: 0,
  totalFiles: 0,
};

export const useDownloadStore = create<DownloadState & DownloadActions>((set) => ({
  status: 'idle',
  progress: initialProgress,
  error: null,
  downloadedCityIds: new Set<number>(),

  setStatus: (status) => set({ status }),
  setProgress: (progress) => set((state) => ({ progress: { ...state.progress, ...progress } })),
  setError: (error) => set({ error }),
  markCityDownloaded: (cityId) =>
    set((state) => {
      const next = new Set(state.downloadedCityIds);
      next.add(cityId);
      return { downloadedCityIds: next };
    }),
  removeCityDownloaded: (cityId) =>
    set((state) => {
      const next = new Set(state.downloadedCityIds);
      next.delete(cityId);
      return { downloadedCityIds: next };
    }),
  reset: () => set({ status: 'idle', progress: initialProgress, error: null }),
}));
