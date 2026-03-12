import { SyncQueue, computeBackoffMs } from '../SyncQueue';
import type { SyncQueueItem, EnqueueRequest } from '../SyncQueue';

// Mock expo-sqlite
const mockRunAsync = jest.fn().mockResolvedValue(undefined);
const mockGetFirstAsync = jest.fn().mockResolvedValue(null);
const mockGetAllAsync = jest.fn().mockResolvedValue([]);
const mockExecAsync = jest.fn().mockResolvedValue(undefined);
const mockCloseAsync = jest.fn().mockResolvedValue(undefined);

jest.mock('expo-sqlite', () => ({
  openDatabaseAsync: jest.fn().mockResolvedValue({
    runAsync: (...args: unknown[]) => mockRunAsync(...args),
    getFirstAsync: (...args: unknown[]) => mockGetFirstAsync(...args),
    getAllAsync: (...args: unknown[]) => mockGetAllAsync(...args),
    execAsync: (...args: unknown[]) => mockExecAsync(...args),
    closeAsync: () => mockCloseAsync(),
  }),
}));

function makeRequest(overrides: Partial<EnqueueRequest> = {}): EnqueueRequest {
  return {
    endpoint: '/api/v1/reports',
    method: 'POST',
    body: { story_id: 1, reason: 'inappropriate' },
    headers: { Authorization: 'Bearer token123' },
    ...overrides,
  };
}

function makeQueueItem(overrides: Partial<SyncQueueItem> = {}): SyncQueueItem {
  return {
    id: 1,
    endpoint: '/api/v1/reports',
    method: 'POST',
    body: JSON.stringify({ story_id: 1, reason: 'inappropriate' }),
    headers: JSON.stringify({ Authorization: 'Bearer token123' }),
    created_at: Date.now(),
    retry_count: 0,
    last_error: null,
    ...overrides,
  };
}

