import { File, Directory, Paths } from 'expo-file-system';
import { createDownloadResumable, type DownloadResumable } from 'expo-file-system/legacy';
import * as SQLite from 'expo-sqlite';
import { bearing, angleDiff } from '@/services/story-engine/ScoringAlgorithm';
import type { NearbyStoryCandidate } from '@/types';

const DB_NAME = 'story_cache.db';
const CACHE_SUBDIR = 'story_audio';
const DEFAULT_MAX_CACHE_BYTES = 100 * 1024 * 1024; // 100 MB
const PREFETCH_RADIUS_M = 500;
const PREFETCH_DIRECTION_ANGLE = 90; // ±45° cone from heading

export interface CacheStats {
  totalSizeBytes: number;
  cachedFileCount: number;
  maxSizeBytes: number;
}

export interface FileDownloadProgress {
  bytesWritten: number;
  contentLength: number;
}

export interface CancelableDownload {
  promise: Promise<string | null>;
  cancel: () => Promise<void>;
}

export interface CachedStoryMeta {
  storyId: number;
  poiId: number;
  audioUrl: string;
  localPath: string;
  fileSizeBytes: number;
  lastAccessedAt: number;
  cachedAt: number;
}

export class StoryCacheManager {
  private db: SQLite.SQLiteDatabase | null = null;
  private maxCacheBytes: number;
  private initialized = false;
  private prefetchInProgress = new Set<number>();
  private cacheDir: Directory | null = null;

  constructor(maxCacheBytes: number = DEFAULT_MAX_CACHE_BYTES) {
    this.maxCacheBytes = maxCacheBytes;
  }

  async init(): Promise<void> {
    if (this.initialized) return;

    // Ensure cache directory exists
    this.cacheDir = new Directory(Paths.cache, CACHE_SUBDIR);
    if (!this.cacheDir.exists) {
      this.cacheDir.create({ intermediates: true });
    }

    this.db = await SQLite.openDatabaseAsync(DB_NAME);

    await this.db.execAsync(`
      CREATE TABLE IF NOT EXISTS cached_stories (
        story_id INTEGER PRIMARY KEY,
        poi_id INTEGER NOT NULL,
        audio_url TEXT NOT NULL,
        local_path TEXT NOT NULL,
        file_size_bytes INTEGER NOT NULL DEFAULT 0,
        last_accessed_at INTEGER NOT NULL,
        cached_at INTEGER NOT NULL
      );
      CREATE INDEX IF NOT EXISTS idx_cached_last_accessed ON cached_stories(last_accessed_at);
    `);

    this.initialized = true;
  }

  private ensureInit(): void {
    if (!this.initialized || !this.db) {
      throw new Error('StoryCacheManager not initialized. Call init() first.');
    }
  }

  /**
   * Get a local file URI for a story's audio, downloading if needed.
   * Returns the local URI if cached, null if no audio_url.
   */
  async getAudioPath(candidate: NearbyStoryCandidate): Promise<string | null> {
    this.ensureInit();
    if (!candidate.audio_url) return null;

    const existing = await this.getCachedMeta(candidate.story_id);
    if (existing) {
      // Verify file still exists
      const file = new File(existing.localPath);
      if (file.exists) {
        await this.touchEntry(candidate.story_id);
        return existing.localPath;
      }
      // File gone — remove stale entry
      await this.removeDbEntry(candidate.story_id);
    }

    // Download and cache
    return this.downloadAndCache(candidate);
  }

