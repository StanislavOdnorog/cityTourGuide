import AsyncStorage from '@react-native-async-storage/async-storage';
import { useDownloadStore } from '../useDownloadStore';
import type { DownloadedCityMeta } from '../useDownloadStore';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn().mockResolvedValue(null),
  setItem: jest.fn().mockResolvedValue(undefined),
  removeItem: jest.fn().mockResolvedValue(undefined),
}));

const mockGetItem = AsyncStorage.getItem as jest.Mock;

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

describe('useDownloadStore', () => {
  beforeEach(() => {
    useDownloadStore.setState({
      downloadsByCityId: {},
      downloadedCities: {},
      _hasHydrated: false,
      _rehydrationError: null,
    });
  });

  it('returns an idle state for cities without an entry', () => {
    const state = useDownloadStore.getState().getCityDownload(1);
    expect(state.status).toBe('idle');
    expect(state.progress.completedBytes).toBe(0);
    expect(state.progress.totalBytes).toBe(0);
    expect(state.progress.completedFiles).toBe(0);
    expect(state.progress.totalFiles).toBe(0);
    expect(state.error).toBeNull();
  });

  it('tracks status per city', () => {
    const store = useDownloadStore.getState();
    store.setCityStatus(1, 'downloading');
    store.setCityStatus(2, 'fetching_manifest');

    expect(store.getCityDownload(1).status).toBe('downloading');
    expect(store.getCityDownload(2).status).toBe('fetching_manifest');
  });

  it('updates progress partially for one city without touching another', () => {
    const store = useDownloadStore.getState();
    store.setCityProgress(1, { totalBytes: 1000, totalFiles: 10 });
    store.setCityProgress(2, { completedBytes: 25, totalBytes: 50 });

    const cityOne = store.getCityDownload(1);
    expect(cityOne.progress.totalBytes).toBe(1000);
    expect(cityOne.progress.totalFiles).toBe(10);
    expect(cityOne.progress.completedBytes).toBe(0);
    expect(cityOne.progress.completedFiles).toBe(0);

    store.setCityProgress(1, { completedFiles: 5, completedBytes: 500 });
    const cityOneUpdated = store.getCityDownload(1);
    expect(cityOneUpdated.progress.completedFiles).toBe(5);
    expect(cityOneUpdated.progress.completedBytes).toBe(500);
    expect(cityOneUpdated.progress.totalBytes).toBe(1000);

    const cityTwo = store.getCityDownload(2);
    expect(cityTwo.progress.completedBytes).toBe(25);
    expect(cityTwo.progress.totalBytes).toBe(50);
  });

  it('stores errors per city', () => {
    const store = useDownloadStore.getState();
    store.setCityError(1, 'Network error');
    store.setCityError(2, 'Disk full');

    expect(store.getCityDownload(1).error).toBe('Network error');
    expect(store.getCityDownload(2).error).toBe('Disk full');
  });

  it('resetCityDownload clears only the targeted city state', () => {
    const store = useDownloadStore.getState();
    store.setCityStatus(1, 'downloading');
    store.setCityProgress(1, { completedFiles: 5, totalFiles: 10 });
    store.setCityError(1, 'error');

    store.setCityStatus(2, 'completed');
    store.setCityProgress(2, { completedFiles: 2, totalFiles: 2 });

    store.resetCityDownload(1);

    expect(store.getCityDownload(1).status).toBe('idle');
    expect(store.getCityDownload(1).progress.completedFiles).toBe(0);
    expect(store.getCityDownload(1).error).toBeNull();

    expect(store.getCityDownload(2).status).toBe('completed');
    expect(store.getCityDownload(2).progress.completedFiles).toBe(2);
  });

  it('markCityDownloaded adds city metadata', () => {
    useDownloadStore.getState().markCityDownloaded(makeMeta(1));
    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(true);
    expect(useDownloadStore.getState().downloadedCities['1'].language).toBe('en');

    useDownloadStore.getState().markCityDownloaded(makeMeta(2));
    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(true);
    expect(useDownloadStore.getState().isCityDownloaded(2)).toBe(true);
  });

  it('markCityDownloaded overwrites existing entry without duplicates', () => {
    const meta1 = makeMeta(1);
    useDownloadStore.getState().markCityDownloaded(meta1);

    const meta1Updated = { ...makeMeta(1), language: 'ru' as const };
    useDownloadStore.getState().markCityDownloaded(meta1Updated);

    expect(useDownloadStore.getState().downloadedCities['1'].language).toBe('ru');
    expect(Object.keys(useDownloadStore.getState().downloadedCities)).toHaveLength(1);
  });

  it('removeCityDownloaded removes city from record', () => {
    useDownloadStore.getState().markCityDownloaded(makeMeta(1));
    useDownloadStore.getState().markCityDownloaded(makeMeta(2));

    useDownloadStore.getState().removeCityDownloaded(1);
    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(false);
    expect(useDownloadStore.getState().isCityDownloaded(2)).toBe(true);
  });

  it('isCityDownloaded returns false for unknown cities', () => {
    expect(useDownloadStore.getState().isCityDownloaded(999)).toBe(false);
  });

  it('clearAllDownloads removes all downloaded cities and resets all download state', () => {
    const store = useDownloadStore.getState();
    store.markCityDownloaded(makeMeta(1));
    store.markCityDownloaded(makeMeta(2));
    store.setCityStatus(1, 'completed');
    store.setCityProgress(1, { completedFiles: 5, totalFiles: 5 });
    store.setCityStatus(2, 'downloading');
    store.setCityProgress(2, { completedFiles: 1, totalFiles: 3 });
    store.setCityError(2, 'Network error');

    store.clearAllDownloads();

    expect(store.isCityDownloaded(1)).toBe(false);
    expect(store.isCityDownloaded(2)).toBe(false);
    expect(Object.keys(useDownloadStore.getState().downloadedCities)).toHaveLength(0);
    expect(Object.keys(useDownloadStore.getState().downloadsByCityId)).toHaveLength(0);
    expect(store.getCityDownload(1).status).toBe('idle');
    expect(store.getCityDownload(2).status).toBe('idle');
  });

  it('clearAllDownloads is a no-op when store is already empty', () => {
    const store = useDownloadStore.getState();
    store.clearAllDownloads();

    expect(Object.keys(useDownloadStore.getState().downloadedCities)).toHaveLength(0);
    expect(Object.keys(useDownloadStore.getState().downloadsByCityId)).toHaveLength(0);
  });

  it('after clearAllDownloads, new downloads can be tracked normally', () => {
    const store = useDownloadStore.getState();
    store.markCityDownloaded(makeMeta(1));
    store.setCityStatus(1, 'completed');

    store.clearAllDownloads();

    store.setCityStatus(3, 'downloading');
    store.markCityDownloaded(makeMeta(3));
    expect(store.isCityDownloaded(3)).toBe(true);
    expect(store.getCityDownload(3).status).toBe('downloading');
    expect(store.isCityDownloaded(1)).toBe(false);
  });

  describe('pre-hydration defaults', () => {
    it('returns defaults before hydration completes', () => {
      useDownloadStore.setState({ _hasHydrated: false });
      const state = useDownloadStore.getState();
      expect(state._hasHydrated).toBe(false);
      expect(state.downloadedCities).toEqual({});
      expect(state.downloadsByCityId).toEqual({});
    });
  });

  describe('corrupted AsyncStorage data', () => {
    it('handles non-JSON garbage gracefully and sets rehydration error', async () => {
      mockGetItem.mockResolvedValueOnce('not valid json {{{');
      await useDownloadStore.persist.rehydrate();

      const state = useDownloadStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state._rehydrationError).toBe('Failed to restore download state.');
      expect(state.downloadedCities).toEqual({});
      expect(state.downloadsByCityId).toEqual({});
    });

    it('handles null persisted state gracefully without error', async () => {
      mockGetItem.mockResolvedValueOnce(null);
      await useDownloadStore.persist.rehydrate();

      const state = useDownloadStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state._rehydrationError).toBeNull();
      expect(state.downloadedCities).toEqual({});
    });

    it('handles state with wrong types gracefully', async () => {
      mockGetItem.mockResolvedValueOnce(
        JSON.stringify({
          state: {
            downloadedCities: 'not-an-object',
          },
          version: 1,
        }),
      );
      await useDownloadStore.persist.rehydrate();

      const state = useDownloadStore.getState();
      expect(state._hasHydrated).toBe(true);
      // Store merges whatever comes back; the key point is it doesn't crash
      expect(state.downloadsByCityId).toEqual({});
    });

    it('handles empty object persisted state gracefully', async () => {
      mockGetItem.mockResolvedValueOnce(JSON.stringify({ state: {}, version: 1 }));
      await useDownloadStore.persist.rehydrate();

      const state = useDownloadStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state.downloadsByCityId).toEqual({});
    });
  });

  it('resetting transient state does not remove downloaded cities', () => {
    const store = useDownloadStore.getState();
    store.setCityStatus(1, 'downloading');
    store.setCityProgress(1, { completedFiles: 5, totalFiles: 10 });
    store.setCityError(1, 'error');
    store.markCityDownloaded(makeMeta(1));

    store.resetCityDownload(1);

    expect(store.getCityDownload(1).status).toBe('idle');
    expect(store.getCityDownload(1).progress.completedFiles).toBe(0);
    expect(store.getCityDownload(1).error).toBeNull();
    expect(store.isCityDownloaded(1)).toBe(true);
  });
});
