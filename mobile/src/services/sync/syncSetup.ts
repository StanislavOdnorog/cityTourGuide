import apiClient, {
  getAccessToken,
  hasRefreshToken,
  refreshAccessToken,
  setOfflineEnqueue,
} from '@/api/client';
import type { RetryableRequestConfig } from '@/api/retry';
import { useSyncStatus } from '@/hooks/useSyncStatus';
import { ConnectivityMonitor } from './ConnectivityMonitor';
import { SyncQueue } from './SyncQueue';
import type { SyncQueueItem } from './SyncQueue';

let syncQueue: SyncQueue | null = null;
let connectivityMonitor: ConnectivityMonitor | null = null;

async function sendQueuedRequest(item: SyncQueueItem): Promise<void> {
  const headers = JSON.parse(item.headers) as Record<string, string>;
  let body: unknown;
  try {
    body = JSON.parse(item.body);
  } catch {
    body = item.body;
  }

  if (headers.Authorization || headers.authorization) {
    const currentToken = getAccessToken();
    const activeToken = currentToken ?? (hasRefreshToken() ? await refreshAccessToken() : null);

    if (!activeToken) {
      throw new Error('Unable to authenticate queued request.');
    }

    headers.Authorization = `Bearer ${activeToken}`;
    delete headers.authorization;
  }

  await apiClient.request({
    url: item.endpoint,
    method: item.method,
    data: body,
    headers,
    _skipOfflineQueue: true,
  } as RetryableRequestConfig);
}

async function updatePendingCount(): Promise<void> {
  if (!syncQueue) return;
  const count = await syncQueue.getPendingCount();
  useSyncStatus.getState().setPendingCount(count);
}

export async function initSyncQueue(): Promise<void> {
  if (syncQueue) return;

  syncQueue = new SyncQueue();
  await syncQueue.init();

  // Wire up the offline enqueue handler
  setOfflineEnqueue(async (request) => {
    if (!syncQueue) return false;
    const result = await syncQueue.enqueue(request);
    await updatePendingCount();
    return result;
  });

  // Start connectivity monitor
  connectivityMonitor = new ConnectivityMonitor(syncQueue, sendQueuedRequest, async () => {
    await updatePendingCount();
  });
  connectivityMonitor.start();

  // Set initial pending count
  await updatePendingCount();
}

export async function teardownSyncQueue(): Promise<void> {
  setOfflineEnqueue(null);

  if (connectivityMonitor) {
    connectivityMonitor.stop();
    connectivityMonitor = null;
  }

  if (syncQueue) {
    await syncQueue.destroy();
    syncQueue = null;
  }
}

export function getSyncQueue(): SyncQueue | null {
  return syncQueue;
}
