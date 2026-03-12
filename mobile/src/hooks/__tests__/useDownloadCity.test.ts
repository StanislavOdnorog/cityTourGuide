// Mock AsyncStorage before importing store
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
import type { CancelableDownload, FileDownloadProgress } from '@/services/cache/StoryCacheManager';
import { useDownloadStore } from '@/store/useDownloadStore';
import { useDownloadCity } from '../useDownloadCity';

(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mockFetchManifest = jest.fn();
jest.mock('@/api/endpoints', () => ({
  fetchCityDownloadManifest: (...args: unknown[]) => mockFetchManifest(...args),
}));

type DownloadHandleMock = {
  callback?: (progress: FileDownloadProgress) => void;
  cancel: jest.Mock<Promise<void>, []>;
  promise: Promise<string | null>;
  reject: (error: Error) => void;
  resolve: (value: string | null) => void;
};

const mockInit = jest.fn().mockResolvedValue(undefined);
const downloadHandles: DownloadHandleMock[] = [];

const mockDownloadAudioWithProgress = jest
  .fn()
  .mockImplementation(
    (
      _candidate: unknown,
      onProgress?: (progress: FileDownloadProgress) => void,
    ): CancelableDownload => {
      let resolve!: (value: string | null) => void;
      let reject!: (error: Error) => void;
      const promise = new Promise<string | null>((res, rej) => {
        resolve = res;
        reject = rej;
      });

      const handle: DownloadHandleMock = {
        callback: onProgress,
        cancel: jest.fn().mockResolvedValue(undefined),
        promise,
        reject,
        resolve,
      };
      downloadHandles.push(handle);
      return { promise, cancel: handle.cancel };
    },
  );

jest.mock('@/services/cache', () => ({
  StoryCacheManager: jest.fn().mockImplementation(() => ({
    init: mockInit,
    downloadAudioWithProgress: mockDownloadAudioWithProgress,
  })),
}));

function makeManifest(
  items: Array<{
    story_id: number;
    poi_id: number;
    audio_url: string | null;
    file_size_bytes: number;
    duration_sec: number | null;
    poi_name: string;
  }>,
) {
  return {
    data: items,
    total_size_bytes: items.reduce((sum, item) => sum + item.file_size_bytes, 0),
    total_stories: items.length,
    city_name: 'Test City',
  };
}

const CITY_ONE_ITEMS = [
  {
    story_id: 1,
    poi_id: 1,
    poi_name: 'POI A',
    audio_url: 'https://example.com/1.mp3',
    file_size_bytes: 50000,
    duration_sec: 30,
  },
];

const CITY_TWO_ITEMS = [
  {
    story_id: 2,
    poi_id: 2,
    poi_name: 'POI B',
    audio_url: 'https://example.com/2.mp3',
    file_size_bytes: 30000,
    duration_sec: 20,
  },
];

type HookSnapshot = ReturnType<typeof useDownloadCity>;

const hookResults = new Map<string, HookSnapshot>();
const originalConsoleError = console.error;

function HookHarness({ cityId, hookId }: { cityId: number; hookId: string }) {
  hookResults.set(hookId, useDownloadCity(cityId));
  return null;
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}

function getResult(hookId: string): HookSnapshot {
  const result = hookResults.get(hookId);
  if (!result) {
    throw new Error(`Missing hook result for ${hookId}`);
  }
  return result;
}

async function startDownload(hookId: string) {
  await act(async () => {
    void getResult(hookId).startDownload();
    await flushPromises();
  });
}

async function resolveDownload(handleIndex: number, value: string | null) {
  await act(async () => {
    downloadHandles[handleIndex].resolve(value);
    await flushPromises();
  });
}

describe('useDownloadCity', () => {
  let renderer: ReactTestRenderer | null = null;

  beforeAll(() => {
    jest.spyOn(console, 'error').mockImplementation((...args: unknown[]) => {
      if (typeof args[0] === 'string' && args[0].includes('react-test-renderer is deprecated')) {
        return;
      }
      originalConsoleError(...args);
    });
  });

  beforeEach(() => {
    jest.clearAllMocks();
    downloadHandles.length = 0;
    hookResults.clear();
    useDownloadStore.setState({
      downloadsByCityId: {},
      downloadedCities: {},
      _hasHydrated: false,
    });
  });

  afterEach(() => {
    if (renderer) {
      act(() => {
        renderer?.unmount();
      });
      renderer = null;
    }
  });

  afterAll(() => {
    jest.restoreAllMocks();
  });

  it('starting one city download does not overwrite another city progress', async () => {
    mockFetchManifest
      .mockResolvedValueOnce(makeManifest(CITY_ONE_ITEMS))
      .mockResolvedValueOnce(makeManifest(CITY_TWO_ITEMS));

    await act(async () => {
      renderer = create(
        React.createElement(
          React.Fragment,
          null,
          React.createElement(HookHarness, { cityId: 1, hookId: 'city-1' }),
          React.createElement(HookHarness, { cityId: 2, hookId: 'city-2' }),
        ),
      );
      await flushPromises();
    });

    await startDownload('city-1');

    await act(async () => {
      downloadHandles[0].callback?.({ bytesWritten: 10000, contentLength: 50000 });
      await flushPromises();
    });

    expect(useDownloadStore.getState().getCityDownload(1).status).toBe('downloading');
    expect(useDownloadStore.getState().getCityDownload(1).progress.completedBytes).toBe(10000);
    expect(useDownloadStore.getState().getCityDownload(2).status).toBe('idle');

    await startDownload('city-2');

    expect(useDownloadStore.getState().getCityDownload(1).status).toBe('downloading');
    expect(useDownloadStore.getState().getCityDownload(1).progress.completedBytes).toBe(10000);
    expect(useDownloadStore.getState().getCityDownload(2).status).toBe('downloading');

    await resolveDownload(1, 'file:///cache/story_2.mp3');

    expect(useDownloadStore.getState().isCityDownloaded(2)).toBe(true);
    expect(useDownloadStore.getState().getCityDownload(2).status).toBe('completed');
    expect(useDownloadStore.getState().getCityDownload(1).progress.completedBytes).toBe(10000);
    expect(getResult('city-2').isDownloaded).toBe(true);
  });

  it('cancelling one city resets only that city state', async () => {
    mockFetchManifest
      .mockResolvedValueOnce(makeManifest(CITY_ONE_ITEMS))
      .mockResolvedValueOnce(makeManifest(CITY_TWO_ITEMS));

    await act(async () => {
      renderer = create(
        React.createElement(
          React.Fragment,
          null,
          React.createElement(HookHarness, { cityId: 1, hookId: 'city-1' }),
          React.createElement(HookHarness, { cityId: 2, hookId: 'city-2' }),
        ),
      );
      await flushPromises();
    });

    await startDownload('city-1');
    await startDownload('city-2');

    await act(async () => {
      getResult('city-1').cancelDownload();
      await flushPromises();
    });

    expect(downloadHandles[0].cancel).toHaveBeenCalledTimes(1);
    expect(useDownloadStore.getState().getCityDownload(1).status).toBe('idle');
    expect(useDownloadStore.getState().getCityDownload(1).progress.totalFiles).toBe(0);
    expect(useDownloadStore.getState().getCityDownload(2).status).toBe('downloading');
    expect(downloadHandles[1].cancel).not.toHaveBeenCalled();
  });

  it('stores errors only for the failing city', async () => {
    mockFetchManifest
      .mockResolvedValueOnce(makeManifest(CITY_TWO_ITEMS))
      .mockRejectedValueOnce(new Error('Manifest failed'));

    await act(async () => {
      renderer = create(
        React.createElement(
          React.Fragment,
          null,
          React.createElement(HookHarness, { cityId: 2, hookId: 'city-2' }),
          React.createElement(HookHarness, { cityId: 3, hookId: 'city-3' }),
        ),
      );
      await flushPromises();
    });

    await startDownload('city-2');
    await act(async () => {
      downloadHandles[0].callback?.({ bytesWritten: 5000, contentLength: 30000 });
      await flushPromises();
    });

    await startDownload('city-3');

    const cityTwo = useDownloadStore.getState().getCityDownload(2);
    const cityThree = useDownloadStore.getState().getCityDownload(3);

    expect(cityThree.status).toBe('error');
    expect(cityThree.error).toBe('Manifest failed');
    expect(cityThree.progress.totalFiles).toBe(0);
    expect(cityTwo.status).toBe('downloading');
    expect(cityTwo.progress.completedBytes).toBe(5000);
  });

  it('marks a city as downloaded when the manifest has no audio files', async () => {
    mockFetchManifest.mockResolvedValueOnce(
      makeManifest([
        {
          story_id: 9,
          poi_id: 9,
          poi_name: 'Silent POI',
          audio_url: null,
          file_size_bytes: 0,
          duration_sec: null,
        },
      ]),
    );

    await act(async () => {
      renderer = create(React.createElement(HookHarness, { cityId: 9, hookId: 'city-9' }));
      await flushPromises();
    });

    await startDownload('city-9');

    const cityNine = useDownloadStore.getState().getCityDownload(9);
    expect(cityNine.status).toBe('completed');
    expect(cityNine.progress.totalFiles).toBe(0);
    expect(useDownloadStore.getState().isCityDownloaded(9)).toBe(true);
    expect(getResult('city-9').isDownloaded).toBe(true);
  });

  it('completion preserves isDownloaded while another city remains in progress', async () => {
    mockFetchManifest
      .mockResolvedValueOnce(makeManifest(CITY_ONE_ITEMS))
      .mockResolvedValueOnce(makeManifest(CITY_TWO_ITEMS));

    await act(async () => {
      renderer = create(
        React.createElement(
          React.Fragment,
          null,
          React.createElement(HookHarness, { cityId: 1, hookId: 'city-1' }),
          React.createElement(HookHarness, { cityId: 2, hookId: 'city-2' }),
        ),
      );
      await flushPromises();
    });

    await startDownload('city-1');
    await startDownload('city-2');
    await resolveDownload(0, 'file:///cache/story_1.mp3');

    expect(useDownloadStore.getState().isCityDownloaded(1)).toBe(true);
    expect(getResult('city-1').isDownloaded).toBe(true);
    expect(useDownloadStore.getState().getCityDownload(1).status).toBe('completed');
    expect(useDownloadStore.getState().getCityDownload(2).status).toBe('downloading');
  });
});
