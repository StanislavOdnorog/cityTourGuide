// Mock expo-file-system
const mockFileExists = jest.fn(() => true);
const mockFileDelete = jest.fn();
const mockFileSize = jest.fn(() => 50000);
const mockFileUri = 'file:///cache/story_audio/story_1.mp3';
const mockDownloadFileAsync = jest.fn().mockResolvedValue({
  uri: mockFileUri,
  exists: true,
  size: 50000,
  delete: mockFileDelete,
});
const mockDirExists = jest.fn(() => false);
const mockDirCreate = jest.fn();
const mockDirDelete = jest.fn();
const mockDirList = jest.fn<unknown[], []>(() => []);

// Mock for createDownloadResumable
const mockCancelAsync = jest.fn().mockResolvedValue(undefined);
let mockDownloadResumableCallback:
  | ((data: { totalBytesWritten: number; totalBytesExpectedToWrite: number }) => void)
  | undefined;
const mockResumableDownloadAsync = jest.fn().mockImplementation(async () => {
  // Simulate progress events
  if (mockDownloadResumableCallback) {
    mockDownloadResumableCallback({ totalBytesWritten: 25000, totalBytesExpectedToWrite: 50000 });
    mockDownloadResumableCallback({ totalBytesWritten: 50000, totalBytesExpectedToWrite: 50000 });
  }
  return { uri: mockFileUri, status: 200, headers: {} };
});
const mockCreateDownloadResumable = jest
  .fn()
  .mockImplementation(
    (
      _url: string,
      _fileUri: string,
      _options: unknown,
      callback?: (data: { totalBytesWritten: number; totalBytesExpectedToWrite: number }) => void,
    ) => {
      mockDownloadResumableCallback = callback;
      return {
        downloadAsync: mockResumableDownloadAsync,
        cancelAsync: mockCancelAsync,
      };
    },
  );

jest.mock('expo-file-system', () => {
  const FileClass = class MockFile {
    uri: string;
    constructor(...uris: unknown[]) {
      this.uri = String(uris.join('/'));
    }
    get exists() {
      return mockFileExists();
    }
    get size() {
      return mockFileSize();
    }
    delete() {
      mockFileDelete();
    }
    static downloadFileAsync = mockDownloadFileAsync;
  };

  const DirectoryClass = class MockDirectory {
    uri: string;
    constructor(...uris: unknown[]) {
      this.uri = String(uris.join('/'));
    }
    get exists() {
      return mockDirExists();
    }
    create = mockDirCreate;
    delete = mockDirDelete;
    list = mockDirList;
  };

  return {
    File: FileClass,
    Directory: DirectoryClass,
    Paths: {
      cache: { uri: 'file:///cache' },
    },
  };
});

jest.mock('expo-file-system/legacy', () => ({
  createDownloadResumable: (...args: unknown[]) => mockCreateDownloadResumable(...args),
}));

// Mock expo-sqlite
const mockRunAsync = jest.fn().mockResolvedValue({ changes: 1 });
const mockGetFirstAsync = jest.fn().mockResolvedValue(null);
const mockGetAllAsync = jest.fn().mockResolvedValue([]);
const mockExecAsync = jest.fn().mockResolvedValue(undefined);
const mockCloseAsync = jest.fn().mockResolvedValue(undefined);

jest.mock('expo-sqlite', () => ({
  openDatabaseAsync: jest.fn().mockResolvedValue({
    runAsync: mockRunAsync,
    getFirstAsync: mockGetFirstAsync,
    getAllAsync: mockGetAllAsync,
    execAsync: mockExecAsync,
    closeAsync: mockCloseAsync,
  }),
}));

import type { NearbyStoryCandidate } from '@/types';
import { StoryCacheManager } from '../StoryCacheManager';
import type { FileDownloadProgress } from '../StoryCacheManager';

function makeCandidate(overrides: Partial<NearbyStoryCandidate> = {}): NearbyStoryCandidate {
  return {
    poi_id: 1,
    poi_name: 'Test POI',
    poi_lat: 41.716,
    poi_lng: 44.828,
    story_id: 1,
    story_text: 'A story',
    audio_url: 'https://example.com/audio.mp3',
    duration_sec: 30,
    distance_m: 100,
    score: 50,
    ...overrides,
  };
}