  /**
   * Download audio with byte-level progress reporting and cancellation support.
   * Returns a CancelableDownload with a promise that resolves to the local URI
   * and a cancel() method to abort the download.
   */
  downloadAudioWithProgress(
    candidate: NearbyStoryCandidate,
    onProgress?: (progress: FileDownloadProgress) => void,
  ): CancelableDownload {
    this.ensureInit();

    if (!candidate.audio_url || !this.cacheDir) {
      return { promise: Promise.resolve(null), cancel: async () => {} };
    }

    let downloadResumable: DownloadResumable | null = null;
    let cancelled = false;

    const cancel = async () => {
      cancelled = true;
      if (downloadResumable) {
        try {
          await downloadResumable.cancelAsync();
        } catch {
          // Ignore cancel errors
        }
      }
    };

    const promise = (async (): Promise<string | null> => {
      // Check cache first
      const existing = await this.getCachedMeta(candidate.story_id);
      if (existing) {
        const file = new File(existing.localPath);
        if (file.exists) {
          await this.touchEntry(candidate.story_id);
          return existing.localPath;
        }
        await this.removeDbEntry(candidate.story_id);
      }

      if (cancelled) return null;

      const fileName = this.getFileName(candidate.story_id, candidate.audio_url!);
      const destination = new File(this.cacheDir!, fileName);
      const now = Date.now();

      try {
        downloadResumable = createDownloadResumable(
          candidate.audio_url!,
          destination.uri,
          {},
          (data) => {
            if (!cancelled && onProgress) {
              onProgress({
                bytesWritten: data.totalBytesWritten,
                contentLength: data.totalBytesExpectedToWrite,
              });
            }
          },
        );

        const result = await downloadResumable.downloadAsync();
        downloadResumable = null;

        if (cancelled || !result) {
          try {
            if (destination.exists) destination.delete();
          } catch {
            // Ignore cleanup errors
          }
          return null;
        }

        const downloadedFile = new File(result.uri);
        const fileSize = downloadedFile.size;

        await this.db!.runAsync(
          `INSERT OR REPLACE INTO cached_stories
           (story_id, poi_id, audio_url, local_path, file_size_bytes, last_accessed_at, cached_at)
           VALUES (?, ?, ?, ?, ?, ?, ?)`,
          candidate.story_id,
          candidate.poi_id,
          candidate.audio_url!,
          result.uri,
          fileSize,
          now,
          now,
        );

        await this.evictIfNeeded();
        return result.uri;
      } catch {
        downloadResumable = null;
        // Download failed — clean up partial file
        try {
          if (destination.exists) destination.delete();
        } catch {
          // Ignore cleanup errors
        }
        // Don't insert into DB on failure
        return null;
      }
    })();

    return { promise, cancel };
  }

  /**
   * Check if a story's audio is already cached locally.
   */
  async isCached(storyId: number): Promise<boolean> {
    this.ensureInit();
    const meta = await this.getCachedMeta(storyId);
    if (!meta) return false;

    const file = new File(meta.localPath);
    if (!file.exists) {
      await this.removeDbEntry(storyId);
      return false;
    }
    return true;
  }

  /**
   * Pre-fetch audio for stories that are ahead of the user's path.
   * Only fetches stories within PREFETCH_RADIUS_M and ±45° of heading.
   */
  async prefetchAhead(
    candidates: NearbyStoryCandidate[],
    userLat: number,
    userLng: number,
    heading: number,
  ): Promise<void> {
    this.ensureInit();

    const toPrefetch = candidates.filter((c) => {
      if (!c.audio_url) return false;
      if (c.distance_m > PREFETCH_RADIUS_M) return false;
      if (this.prefetchInProgress.has(c.story_id)) return false;

      // Only prefetch stories ahead (within direction cone)
      if (heading >= 0) {
        // Skip direction filter if candidate is missing real coordinates
        if (c.poi_lat == null || c.poi_lng == null) return true;

        const brng = bearing(userLat, userLng, c.poi_lat, c.poi_lng);
        const diff = angleDiff(heading, brng);
        if (diff > PREFETCH_DIRECTION_ANGLE) return false;
      }

      return true;
    });

    // Prefetch in background (fire and forget)
    for (const candidate of toPrefetch) {
      this.prefetchOne(candidate);
    }
  }

