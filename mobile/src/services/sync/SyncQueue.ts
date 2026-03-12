import * as SQLite from 'expo-sqlite';

const DB_NAME = 'sync_queue.db';
const MAX_RETRY_COUNT = 5;
const MAX_QUEUE_SIZE = 500;
const DEDUP_WINDOW_MS = 60_000; // 1 minute

export interface SyncQueueItem {
  id: number;
  endpoint: string;
  method: string;
  body: string;
  headers: string;
  created_at: number;
  retry_count: number;
  last_error: string | null;
}

export interface EnqueueRequest {
  endpoint: string;
  method: string;
  body: unknown;
  headers: Record<string, string>;
}

type ListeningPayload = {
  story_id?: number;
};

export function computeBackoffMs(retryCount: number): number {
  // Exponential backoff: 1s, 2s, 4s, 8s, 16s
  return 1000 * Math.pow(2, retryCount);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export class SyncQueue {
  private db: SQLite.SQLiteDatabase | null = null;
  private initialized = false;
  private processing = false;

  async init(): Promise<void> {
    if (this.initialized) return;

    this.db = await SQLite.openDatabaseAsync(DB_NAME);

    await this.db.execAsync(`
      CREATE TABLE IF NOT EXISTS sync_queue (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        endpoint TEXT NOT NULL,
        method TEXT NOT NULL,
        body TEXT NOT NULL,
        headers TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        retry_count INTEGER NOT NULL DEFAULT 0,
        last_error TEXT
      );
      CREATE INDEX IF NOT EXISTS idx_sync_queue_created ON sync_queue(created_at);
    `);

    this.initialized = true;
  }

  private ensureInit(): void {
    if (!this.initialized || !this.db) {
      throw new Error('SyncQueue not initialized. Call init() first.');
    }
  }

  async enqueue(request: EnqueueRequest): Promise<boolean> {
    this.ensureInit();
    const db = this.db!;

    // Check max queue size
    const countResult = await db.getFirstAsync<{ count: number }>(
      'SELECT COUNT(*) as count FROM sync_queue',
    );
    if ((countResult?.count ?? 0) >= MAX_QUEUE_SIZE) {
      return false;
    }

    // Deduplicate listening events for the same story within 1 minute.
    if (request.endpoint.includes('/listenings')) {
      const body = (request.body ?? {}) as ListeningPayload;
      if (typeof body.story_id === 'number') {
        const cutoff = Date.now() - DEDUP_WINDOW_MS;
        const recentItems = await db.getAllAsync<Pick<SyncQueueItem, 'body'>>(
          'SELECT body FROM sync_queue WHERE endpoint = ? AND created_at > ? ORDER BY created_at DESC',
          request.endpoint,
          cutoff,
        );

        for (const item of recentItems) {
          try {
            const queuedBody = JSON.parse(item.body) as ListeningPayload;
            if (queuedBody.story_id === body.story_id) {
              return false;
            }
          } catch {
            continue;
          }
        }
      }
    }

    await db.runAsync(
      `INSERT INTO sync_queue (endpoint, method, body, headers, created_at, retry_count)
       VALUES (?, ?, ?, ?, ?, 0)`,
      request.endpoint,
      request.method,
      JSON.stringify(request.body),
      JSON.stringify(request.headers),
      Date.now(),
    );

    return true;
  }

  async processQueue(
    sendRequest: (item: SyncQueueItem) => Promise<void>,
  ): Promise<{ processed: number; failed: number }> {
    this.ensureInit();
    if (this.processing) return { processed: 0, failed: 0 };
    this.processing = true;

    let processed = 0;
    let failed = 0;

    try {
      const db = this.db!;
      const items = await db.getAllAsync<SyncQueueItem>(
        'SELECT * FROM sync_queue ORDER BY created_at ASC',
      );

      for (const item of items) {
        let retryCount = item.retry_count;

        while (retryCount < MAX_RETRY_COUNT) {
          try {
            await sendRequest({ ...item, retry_count: retryCount });
            await db.runAsync('DELETE FROM sync_queue WHERE id = ?', item.id);
            processed++;
            break;
          } catch (error) {
            const newRetryCount = retryCount + 1;
            const errorMessage = error instanceof Error ? error.message : String(error);

            if (newRetryCount >= MAX_RETRY_COUNT) {
              await db.runAsync('DELETE FROM sync_queue WHERE id = ?', item.id);
              failed++;
              break;
            }

            await db.runAsync(
              'UPDATE sync_queue SET retry_count = ?, last_error = ? WHERE id = ?',
              newRetryCount,
              errorMessage,
              item.id,
            );

            await sleep(computeBackoffMs(retryCount));
            retryCount = newRetryCount;
          }
        }
      }
    } finally {
      this.processing = false;
    }

    return { processed, failed };
  }

  async getPendingCount(): Promise<number> {
    this.ensureInit();
    const result = await this.db!.getFirstAsync<{ count: number }>(
      'SELECT COUNT(*) as count FROM sync_queue',
    );
    return result?.count ?? 0;
  }

  async clearCompleted(): Promise<void> {
    // All completed items are already deleted during processQueue.
    // This method clears any items that exceeded max retries but
    // are still lingering (defensive cleanup).
    this.ensureInit();
    await this.db!.runAsync('DELETE FROM sync_queue WHERE retry_count >= ?', MAX_RETRY_COUNT);
  }

  async clearAll(): Promise<void> {
    this.ensureInit();
    await this.db!.runAsync('DELETE FROM sync_queue');
  }

  async getAll(): Promise<SyncQueueItem[]> {
    this.ensureInit();
    return this.db!.getAllAsync<SyncQueueItem>('SELECT * FROM sync_queue ORDER BY created_at ASC');
  }

  async destroy(): Promise<void> {
    if (this.db) {
      await this.db.closeAsync();
      this.db = null;
    }
    this.initialized = false;
    this.processing = false;
  }
}
