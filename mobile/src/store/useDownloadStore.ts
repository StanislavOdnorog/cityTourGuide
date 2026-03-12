import AsyncStorage from '@react-native-async-storage/async-storage';
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type DownloadStatus = 'idle' | 'fetching_manifest' | 'downloading' | 'completed' | 'error';

export interface DownloadedCityMeta {
  cityId: number;
  language: string;
  downloadedAt: number;
  storyIds: number[];
  totalFiles: number;
  totalSizeBytes: number;
}

export interface DownloadProgress {
  completedBytes: number;
  totalBytes: number;
  completedFiles: number;
  totalFiles: number;
}

export interface CityDownloadState {
  status: DownloadStatus;
  progress: DownloadProgress;
  error: string | null;
}

interface DownloadState {
  /** Keyed by cityId (as string for JSON serialization). */
  downloadsByCityId: Record<string, CityDownloadState>;
  /** Keyed by cityId (as string for JSON serialization). */
  downloadedCities: Record<string, DownloadedCityMeta>;
  _hasHydrated: boolean;
  _rehydrationError: string | null;
}

interface DownloadActions {
  setCityStatus: (cityId: number, status: DownloadStatus) => void;
  setCityProgress: (cityId: number, progress: Partial<DownloadProgress>) => void;
  setCityError: (cityId: number, error: string | null) => void;
  getCityDownload: (cityId: number) => CityDownloadState;
  markCityDownloaded: (meta: DownloadedCityMeta) => void;
  removeCityDownloaded: (cityId: number) => void;
  isCityDownloaded: (cityId: number) => boolean;
  resetCityDownload: (cityId: number) => void;
  clearAllDownloads: () => void;
  setHasHydrated: (value: boolean) => void;
}

const initialProgress = (): DownloadProgress => ({
  completedBytes: 0,
  totalBytes: 0,
  completedFiles: 0,
  totalFiles: 0,
});

const createInitialCityDownloadState = (): CityDownloadState => ({
  status: 'idle',
  progress: initialProgress(),
  error: null,
});

function getStoredCityDownload(
  downloadsByCityId: Record<string, CityDownloadState>,
  cityId: number,
): CityDownloadState {
  return downloadsByCityId[String(cityId)] ?? createInitialCityDownloadState();
}

export const useDownloadStore = create<DownloadState & DownloadActions>()(
  persist(
    (set, get) => ({
      downloadsByCityId: {},
      downloadedCities: {},
      _hasHydrated: false,
      _rehydrationError: null,

      setCityStatus: (cityId, status) =>
        set((state) => {
          const cityKey = String(cityId);
          const current = getStoredCityDownload(state.downloadsByCityId, cityId);
          return {
            downloadsByCityId: {
              ...state.downloadsByCityId,
              [cityKey]: { ...current, status },
            },
          };
        }),
      setCityProgress: (cityId, progress) =>
        set((state) => {
          const cityKey = String(cityId);
          const current = getStoredCityDownload(state.downloadsByCityId, cityId);
          return {
            downloadsByCityId: {
              ...state.downloadsByCityId,
              [cityKey]: {
                ...current,
                progress: { ...current.progress, ...progress },
              },
            },
          };
        }),
      setCityError: (cityId, error) =>
        set((state) => {
          const cityKey = String(cityId);
          const current = getStoredCityDownload(state.downloadsByCityId, cityId);
          return {
            downloadsByCityId: {
              ...state.downloadsByCityId,
              [cityKey]: { ...current, error },
            },
          };
        }),
      getCityDownload: (cityId) => getStoredCityDownload(get().downloadsByCityId, cityId),
      markCityDownloaded: (meta) =>
        set((state) => ({
          downloadedCities: {
            ...state.downloadedCities,
            [String(meta.cityId)]: meta,
          },
        })),
      removeCityDownloaded: (cityId) =>
        set((state) => {
          const { [String(cityId)]: _, ...rest } = state.downloadedCities;
          return { downloadedCities: rest };
        }),
      isCityDownloaded: (cityId) => String(cityId) in get().downloadedCities,
      resetCityDownload: (cityId) =>
        set((state) => ({
          downloadsByCityId: {
            ...state.downloadsByCityId,
            [String(cityId)]: createInitialCityDownloadState(),
          },
        })),
      clearAllDownloads: () => set({ downloadsByCityId: {}, downloadedCities: {} }),
      setHasHydrated: (value) => set({ _hasHydrated: value }),
    }),
    {
      name: 'city-stories-downloads',
      storage: createJSONStorage(() => AsyncStorage),
      version: 1,
      migrate: (persisted, version) => {
        const state = (persisted ?? {}) as Partial<Pick<DownloadState, 'downloadedCities'>>;
        if (version === 0) {
          return {
            downloadedCities: state.downloadedCities ?? {},
          };
        }
        return state as Pick<DownloadState, 'downloadedCities'>;
      },
      partialize: (state) => ({
        downloadedCities: state.downloadedCities,
      }),
      onRehydrateStorage: () => (_state, error) => {
        if (error) {
          console.warn('useDownloadStore: rehydration failed', error);
        }
        useDownloadStore.setState({
          _hasHydrated: true,
          _rehydrationError: error ? 'Failed to restore download state.' : null,
        });
      },
    },
  ),
);
