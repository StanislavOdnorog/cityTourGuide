import { LocationTracker } from '../LocationTracker';
import type { LocationUpdate } from '../LocationTracker';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const mockWatchPositionAsync = jest.fn();
const mockStartLocationUpdatesAsync = jest.fn();
const mockStopLocationUpdatesAsync = jest.fn();
const mockRequestForegroundPermissionsAsync = jest.fn();
const mockRequestBackgroundPermissionsAsync = jest.fn();
const mockGetForegroundPermissionsAsync = jest.fn();
const mockGetBackgroundPermissionsAsync = jest.fn();

jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: (...args: unknown[]) =>
    mockRequestForegroundPermissionsAsync(...args),
  requestBackgroundPermissionsAsync: (...args: unknown[]) =>
    mockRequestBackgroundPermissionsAsync(...args),
  getForegroundPermissionsAsync: (...args: unknown[]) => mockGetForegroundPermissionsAsync(...args),
  getBackgroundPermissionsAsync: (...args: unknown[]) => mockGetBackgroundPermissionsAsync(...args),
  watchPositionAsync: (...args: unknown[]) => mockWatchPositionAsync(...args),
  startLocationUpdatesAsync: (...args: unknown[]) => mockStartLocationUpdatesAsync(...args),
  stopLocationUpdatesAsync: (...args: unknown[]) => mockStopLocationUpdatesAsync(...args),
  PermissionStatus: { GRANTED: 'granted', DENIED: 'denied', UNDETERMINED: 'undetermined' },
  Accuracy: { High: 6, Balanced: 3 },
  ActivityType: { Fitness: 1 },
}));

const mockIsTaskRegisteredAsync = jest.fn();

jest.mock('expo-task-manager', () => ({
  defineTask: jest.fn(),
  isTaskRegisteredAsync: (...args: unknown[]) => mockIsTaskRegisteredAsync(...args),
}));

// ─── Helpers ──────────────────────────────────────────────────────────────────

function makeLocationObject(
  overrides: {
    latitude?: number;
    longitude?: number;
    speed?: number;
    heading?: number;
    timestamp?: number;
  } = {},
) {
  return {
    coords: {
      latitude: overrides.latitude ?? 41.7151,
      longitude: overrides.longitude ?? 44.8271,
      altitude: 400,
      accuracy: 10,
      altitudeAccuracy: 5,
      speed: overrides.speed ?? 1.2,
      heading: overrides.heading ?? 90,
    },
    timestamp: overrides.timestamp ?? Date.now(),
  };
}

function grantAllPermissions() {
  mockGetForegroundPermissionsAsync.mockResolvedValue({ status: 'granted' });
  mockGetBackgroundPermissionsAsync.mockResolvedValue({ status: 'granted' });
}

function createStartedTracker(config?: Partial<ConstructorParameters<typeof LocationTracker>[0]>) {
  const tracker = new LocationTracker(config);
  grantAllPermissions();
  mockWatchPositionAsync.mockResolvedValue({ remove: jest.fn() });
  mockIsTaskRegisteredAsync.mockResolvedValue(false);
  mockStartLocationUpdatesAsync.mockResolvedValue(undefined);
  return tracker;
}

// ─── Tests ────────────────────────────────────────────────────────────────────

beforeEach(() => {
  jest.clearAllMocks();
  jest.restoreAllMocks();
});