describe('StoryCacheManager', () => {
  let manager: StoryCacheManager;

  beforeEach(async () => {
    jest.clearAllMocks();
    mockDirExists.mockReturnValue(false);
    mockFileExists.mockReturnValue(true);
    mockGetFirstAsync.mockResolvedValue(null);
    mockGetAllAsync.mockResolvedValue([]);

    manager = new StoryCacheManager();
    await manager.init();
  });

  afterEach(async () => {
    await manager.destroy();
  });

  describe('init', () => {
    it('creates cache directory if it does not exist', async () => {
      expect(mockDirCreate).toHaveBeenCalled();
    });

    it('creates SQLite table', async () => {
      expect(mockExecAsync).toHaveBeenCalledWith(
        expect.stringContaining('CREATE TABLE IF NOT EXISTS cached_stories'),
      );
    });

    it('is idempotent', async () => {
      const callCount = mockExecAsync.mock.calls.length;
      await manager.init();
      expect(mockExecAsync.mock.calls.length).toBe(callCount);
    });
  });

  describe('getAudioPath', () => {
    it('returns null for candidates without audio_url', async () => {
      const candidate = makeCandidate({ audio_url: null });
      const result = await manager.getAudioPath(candidate);
      expect(result).toBeNull();
    });

    it('downloads and caches new audio', async () => {
      const candidate = makeCandidate();
      const result = await manager.getAudioPath(candidate);
      expect(mockDownloadFileAsync).toHaveBeenCalledWith(
        'https://example.com/audio.mp3',
        expect.anything(),
        { idempotent: true },
      );
      expect(result).toBe(mockFileUri);
    });

    it('returns cached path for existing entry', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({
        story_id: 1,
        poi_id: 1,
        audio_url: 'https://example.com/audio.mp3',
        local_path: 'file:///cache/story_audio/story_1.mp3',
        file_size_bytes: 50000,
        last_accessed_at: Date.now(),
        cached_at: Date.now(),
      });

      const candidate = makeCandidate();
      const result = await manager.getAudioPath(candidate);
      expect(result).toBe('file:///cache/story_audio/story_1.mp3');
      expect(mockDownloadFileAsync).not.toHaveBeenCalled();
    });

    it('re-downloads when cached file no longer exists', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({
        story_id: 1,
        poi_id: 1,
        audio_url: 'https://example.com/audio.mp3',
        local_path: 'file:///cache/story_audio/story_1.mp3',
        file_size_bytes: 50000,
        last_accessed_at: Date.now(),
        cached_at: Date.now(),
      });
      mockFileExists.mockReturnValueOnce(false);

      const candidate = makeCandidate();
      const result = await manager.getAudioPath(candidate);
      expect(mockDownloadFileAsync).toHaveBeenCalled();
      expect(result).toBe(mockFileUri);
    });
  });

  describe('isCached', () => {
    it('returns false when no entry in database', async () => {
      const result = await manager.isCached(1);
      expect(result).toBe(false);
    });

    it('returns true when entry and file exist', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({
        story_id: 1,
        poi_id: 1,
        audio_url: 'https://example.com/audio.mp3',
        local_path: 'file:///cache/story_audio/story_1.mp3',
        file_size_bytes: 50000,
        last_accessed_at: Date.now(),
        cached_at: Date.now(),
      });

      const result = await manager.isCached(1);
      expect(result).toBe(true);
    });

    it('returns false and cleans up when file is missing', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({
        story_id: 1,
        poi_id: 1,
        audio_url: 'https://example.com/audio.mp3',
        local_path: 'file:///cache/story_audio/story_1.mp3',
        file_size_bytes: 50000,
        last_accessed_at: Date.now(),
        cached_at: Date.now(),
      });
      mockFileExists.mockReturnValueOnce(false);

      const result = await manager.isCached(1);
      expect(result).toBe(false);
      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM cached_stories WHERE story_id = ?', 1);
    });
  });

  describe('getStats', () => {
    it('returns zeros for empty cache', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ total: 0, count: 0 });
      const stats = await manager.getStats();
      expect(stats.totalSizeBytes).toBe(0);
      expect(stats.cachedFileCount).toBe(0);
      expect(stats.maxSizeBytes).toBe(100 * 1024 * 1024);
    });

    it('returns actual stats from database', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ total: 5000000, count: 10 });
      const stats = await manager.getStats();
      expect(stats.totalSizeBytes).toBe(5000000);
      expect(stats.cachedFileCount).toBe(10);
    });
  });

  describe('clearAll', () => {
    it('deletes cache directory and clears database', async () => {
      mockDirExists.mockReturnValue(true);
      await manager.clearAll();
      expect(mockDirDelete).toHaveBeenCalled();
      expect(mockDirCreate).toHaveBeenCalled();
      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM cached_stories');
    });
  });

  describe('evictIfNeeded', () => {
    it('does nothing when cache is under limit', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ total: 1000, count: 1 });
      const evicted = await manager.evictIfNeeded();
      expect(evicted).toBe(0);
    });

    it('evicts LRU entries when cache exceeds limit', async () => {
      const smallLimit = new StoryCacheManager(100);
      await smallLimit.init();

      mockGetFirstAsync.mockResolvedValueOnce({ total: 200, count: 2 });
      mockGetAllAsync.mockResolvedValueOnce([
        { story_id: 1, local_path: 'file:///cache/story_1.mp3', file_size_bytes: 120 },
        { story_id: 2, local_path: 'file:///cache/story_2.mp3', file_size_bytes: 80 },
      ]);

      const evicted = await smallLimit.evictIfNeeded();
      expect(evicted).toBe(1); // First entry (120 bytes) is enough to bring under 100
      expect(mockFileDelete).toHaveBeenCalled();
      await smallLimit.destroy();
    });
  });

  describe('prefetchAhead', () => {
    it('prefetches stories within radius and direction cone', async () => {
      const candidates = [
        makeCandidate({ story_id: 1, distance_m: 100 }),
        makeCandidate({ story_id: 2, distance_m: 200 }),
      ];

      // Mock isCached to return false so download happens
      mockGetFirstAsync.mockResolvedValue(null);

      await manager.prefetchAhead(candidates, 41.7, 44.8, 0);

      // Give time for async prefetch
      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockDownloadFileAsync).toHaveBeenCalled();
    });

    it('skips stories beyond prefetch radius', async () => {
      const candidates = [makeCandidate({ story_id: 1, distance_m: 600 })];

      await manager.prefetchAhead(candidates, 41.7, 44.8, 0);

      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockDownloadFileAsync).not.toHaveBeenCalled();
    });

    it('skips stories without audio_url', async () => {
      const candidates = [makeCandidate({ story_id: 1, audio_url: null, distance_m: 100 })];

      await manager.prefetchAhead(candidates, 41.7, 44.8, 0);

      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockDownloadFileAsync).not.toHaveBeenCalled();
    });

    it('uses real POI coordinates for direction filtering', async () => {
      // User at (41.7, 44.8), heading east (90°).
      // POI to the west (behind) — should be skipped.
      const behindCandidate = makeCandidate({
        story_id: 3,
        distance_m: 100,
        poi_lat: 41.7,
        poi_lng: 44.79, // west of user
      });

      mockGetFirstAsync.mockResolvedValue(null);

      await manager.prefetchAhead([behindCandidate], 41.7, 44.8, 90);
      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockDownloadFileAsync).not.toHaveBeenCalled();
    });

    it('prefetches stories ahead using real POI coordinates', async () => {
      // User at (41.7, 44.8), heading east (90°).
      // POI to the east (ahead) — should be prefetched.
      const aheadCandidate = makeCandidate({
        story_id: 4,
        distance_m: 100,
        poi_lat: 41.7,
        poi_lng: 44.81, // east of user
      });

      mockGetFirstAsync.mockResolvedValue(null);

      await manager.prefetchAhead([aheadCandidate], 41.7, 44.8, 90);
      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockDownloadFileAsync).toHaveBeenCalled();
    });

    it('includes candidates with missing coordinates (graceful fallback)', async () => {
      // When poi_lat/poi_lng are null/undefined, direction filter is skipped
      // and the candidate is included.
      const noCoordCandidate = makeCandidate({
        story_id: 5,
        distance_m: 100,
        poi_lat: null as unknown as number,
        poi_lng: null as unknown as number,
      });

      mockGetFirstAsync.mockResolvedValue(null);

      await manager.prefetchAhead([noCoordCandidate], 41.7, 44.8, 90);
      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockDownloadFileAsync).toHaveBeenCalled();
    });
  });

  describe('downloadAudioWithProgress', () => {
    it('returns null immediately for candidates without audio_url', async () => {
      const candidate = makeCandidate({ audio_url: null });
      const handle = manager.downloadAudioWithProgress(candidate);
      const result = await handle.promise;
      expect(result).toBeNull();
    });

    it('returns cached path without downloading', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({
        story_id: 1,
        poi_id: 1,
        audio_url: 'https://example.com/audio.mp3',
        local_path: 'file:///cache/story_audio/story_1.mp3',
        file_size_bytes: 50000,
        last_accessed_at: Date.now(),
        cached_at: Date.now(),
      });

      const candidate = makeCandidate();
      const handle = manager.downloadAudioWithProgress(candidate);
      const result = await handle.promise;
      expect(result).toBe('file:///cache/story_audio/story_1.mp3');
      expect(mockCreateDownloadResumable).not.toHaveBeenCalled();
    });

    it('downloads with progress events', async () => {
      const progressEvents: FileDownloadProgress[] = [];
      const candidate = makeCandidate();

      const handle = manager.downloadAudioWithProgress(candidate, (p) => {
        progressEvents.push({ ...p });
      });

      const result = await handle.promise;
      expect(result).toBe(mockFileUri);
      expect(mockCreateDownloadResumable).toHaveBeenCalledWith(
        'https://example.com/audio.mp3',
        expect.stringContaining('story_1.mp3'),
        {},
        expect.any(Function),
      );
      expect(progressEvents.length).toBeGreaterThan(0);
      expect(progressEvents[0].bytesWritten).toBe(25000);
      expect(progressEvents[1].bytesWritten).toBe(50000);
    });

    it('inserts cache entry after successful download', async () => {
      const candidate = makeCandidate();
      const handle = manager.downloadAudioWithProgress(candidate);
      await handle.promise;

      expect(mockRunAsync).toHaveBeenCalledWith(
        expect.stringContaining('INSERT OR REPLACE INTO cached_stories'),
        1, // story_id
        1, // poi_id
        'https://example.com/audio.mp3',
        mockFileUri,
        50000, // file size
        expect.any(Number),
        expect.any(Number),
      );
    });

    it('cancellation returns null and does not insert cache entry', async () => {
      // Make downloadAsync hang until cancel
      let resolveDownload: (v: unknown) => void;
      mockResumableDownloadAsync.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveDownload = resolve;
          }),
      );

      const candidate = makeCandidate();
      const handle = manager.downloadAudioWithProgress(candidate);

      // Wait for the async IIFE to progress past getCachedMeta to downloadAsync
      await new Promise((r) => setTimeout(r, 0));

      // Cancel
      await handle.cancel();

      // Resolve the download (simulating the request finishing after cancel)
      resolveDownload!({ uri: mockFileUri, status: 200, headers: {} });

      const result = await handle.promise;
      expect(result).toBeNull();
      // Should NOT have inserted into DB
      expect(mockRunAsync).not.toHaveBeenCalledWith(
        expect.stringContaining('INSERT OR REPLACE'),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
      );
    });

    it('suppresses progress callbacks after cancellation', async () => {
      const progressEvents: FileDownloadProgress[] = [];
      let capturedCallback:
        | ((data: { totalBytesWritten: number; totalBytesExpectedToWrite: number }) => void)
        | undefined;

      mockCreateDownloadResumable.mockImplementationOnce(
        (
          _url: string,
          _fileUri: string,
          _options: unknown,
          callback?: (data: {
            totalBytesWritten: number;
            totalBytesExpectedToWrite: number;
          }) => void,
        ) => {
          capturedCallback = callback;
          return {
            downloadAsync: () => new Promise(() => {}), // never resolves
            cancelAsync: mockCancelAsync,
          };
        },
      );

      const candidate = makeCandidate();
      const handle = manager.downloadAudioWithProgress(candidate, (p) => {
        progressEvents.push({ ...p });
      });

      // Wait for the async IIFE to progress past getCachedMeta to createDownloadResumable
      await new Promise((r) => setTimeout(r, 0));

      // Fire a progress event before cancel
      capturedCallback!({ totalBytesWritten: 10000, totalBytesExpectedToWrite: 50000 });
      expect(progressEvents).toHaveLength(1);

      // Cancel
      await handle.cancel();

      // Fire another progress event after cancel — should be suppressed
      capturedCallback!({ totalBytesWritten: 30000, totalBytesExpectedToWrite: 50000 });
      expect(progressEvents).toHaveLength(1);
    });

    it('cleans up partial file on download failure', async () => {
      mockResumableDownloadAsync.mockRejectedValueOnce(new Error('Network error'));

      const candidate = makeCandidate();
      const handle = manager.downloadAudioWithProgress(candidate);
      const result = await handle.promise;

      expect(result).toBeNull();
      expect(mockFileDelete).toHaveBeenCalled();
      // Should NOT have inserted into DB
      expect(mockRunAsync).not.toHaveBeenCalledWith(
        expect.stringContaining('INSERT OR REPLACE'),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
        expect.anything(),
      );
    });

    it('cleans up partial file when downloadAsync returns undefined (cancelled)', async () => {
      mockResumableDownloadAsync.mockResolvedValueOnce(undefined);

      const candidate = makeCandidate();
      const handle = manager.downloadAudioWithProgress(candidate);
      const result = await handle.promise;

      expect(result).toBeNull();
    });
  });

  describe('reconcile', () => {
    it('removes DB rows for files that no longer exist on disk', async () => {
      // DB has two entries, but both files are missing
      mockGetAllAsync
        .mockResolvedValueOnce([
          { story_id: 1, local_path: 'file:///cache/story_audio/story_1.mp3' },
          { story_id: 2, local_path: 'file:///cache/story_audio/story_2.mp3' },
        ])
        // Second call for orphan check returns empty (rows were removed)
        .mockResolvedValueOnce([]);
      mockFileExists.mockReturnValue(false);
      mockDirExists.mockReturnValue(true);
      mockDirList.mockReturnValue([]);

      const result = await manager.reconcile();

      expect(result.removedRows).toBe(2);
      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM cached_stories WHERE story_id = ?', 1);
      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM cached_stories WHERE story_id = ?', 2);
    });

    it('keeps DB rows whose files still exist', async () => {
      mockGetAllAsync
        .mockResolvedValueOnce([
          { story_id: 1, local_path: 'file:///cache/story_audio/story_1.mp3' },
        ])
        .mockResolvedValueOnce([{ local_path: 'file:///cache/story_audio/story_1.mp3' }]);
      mockFileExists.mockReturnValue(true);
      mockDirExists.mockReturnValue(true);
      mockDirList.mockReturnValue([]);

      const result = await manager.reconcile();

      expect(result.removedRows).toBe(0);
    });

    it('deletes orphaned files not referenced in SQLite', async () => {
      // No stale DB rows
      mockGetAllAsync
        .mockResolvedValueOnce([])
        // After cleanup, DB has one valid entry
        .mockResolvedValueOnce([{ local_path: 'file:///cache/story_audio/story_1.mp3' }]);
      mockFileExists.mockReturnValue(true);
      mockDirExists.mockReturnValue(true);

      // Directory contains two files: one referenced, one orphaned
      // File objects below used via MockFile instances
      const _referencedFile = {
        uri: 'file:///cache/story_audio/story_1.mp3',
        exists: true,
        delete: jest.fn(),
      };
      const _orphanedFile = {
        uri: 'file:///cache/story_audio/story_99.mp3',
        exists: true,
        delete: jest.fn(),
      };

      // Need the orphaned file to be instanceof File — use the mock FileClass
      const { File: MockFile } = jest.requireMock('expo-file-system');
      const refFileInstance = new MockFile('file:///cache/story_audio/story_1.mp3');
      const orphanFileInstance = new MockFile('file:///cache/story_audio/story_99.mp3');

      mockDirList.mockReturnValue([refFileInstance, orphanFileInstance]);

      // mockFileDelete is called when orphanFileInstance.delete() is invoked
      // mockFileExists should return true for the DB row check
      const result = await manager.reconcile();

      expect(result.deletedFiles).toBe(1);
      // The referenced file should NOT be deleted — only the orphan.
      // mockFileDelete is shared, so we check it was called once (for the orphan)
      expect(mockFileDelete).toHaveBeenCalledTimes(1);
    });

    it('is idempotent — running twice produces the same result', async () => {
      // First run: one stale row, no orphans
      mockGetAllAsync
        .mockResolvedValueOnce([
          { story_id: 5, local_path: 'file:///cache/story_audio/story_5.mp3' },
        ])
        .mockResolvedValueOnce([]);
      mockFileExists.mockReturnValue(false);
      mockDirExists.mockReturnValue(true);
      mockDirList.mockReturnValue([]);

      const first = await manager.reconcile();
      expect(first.removedRows).toBe(1);

      // Second run: DB is now empty, no orphans
      mockGetAllAsync.mockResolvedValueOnce([]).mockResolvedValueOnce([]);
      mockDirList.mockReturnValue([]);

      const second = await manager.reconcile();
      expect(second.removedRows).toBe(0);
      expect(second.deletedFiles).toBe(0);
    });
  });

  describe('destroy', () => {
    it('closes database', async () => {
      await manager.destroy();
      expect(mockCloseAsync).toHaveBeenCalled();
    });
  });
});
