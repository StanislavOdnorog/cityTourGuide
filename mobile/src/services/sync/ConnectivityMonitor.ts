import NetInfo, { NetInfoState } from '@react-native-community/netinfo';
import type { SyncQueue, SyncQueueItem } from './SyncQueue';

const DEBOUNCE_MS = 3000;

export class ConnectivityMonitor {
  private unsubscribe: (() => void) | null = null;
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;
  private syncQueue: SyncQueue;
  private sendRequest: (item: SyncQueueItem) => Promise<void>;
  private onSyncComplete?: (processed: number, failed: number) => void;

  constructor(
    syncQueue: SyncQueue,
    sendRequest: (item: SyncQueueItem) => Promise<void>,
    onSyncComplete?: (processed: number, failed: number) => void,
  ) {
    this.syncQueue = syncQueue;
    this.sendRequest = sendRequest;
    this.onSyncComplete = onSyncComplete;
  }

  start(): void {
    if (this.unsubscribe) return;

    let wasConnected = true;

    this.unsubscribe = NetInfo.addEventListener((state: NetInfoState) => {
      const isConnected = state.isConnected === true && state.isInternetReachable !== false;

      if (isConnected && !wasConnected) {
        this.debouncedProcess();
      }

      wasConnected = isConnected;
    });
  }

  stop(): void {
    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = null;
    }
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }
  }

  private debouncedProcess(): void {
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }

    this.debounceTimer = setTimeout(async () => {
      this.debounceTimer = null;
      const result = await this.syncQueue.processQueue(this.sendRequest);
      this.onSyncComplete?.(result.processed, result.failed);
    }, DEBOUNCE_MS);
  }
}
