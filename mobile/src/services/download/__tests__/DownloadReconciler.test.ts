import { useDownloadStore } from '@/store/useDownloadStore';
import type { DownloadedCityMeta } from '@/store/useDownloadStore';
import { reconcileDownloadState } from '../DownloadReconciler';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn().mockResolvedValue(null),
  setItem: jest.fn().mockResolvedValue(undefined),
  removeItem: jest.fn().mockResolvedValue(undefined),
}));

const mockIsCached = jest.fn<Promise<boolean>, [number]>();
const mockInit = jest.fn().mockResolvedValue(undefined);
const mockDestroy = jest.fn().mockResolvedValue(undefined);
const mockReconcile = jest.fn().mockResolvedValue({ removedRows: 0, deletedFiles: 0 });

jest.mock('@/services/cache', () => ({
  StoryCacheManager: jest.fn().mockImplementation(() => ({
    init: mockInit,
    isCached: mockIsCached,
    destroy: mockDestroy,
    reconcile: mockReconcile,
  })),
}));

function makeMeta(cityId: number, storyIds: number[]): DownloadedCityMeta {
  return {
    cityId,
    language: 'en',
    downloadedAt: Date.now(),
    storyIds,
    totalFiles: storyIds.length,
    totalSizeBytes: storyIds.length * 1000,
  };
}

describe('reconcileDownloadState', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useDownloadStore.setState({ downloadedCities: {} });
  });

  it('calls reconcile on cache manager even with no downloaded cities', async () => {
    await reconcileDownloadState();
    expect(mockInit).toHaveBeenCalled();
    expect(mockReconcile).toHaveBeenCalled();
    expect(mockDestroy).toHaveBeenCalled();
  });

  it('keeps cities whose files are still cached', async () => {
    useDownloadStore.setState({
      downloadedCities: { '1': makeMeta(1, [10, 11]) },
    });
    mockIsCached.mockResolvedValue(true);

    await reconcileDownloadState();

    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(true);
    expect(mockDestroy).toHaveBeenCalled();
  });

  it('removes cities whose files are all missing', async () => {
    useDownloadStore.setState({
      downloadedCities: { '1': makeMeta(1, [10, 11]) },
    });
    mockIsCached.mockResolvedValue(false);

    await reconcileDownloadState();

    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(false);
  });

  it('keeps city if at least one file is cached', async () => {
    useDownloadStore.setState({
      downloadedCities: { '1': makeMeta(1, [10, 11, 12]) },
    });
    // First story missing, second present
    mockIsCached.mockResolvedValueOnce(false).mockResolvedValueOnce(true);

    await reconcileDownloadState();

    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(true);
    // Should have short-circuited after finding one cached file
    expect(mockIsCached).toHaveBeenCalledTimes(2);
  });

  it('removes entry with empty storyIds', async () => {
    useDownloadStore.setState({
      downloadedCities: { '5': makeMeta(5, []) },
    });

    await reconcileDownloadState();

    expect(useDownloadStore.getState().isCityDownloaded(5)).toBe(false);
  });

  it('handles multiple cities independently', async () => {
    useDownloadStore.setState({
      downloadedCities: {
        '1': makeMeta(1, [10]),
        '2': makeMeta(2, [20]),
      },
    });
    // City 1 cached, city 2 not
    mockIsCached.mockImplementation(async (storyId: number) => storyId === 10);

    await reconcileDownloadState();

    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(true);
    expect(useDownloadStore.getState().isCityDownloaded(2)).toBe(false);
  });

  it('always destroys the cache manager even on error', async () => {
    useDownloadStore.setState({
      downloadedCities: { '1': makeMeta(1, [10]) },
    });
    mockIsCached.mockRejectedValue(new Error('DB error'));

    await expect(reconcileDownloadState()).rejects.toThrow('DB error');
    expect(mockDestroy).toHaveBeenCalled();
  });
});