  /**
   * Get current cache statistics.
   */
  async getStats(): Promise<CacheStats> {
    this.ensureInit();
    const db = this.db!;

    const result = await db.getFirstAsync<{ total: number; count: number }>(
      'SELECT COALESCE(SUM(file_size_bytes), 0) as total, COUNT(*) as count FROM cached_stories',
    );

    return {
      totalSizeBytes: result?.total ?? 0,
      cachedFileCount: result?.count ?? 0,
      maxSizeBytes: this.maxCacheBytes,
    };
  }

  /**
   * Remove cached audio files for specific story IDs (e.g. for a single city).
   */
  async clearStories(storyIds: number[]): Promise<void> {
    this.ensureInit();
    if (storyIds.length === 0) return;
    const db = this.db!;

    const rows = await db.getAllAsync<{ story_id: number; local_path: string }>(
      `SELECT story_id, local_path FROM cached_stories WHERE story_id IN (${storyIds.map(() => '?').join(',')})`,
      ...storyIds,
    );

    for (const row of rows) {
      try {
        const file = new File(row.local_path);
        if (file.exists) file.delete();
      } catch {
        // Ignore file deletion errors
      }
    }

    await db.runAsync(
      `DELETE FROM cached_stories WHERE story_id IN (${storyIds.map(() => '?').join(',')})`,
      ...storyIds,
    );
  }

  /**
   * Clear entire cache — delete all files and metadata.
   */
  async clearAll(): Promise<void> {
    this.ensureInit();

    // Delete and recreate cache directory
    if (this.cacheDir && this.cacheDir.exists) {
      this.cacheDir.delete();
      this.cacheDir.create({ intermediates: true });
    }

    // Clear database
    await this.db!.runAsync('DELETE FROM cached_stories');
  }

  /**
   * Run LRU eviction to bring cache under the size limit.
   */
  async evictIfNeeded(): Promise<number> {
    this.ensureInit();
    const db = this.db!;

    const stats = await this.getStats();
    if (stats.totalSizeBytes <= this.maxCacheBytes) return 0;

    let bytesToFree = stats.totalSizeBytes - this.maxCacheBytes;
    let evictedCount = 0;

    // Get least recently accessed entries
    const entries = await db.getAllAsync<{
      story_id: number;
      local_path: string;
      file_size_bytes: number;
    }>(
      'SELECT story_id, local_path, file_size_bytes FROM cached_stories ORDER BY last_accessed_at ASC',
    );

    for (const entry of entries) {
      if (bytesToFree <= 0) break;

      try {
        const file = new File(entry.local_path);
        if (file.exists) {
          file.delete();
        }
      } catch {
        // File may already be deleted
      }
      await db.runAsync('DELETE FROM cached_stories WHERE story_id = ?', entry.story_id);

      bytesToFree -= entry.file_size_bytes;
      evictedCount++;
    }

    return evictedCount;
  }

  /**
   * Reconcile cache: remove stale DB rows for missing files and delete
   * orphaned files on disk that are not referenced in SQLite.
   * Safe to call multiple times (idempotent).
   */
  async reconcile(): Promise<{ removedRows: number; deletedFiles: number }> {
    this.ensureInit();
    const db = this.db!;

    let removedRows = 0;
    let deletedFiles = 0;

    // 1. Remove DB rows whose local_path file no longer exists on disk
    const rows = await db.getAllAsync<{ story_id: number; local_path: string }>(
      'SELECT story_id, local_path FROM cached_stories',
    );

    for (const row of rows) {
      try {
        const file = new File(row.local_path);
        if (!file.exists) {
          await this.removeDbEntry(row.story_id);
          removedRows++;
        }
      } catch {
        // If we can't check, remove the row to be safe
        await this.removeDbEntry(row.story_id);
        removedRows++;
      }
    }

    // 2. Delete orphaned files in the cache directory not referenced in SQLite
    if (this.cacheDir && this.cacheDir.exists) {
      // Re-query DB after cleanup to get current set of referenced paths
      const validRows = await db.getAllAsync<{ local_path: string }>(
        'SELECT local_path FROM cached_stories',
      );
      const referencedPaths = new Set(validRows.map((r) => r.local_path));

      const entries = this.cacheDir.list();
      for (const entry of entries) {
        if (entry instanceof File && !referencedPaths.has(entry.uri)) {
          try {
            entry.delete();
            deletedFiles++;
          } catch {
            // Ignore deletion errors for individual files
          }
        }
      }
    }

    return { removedRows, deletedFiles };
  }