describe('LocationTracker', () => {
  describe('constructor', () => {
    it('uses default config values', () => {
      const tracker = new LocationTracker();
      expect(tracker.getState()).toBe('idle');
      expect(tracker.isTracking()).toBe(false);
      expect(tracker.isSleeping()).toBe(false);
      expect(tracker.getLastLocation()).toBeNull();
    });

    it('accepts custom config overrides', () => {
      const tracker = new LocationTracker({ activeIntervalMs: 3000, sleepThresholdMs: 60_000 });
      expect(tracker.getState()).toBe('idle');
    });
  });

  describe('requestPermissions', () => {
    it('returns true when both permissions granted', async () => {
      mockRequestForegroundPermissionsAsync.mockResolvedValue({ status: 'granted' });
      mockRequestBackgroundPermissionsAsync.mockResolvedValue({ status: 'granted' });

      const tracker = new LocationTracker();
      const result = await tracker.requestPermissions();

      expect(result).toBe(true);
      expect(mockRequestForegroundPermissionsAsync).toHaveBeenCalledTimes(1);
      expect(mockRequestBackgroundPermissionsAsync).toHaveBeenCalledTimes(1);
    });

    it('returns false when foreground permission denied', async () => {
      mockRequestForegroundPermissionsAsync.mockResolvedValue({ status: 'denied' });

      const tracker = new LocationTracker();
      const result = await tracker.requestPermissions();

      expect(result).toBe(false);
      expect(mockRequestBackgroundPermissionsAsync).not.toHaveBeenCalled();
    });

    it('returns false when background permission denied', async () => {
      mockRequestForegroundPermissionsAsync.mockResolvedValue({ status: 'granted' });
      mockRequestBackgroundPermissionsAsync.mockResolvedValue({ status: 'denied' });

      const tracker = new LocationTracker();
      const result = await tracker.requestPermissions();

      expect(result).toBe(false);
    });
  });

  describe('hasPermissions', () => {
    it('returns true when both permissions granted', async () => {
      grantAllPermissions();
      const tracker = new LocationTracker();
      expect(await tracker.hasPermissions()).toBe(true);
    });

    it('returns false when foreground not granted', async () => {
      mockGetForegroundPermissionsAsync.mockResolvedValue({ status: 'denied' });
      const tracker = new LocationTracker();
      expect(await tracker.hasPermissions()).toBe(false);
    });

    it('returns false when background not granted', async () => {
      mockGetForegroundPermissionsAsync.mockResolvedValue({ status: 'granted' });
      mockGetBackgroundPermissionsAsync.mockResolvedValue({ status: 'denied' });
      const tracker = new LocationTracker();
      expect(await tracker.hasPermissions()).toBe(false);
    });
  });

  describe('start', () => {
    it('starts foreground and background tracking', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      expect(tracker.getState()).toBe('active');
      expect(tracker.isTracking()).toBe(true);
      expect(mockWatchPositionAsync).toHaveBeenCalledTimes(1);
      expect(mockStartLocationUpdatesAsync).toHaveBeenCalledTimes(1);
    });

    it('throws if permissions not granted', async () => {
      mockGetForegroundPermissionsAsync.mockResolvedValue({ status: 'denied' });
      const tracker = new LocationTracker();

      await expect(tracker.start()).rejects.toThrow('Location permissions not granted');
      expect(tracker.getState()).toBe('idle');
    });

    it('is a no-op if already tracking', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();
      await tracker.start(); // second call

      expect(mockWatchPositionAsync).toHaveBeenCalledTimes(1);
    });

    it('skips background registration if task already registered', async () => {
      const tracker = await createStartedTracker();
      mockIsTaskRegisteredAsync.mockResolvedValue(true);
      await tracker.start();

      expect(mockStartLocationUpdatesAsync).not.toHaveBeenCalled();
    });

    it('passes correct options to watchPositionAsync', async () => {
      const tracker = await createStartedTracker({ activeIntervalMs: 3000, distanceFilterM: 5 });
      await tracker.start();

      const options = mockWatchPositionAsync.mock.calls[0][0];
      expect(options.timeInterval).toBe(3000);
      expect(options.distanceInterval).toBe(5);
    });
  });

  describe('stop', () => {
    it('stops all tracking and resets state', async () => {
      const tracker = await createStartedTracker();
      const removeMock = jest.fn();
      mockWatchPositionAsync.mockResolvedValue({ remove: removeMock });
      mockIsTaskRegisteredAsync.mockResolvedValue(true);

      await tracker.start();
      await tracker.stop();

      expect(tracker.getState()).toBe('idle');
      expect(tracker.isTracking()).toBe(false);
      expect(removeMock).toHaveBeenCalledTimes(1);
      expect(mockStopLocationUpdatesAsync).toHaveBeenCalledTimes(1);
    });

    it('is a no-op if idle', async () => {
      const tracker = new LocationTracker();
      await tracker.stop();
      expect(tracker.getState()).toBe('idle');
      expect(mockStopLocationUpdatesAsync).not.toHaveBeenCalled();
    });

    it('skips stopping background task if not registered', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();
      mockIsTaskRegisteredAsync.mockResolvedValue(false);
      await tracker.stop();

      expect(mockStopLocationUpdatesAsync).not.toHaveBeenCalled();
    });

    it('clears last location on stop', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      tracker.processLocationObject(makeLocationObject());
      expect(tracker.getLastLocation()).not.toBeNull();

      mockIsTaskRegisteredAsync.mockResolvedValue(false);
      await tracker.stop();
      expect(tracker.getLastLocation()).toBeNull();
    });
  });

  describe('processLocationObject', () => {
    it('converts expo LocationObject to LocationUpdate', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      const locationObj = makeLocationObject({
        latitude: 41.72,
        longitude: 44.83,
        speed: 1.5,
        heading: 180,
        timestamp: 1000000,
      });
      tracker.processLocationObject(locationObj);

      expect(callback).toHaveBeenCalledTimes(1);
      const update: LocationUpdate = callback.mock.calls[0][0];
      expect(update.lat).toBe(41.72);
      expect(update.lng).toBe(44.83);
      expect(update.speed).toBe(1.5);
      expect(update.heading).toBe(180);
      expect(update.timestamp).toBe(1000000);
    });

    it('clamps negative speed to 0', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      tracker.processLocationObject(makeLocationObject({ speed: -1 }));
      expect(callback.mock.calls[0][0].speed).toBe(0);
    });

    it('uses -1 for null heading', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      const locationObj = makeLocationObject();
      locationObj.coords.heading = null as unknown as number;
      tracker.processLocationObject(locationObj);

      expect(callback.mock.calls[0][0].heading).toBe(-1);
    });

    it('stores last location', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();
      tracker.setCallback(jest.fn());

      tracker.processLocationObject(makeLocationObject({ latitude: 42.0 }));
      expect(tracker.getLastLocation()?.lat).toBe(42.0);
    });

    it('does not invoke callback if no callback set', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      // Should not throw
      tracker.processLocationObject(makeLocationObject());
      expect(tracker.getLastLocation()).not.toBeNull();
    });
  });

  describe('adaptive mode: active → sleeping', () => {
    it('stays active when speed is above moving threshold', async () => {
      const tracker = await createStartedTracker({ sleepThresholdMs: 100 });
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      tracker.processLocationObject(makeLocationObject({ speed: 1.0 }));
      expect(tracker.getState()).toBe('active');
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it('enters sleep mode after stationary for sleepThresholdMs', async () => {
      const tracker = await createStartedTracker({ sleepThresholdMs: 100 });
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      // First update: moving — this sets lastMovementTimestamp
      tracker.processLocationObject(makeLocationObject({ speed: 1.0, timestamp: 1000 }));
      expect(tracker.getState()).toBe('active');
      expect(callback).toHaveBeenCalledTimes(1);

      // Advance time past sleep threshold
      jest.spyOn(Date, 'now').mockReturnValue(Date.now() + 200);

      // Second update: stopped
      tracker.processLocationObject(makeLocationObject({ speed: 0.1, timestamp: 2000 }));
      expect(tracker.getState()).toBe('sleeping');
      // Callback should NOT be called while sleeping
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it('does not enter sleep if stopped briefly', async () => {
      const tracker = await createStartedTracker({ sleepThresholdMs: 120_000 });
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      // Moving
      tracker.processLocationObject(makeLocationObject({ speed: 1.0 }));
      expect(tracker.getState()).toBe('active');

      // Stopped briefly (less than sleep threshold)
      tracker.processLocationObject(makeLocationObject({ speed: 0.1 }));
      expect(tracker.getState()).toBe('active');
      expect(callback).toHaveBeenCalledTimes(2);
    });
  });

  describe('adaptive mode: sleeping → active (wake-up)', () => {
    async function createSleepingTracker() {
      const tracker = await createStartedTracker({ sleepThresholdMs: 100 });
      await tracker.start();

      const callback = jest.fn();
      tracker.setCallback(callback);

      // Move to set lastMovementTimestamp
      tracker.processLocationObject(makeLocationObject({ speed: 1.0 }));

      // Advance time past sleep threshold
      jest.spyOn(Date, 'now').mockReturnValue(Date.now() + 200);

      // Go stationary → triggers sleep
      tracker.processLocationObject(makeLocationObject({ speed: 0.1 }));
      expect(tracker.getState()).toBe('sleeping');

      callback.mockClear();
      return { tracker, callback };
    }

    it('wakes up on movement above threshold', async () => {
      const { tracker, callback } = await createSleepingTracker();

      // Movement detected
      tracker.processLocationObject(makeLocationObject({ speed: 0.8 }));
      expect(tracker.getState()).toBe('active');
      // Wake-up should emit the location
      expect(callback).toHaveBeenCalledTimes(1);
    });

    it('stays sleeping if speed is below moving threshold', async () => {
      const { tracker, callback } = await createSleepingTracker();

      tracker.processLocationObject(makeLocationObject({ speed: 0.2 }));
      expect(tracker.getState()).toBe('sleeping');
      expect(callback).not.toHaveBeenCalled();
    });

    it('does not emit callback while sleeping', async () => {
      const { tracker, callback } = await createSleepingTracker();

      tracker.processLocationObject(makeLocationObject({ speed: 0.1 }));
      tracker.processLocationObject(makeLocationObject({ speed: 0.2 }));
      tracker.processLocationObject(makeLocationObject({ speed: 0.15 }));

      expect(tracker.getState()).toBe('sleeping');
      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('speed threshold boundaries', () => {
    it('speed exactly at moving threshold counts as moving', async () => {
      const tracker = await createStartedTracker({
        movingSpeedThreshold: 0.5,
        sleepThresholdMs: 50,
      });
      await tracker.start();
      const callback = jest.fn();
      tracker.setCallback(callback);

      tracker.processLocationObject(makeLocationObject({ speed: 0.5 }));
      expect(tracker.getState()).toBe('active');
    });

    it('speed exactly at stopped threshold counts as stopped for sleep check', async () => {
      const tracker = await createStartedTracker({
        stoppedSpeedThreshold: 0.3,
        sleepThresholdMs: 50,
      });
      await tracker.start();
      tracker.setCallback(jest.fn());

      // Move first
      tracker.processLocationObject(makeLocationObject({ speed: 1.0 }));

      // Advance past sleep threshold
      jest.spyOn(Date, 'now').mockReturnValue(Date.now() + 100);

      // Speed at threshold → should NOT trigger sleep (< not <=)
      tracker.processLocationObject(makeLocationObject({ speed: 0.3 }));
      expect(tracker.getState()).toBe('active');
    });

    it('speed just below stopped threshold triggers sleep after delay', async () => {
      const tracker = await createStartedTracker({
        stoppedSpeedThreshold: 0.3,
        sleepThresholdMs: 50,
      });
      await tracker.start();
      tracker.setCallback(jest.fn());

      tracker.processLocationObject(makeLocationObject({ speed: 1.0 }));
      jest.spyOn(Date, 'now').mockReturnValue(Date.now() + 100);

      tracker.processLocationObject(makeLocationObject({ speed: 0.29 }));
      expect(tracker.getState()).toBe('sleeping');
    });
  });

  describe('foreground callback integration', () => {
    it('foreground watch callback processes location', async () => {
      const tracker = await createStartedTracker();

      let watchCallback: ((loc: ReturnType<typeof makeLocationObject>) => void) | null = null;
      mockWatchPositionAsync.mockImplementation((_opts: unknown, cb: typeof watchCallback) => {
        watchCallback = cb;
        return Promise.resolve({ remove: jest.fn() });
      });

      const userCallback = jest.fn();
      tracker.setCallback(userCallback);
      await tracker.start();

      expect(watchCallback).not.toBeNull();

      // Simulate expo-location delivering a position
      watchCallback!(makeLocationObject({ latitude: 41.72, speed: 1.5 }));

      expect(userCallback).toHaveBeenCalledTimes(1);
      expect(userCallback.mock.calls[0][0].lat).toBe(41.72);
    });
  });

  describe('setCallback', () => {
    it('replaces previous callback', async () => {
      const tracker = await createStartedTracker();
      await tracker.start();

      const first = jest.fn();
      const second = jest.fn();

      tracker.setCallback(first);
      tracker.processLocationObject(makeLocationObject());
      expect(first).toHaveBeenCalledTimes(1);

      tracker.setCallback(second);
      tracker.processLocationObject(makeLocationObject());
      expect(first).toHaveBeenCalledTimes(1);
      expect(second).toHaveBeenCalledTimes(1);
    });
  });

  describe('multiple start/stop cycles', () => {
    it('can restart after stopping', async () => {
      const tracker = await createStartedTracker();

      await tracker.start();
      expect(tracker.isTracking()).toBe(true);

      mockIsTaskRegisteredAsync.mockResolvedValue(true);
      await tracker.stop();
      expect(tracker.isTracking()).toBe(false);

      mockIsTaskRegisteredAsync.mockResolvedValue(false);
      await tracker.start();
      expect(tracker.isTracking()).toBe(true);
      expect(tracker.getState()).toBe('active');
    });
  });
});
