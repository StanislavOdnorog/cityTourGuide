import { File, Directory, Paths } from 'expo-file-system';
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
        const poiLat = userLat + (c.distance_m / 111320) * Math.cos(heading * (Math.PI / 180));
        const poiLng =
          userLng +
          (c.distance_m / (111320 * Math.cos(userLat * (Math.PI / 180)))) *
            Math.sin(heading * (Math.PI / 180));
        const brng = bearing(userLat, userLng, poiLat, poiLng);
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