  /**
   * Destroy the cache manager — close database.
   */
  async destroy(): Promise<void> {
    if (this.db) {
      await this.db.closeAsync();
      this.db = null;
    }
    this.initialized = false;
    this.prefetchInProgress.clear();
    this.cacheDir = null;
  }

  // ---------- Private methods ----------

  private async getCachedMeta(storyId: number): Promise<CachedStoryMeta | null> {
    const db = this.db!;
    const row = await db.getFirstAsync<{
      story_id: number;
      poi_id: number;
      audio_url: string;
      local_path: string;
      file_size_bytes: number;
      last_accessed_at: number;
      cached_at: number;
    }>('SELECT * FROM cached_stories WHERE story_id = ?', storyId);

    if (!row) return null;

    return {
      storyId: row.story_id,
      poiId: row.poi_id,
      audioUrl: row.audio_url,
      localPath: row.local_path,
      fileSizeBytes: row.file_size_bytes,
      lastAccessedAt: row.last_accessed_at,
      cachedAt: row.cached_at,
    };
  }

  private async touchEntry(storyId: number): Promise<void> {
    await this.db!.runAsync(
      'UPDATE cached_stories SET last_accessed_at = ? WHERE story_id = ?',
      Date.now(),
      storyId,
    );
  }

  private async removeDbEntry(storyId: number): Promise<void> {
    await this.db!.runAsync('DELETE FROM cached_stories WHERE story_id = ?', storyId);
  }

  private async downloadAndCache(candidate: NearbyStoryCandidate): Promise<string | null> {
    if (!candidate.audio_url || !this.cacheDir) return null;

    const fileName = this.getFileName(candidate.story_id, candidate.audio_url);
    const destination = new File(this.cacheDir, fileName);
    const now = Date.now();

    try {
      const downloadedFile = await File.downloadFileAsync(candidate.audio_url, destination, {
        idempotent: true,
      });

      const fileSize = downloadedFile.size;

      await this.db!.runAsync(
        `INSERT OR REPLACE INTO cached_stories
         (story_id, poi_id, audio_url, local_path, file_size_bytes, last_accessed_at, cached_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
        candidate.story_id,
        candidate.poi_id,
        candidate.audio_url,
        downloadedFile.uri,
        fileSize,
        now,
        now,
      );

      // Run eviction check after adding new file
      await this.evictIfNeeded();

      return downloadedFile.uri;
    } catch {
      // Download failed — clean up partial file
      try {
        if (destination.exists) {
          destination.delete();
        }
      } catch {
        // Ignore cleanup errors
      }
      return null;
    }
  }

  private prefetchOne(candidate: NearbyStoryCandidate): void {
    if (this.prefetchInProgress.has(candidate.story_id)) return;
    this.prefetchInProgress.add(candidate.story_id);

    void this.isCached(candidate.story_id)
      .then((cached) => {
        if (!cached) {
          return this.downloadAndCache(candidate);
        }
        return null;
      })
      .finally(() => {
        this.prefetchInProgress.delete(candidate.story_id);
      });
  }

  private getFileName(storyId: number, audioUrl: string): string {
    const ext = this.getExtension(audioUrl);
    return `story_${storyId}${ext}`;
  }

  private getExtension(url: string): string {
    try {
      const pathname = new URL(url).pathname;
      const dotIndex = pathname.lastIndexOf('.');
      if (dotIndex >= 0) {
        const ext = pathname.substring(dotIndex).toLowerCase();
        if (['.mp3', '.m4a', '.wav', '.ogg', '.aac'].includes(ext)) {
          return ext;
        }
      }
    } catch {
      // Invalid URL — default extension
    }
    return '.mp3';
  }
}
