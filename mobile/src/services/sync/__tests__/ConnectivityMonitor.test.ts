// eslint-disable-next-line import/order
import type { SyncQueue } from '../SyncQueue';
let netInfoCallback:
  | ((state: { isConnected: boolean; isInternetReachable: boolean }) => void)
  | null = null;
const mockUnsubscribe = jest.fn();

jest.mock(
  '@react-native-community/netinfo',
  () => ({
    default: {
      addEventListener: jest.fn((cb: typeof netInfoCallback) => {
        netInfoCallback = cb;
        return mockUnsubscribe;
      }),
    },
    __esModule: true,
  }),
  { virtual: true },
);

// eslint-disable-next-line import/order
import { ConnectivityMonitor } from '../ConnectivityMonitor';

function makeMockQueue(): jest.Mocked<SyncQueue> {
  return {
    init: jest.fn(),
    enqueue: jest.fn(),
    processQueue: jest.fn().mockResolvedValue({ processed: 0, failed: 0 }),
    getPendingCount: jest.fn().mockResolvedValue(0),
    clearCompleted: jest.fn(),
    clearAll: jest.fn(),
    getAll: jest.fn().mockResolvedValue([]),
    destroy: jest.fn(),
  } as unknown as jest.Mocked<SyncQueue>;
}

describe('ConnectivityMonitor', () => {
  let mockQueue: jest.Mocked<SyncQueue>;
  let sendRequest: jest.Mock;
  let onSyncComplete: jest.Mock;
  let monitor: ConnectivityMonitor;

  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
    netInfoCallback = null;

    mockQueue = makeMockQueue();
    sendRequest = jest.fn().mockResolvedValue(undefined);
    onSyncComplete = jest.fn();
    monitor = new ConnectivityMonitor(mockQueue, sendRequest, onSyncComplete);
  });

  afterEach(() => {
    monitor.stop();
    jest.useRealTimers();
  });

  it('subscribes to NetInfo on start', () => {
    monitor.start();
    expect(netInfoCallback).not.toBeNull();
  });

  it('unsubscribes on stop', () => {
    monitor.start();
    monitor.stop();
    expect(mockUnsubscribe).toHaveBeenCalled();
  });

  it('does not double-subscribe', () => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const NetInfo = (
      require('@react-native-community/netinfo') as { default: { addEventListener: jest.Mock } }
    ).default;
    monitor.start();
    monitor.start();
    expect(NetInfo.addEventListener).toHaveBeenCalledTimes(1);
  });

  it('triggers processQueue when connectivity is restored', async () => {
    monitor.start();

    // Simulate going offline
    netInfoCallback!({ isConnected: false, isInternetReachable: false });

    // Simulate going online
    netInfoCallback!({ isConnected: true, isInternetReachable: true });

    // Debounce delay
    jest.advanceTimersByTime(3000);

    // Allow microtasks to flush
    await Promise.resolve();

    expect(mockQueue.processQueue).toHaveBeenCalledWith(sendRequest);
  });

  it('debounces rapid connectivity changes', async () => {
    monitor.start();

    // Simulate going offline then online repeatedly
    netInfoCallback!({ isConnected: false, isInternetReachable: false });
    netInfoCallback!({ isConnected: true, isInternetReachable: true });
    netInfoCallback!({ isConnected: false, isInternetReachable: false });
    netInfoCallback!({ isConnected: true, isInternetReachable: true });

    jest.advanceTimersByTime(3000);
    await Promise.resolve();

    // Should only process once due to debouncing
    expect(mockQueue.processQueue).toHaveBeenCalledTimes(1);
  });

  it('does not trigger when already connected', () => {
    monitor.start();

    // First event with connected=true (initial state was also connected)
    netInfoCallback!({ isConnected: true, isInternetReachable: true });

    jest.advanceTimersByTime(3000);

    expect(mockQueue.processQueue).not.toHaveBeenCalled();
  });

  it('calls onSyncComplete after processing', async () => {
    mockQueue.processQueue.mockResolvedValueOnce({ processed: 3, failed: 1 });
    monitor.start();

    netInfoCallback!({ isConnected: false, isInternetReachable: false });
    netInfoCallback!({ isConnected: true, isInternetReachable: true });

    jest.advanceTimersByTime(3000);

    // Flush the async callback chain (setTimeout -> async processQueue -> onSyncComplete)
    for (let i = 0; i < 10; i++) {
      await Promise.resolve();
    }

    expect(onSyncComplete).toHaveBeenCalledWith(3, 1);
  });
});