describe('SyncQueue', () => {
  let queue: SyncQueue;

  beforeEach(async () => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    mockGetFirstAsync.mockResolvedValue({ count: 0 });
    queue = new SyncQueue();
    await queue.init();
  });

  afterEach(async () => {
    await queue.destroy();
    jest.useRealTimers();
  });

  describe('init', () => {
    it('creates database and table', () => {
      expect(mockExecAsync).toHaveBeenCalledWith(
        expect.stringContaining('CREATE TABLE IF NOT EXISTS sync_queue'),
      );
    });

    it('is idempotent', async () => {
      const callCount = mockExecAsync.mock.calls.length;
      await queue.init();
      expect(mockExecAsync).toHaveBeenCalledTimes(callCount);
    });
  });

  describe('enqueue', () => {
    it('inserts a request into the queue', async () => {
      const result = await queue.enqueue(makeRequest());

      expect(result).toBe(true);
      expect(mockRunAsync).toHaveBeenCalledWith(
        expect.stringContaining('INSERT INTO sync_queue'),
        '/api/v1/reports',
        'POST',
        expect.any(String),
        expect.any(String),
        expect.any(Number),
      );
    });

    it('rejects when queue is full (500 items)', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ count: 500 });

      const result = await queue.enqueue(makeRequest());
      expect(result).toBe(false);
    });

    it('deduplicates listening events within 1 minute', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ count: 0 });
      mockGetAllAsync.mockResolvedValueOnce([
        {
          body: JSON.stringify({ story_id: 1, user_id: 'user-1' }),
        },
      ]);

      const result = await queue.enqueue(
        makeRequest({
          endpoint: '/api/v1/listenings',
          body: { story_id: 1, user_id: 'user-2', completed: true },
        }),
      );
      expect(result).toBe(false);
    });

    it('allows listening events after 1 minute dedup window', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ count: 0 });
      mockGetAllAsync.mockResolvedValueOnce([]);

      const result = await queue.enqueue(makeRequest({ endpoint: '/api/v1/listenings' }));
      expect(result).toBe(true);
    });

    it('does not deduplicate non-listening endpoints', async () => {
      mockGetFirstAsync.mockResolvedValue({ count: 0 });

      const result = await queue.enqueue(makeRequest({ endpoint: '/api/v1/reports' }));
      expect(result).toBe(true);
      expect(mockGetFirstAsync).toHaveBeenCalledTimes(1);
      expect(mockGetAllAsync).not.toHaveBeenCalled();
    });
  });

  describe('processQueue', () => {
    it('processes items in FIFO order', async () => {
      const items = [
        makeQueueItem({ id: 1, created_at: 1000 }),
        makeQueueItem({ id: 2, created_at: 2000 }),
      ];
      mockGetAllAsync.mockResolvedValueOnce(items);
      const sendRequest = jest.fn().mockResolvedValue(undefined);

      const result = await queue.processQueue(sendRequest);

      expect(sendRequest).toHaveBeenCalledTimes(2);
      expect(sendRequest.mock.calls[0][0].id).toBe(1);
      expect(sendRequest.mock.calls[1][0].id).toBe(2);
      expect(result).toEqual({ processed: 2, failed: 0 });
    });

    it('deletes items after successful send', async () => {
      mockGetAllAsync.mockResolvedValueOnce([makeQueueItem({ id: 42 })]);
      const sendRequest = jest.fn().mockResolvedValue(undefined);

      await queue.processQueue(sendRequest);

      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM sync_queue WHERE id = ?', 42);
    });

    it('increments retry_count on failure', async () => {
      mockGetAllAsync.mockResolvedValueOnce([makeQueueItem({ id: 1, retry_count: 0 })]);
      const sendRequest = jest.fn().mockRejectedValue(new Error('timeout'));

      const processPromise = queue.processQueue(sendRequest);
      await Promise.resolve();
      await jest.advanceTimersByTimeAsync(15000);
      const result = await processPromise;

      expect(mockRunAsync).toHaveBeenCalledWith(
        'UPDATE sync_queue SET retry_count = ?, last_error = ? WHERE id = ?',
        1,
        'timeout',
        1,
      );
      expect(result).toEqual({ processed: 0, failed: 1 });
      expect(sendRequest).toHaveBeenCalledTimes(5);
    });

    it('removes items after 5 failed retries', async () => {
      mockGetAllAsync.mockResolvedValueOnce([makeQueueItem({ id: 1, retry_count: 4 })]);
      const sendRequest = jest.fn().mockRejectedValue(new Error('still failing'));

      const result = await queue.processQueue(sendRequest);

      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM sync_queue WHERE id = ?', 1);
      expect(result).toEqual({ processed: 0, failed: 1 });
    });

    it('retries with exponential backoff before succeeding', async () => {
      mockGetAllAsync.mockResolvedValueOnce([makeQueueItem({ id: 7, retry_count: 0 })]);
      const sendRequest = jest
        .fn()
        .mockRejectedValueOnce(new Error('offline'))
        .mockRejectedValueOnce(new Error('still offline'))
        .mockResolvedValueOnce(undefined);

      const processPromise = queue.processQueue(sendRequest);
      await Promise.resolve();

      expect(sendRequest).toHaveBeenCalledTimes(1);

      await jest.advanceTimersByTimeAsync(999);
      expect(sendRequest).toHaveBeenCalledTimes(1);

      await jest.advanceTimersByTimeAsync(1);
      expect(sendRequest).toHaveBeenCalledTimes(2);

      await jest.advanceTimersByTimeAsync(1999);
      expect(sendRequest).toHaveBeenCalledTimes(2);

      await jest.advanceTimersByTimeAsync(1);

      const result = await processPromise;

      expect(sendRequest).toHaveBeenCalledTimes(3);
      expect(mockRunAsync).toHaveBeenLastCalledWith('DELETE FROM sync_queue WHERE id = ?', 7);
      expect(result).toEqual({ processed: 1, failed: 0 });
    });

    it('does not run concurrently', async () => {
      let resolveFirst: (() => void) | null = null;
      const firstCallPromise = new Promise<void>((resolve) => {
        resolveFirst = resolve;
      });

      mockGetAllAsync.mockResolvedValue([makeQueueItem()]);
      const sendRequest = jest.fn().mockImplementation(() => firstCallPromise);

      const p1 = queue.processQueue(sendRequest);
      // Allow p1 to start (enter processing state)
      await Promise.resolve();
      await Promise.resolve();

      const p2 = queue.processQueue(sendRequest);
      const r2 = await p2;

      // Second call should return early
      expect(r2).toEqual({ processed: 0, failed: 0 });

      // Now finish the first call
      resolveFirst!();
      await p1;
    });
  });

  describe('getPendingCount', () => {
    it('returns the count from database', async () => {
      mockGetFirstAsync.mockResolvedValueOnce({ count: 7 });

      const count = await queue.getPendingCount();
      expect(count).toBe(7);
    });
  });

  describe('clearAll', () => {
    it('deletes all rows', async () => {
      await queue.clearAll();
      expect(mockRunAsync).toHaveBeenCalledWith('DELETE FROM sync_queue');
    });
  });
});

describe('computeBackoffMs', () => {
  it('computes exponential backoff correctly', () => {
    expect(computeBackoffMs(0)).toBe(1000);
    expect(computeBackoffMs(1)).toBe(2000);
    expect(computeBackoffMs(2)).toBe(4000);
    expect(computeBackoffMs(3)).toBe(8000);
    expect(computeBackoffMs(4)).toBe(16000);
  });
});
