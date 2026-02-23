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
  };

  return {
    File: FileClass,
    Directory: DirectoryClass,
    Paths: {
      cache: { uri: 'file:///cache' },
    },
  };
});

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

function makeCandidate(overrides: Partial<NearbyStoryCandidate> = {}): NearbyStoryCandidate {
  return {
    poi_id: 1,
    poi_name: 'Test POI',
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
  });

  describe('destroy', () => {
    it('closes database', async () => {
      await manager.destroy();
      expect(mockCloseAsync).toHaveBeenCalled();
    });
  });
});
