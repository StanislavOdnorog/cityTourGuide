import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';

const BACKGROUND_LOCATION_TASK = 'city-stories-background-location';

export interface LocationUpdate {
  lat: number;
  lng: number;
  heading: number;
  speed: number;
  timestamp: number;
}

export type LocationCallback = (update: LocationUpdate) => void;

export interface LocationTrackerConfig {
  /** Foreground polling interval when walking (ms). Default: 5000 */
  activeIntervalMs: number;
  /** Time threshold to enter sleep mode when stationary (ms). Default: 120000 */
  sleepThresholdMs: number;
  /** Speed above which user is considered moving (m/s). Default: 0.5 */
  movingSpeedThreshold: number;
  /** Speed below which user is considered stopped (m/s). Default: 0.3 */
  stoppedSpeedThreshold: number;
  /** Distance filter for location updates (meters). Default: 10 */
  distanceFilterM: number;
}

const DEFAULT_CONFIG: LocationTrackerConfig = {
  activeIntervalMs: 5000,
  sleepThresholdMs: 120_000,
  movingSpeedThreshold: 0.5,
  stoppedSpeedThreshold: 0.3,
  distanceFilterM: 10,
};

type TrackerState = 'idle' | 'active' | 'sleeping';

/** Singleton reference for the background task to access */
let activeTracker: LocationTracker | null = null;

/**
 * Register the background location task at module level.
 * TaskManager requires tasks to be defined at module load time.
 */
TaskManager.defineTask(
  BACKGROUND_LOCATION_TASK,
  async ({
    data,
    error,
  }: TaskManager.TaskManagerTaskBody<{ locations: Location.LocationObject[] }>) => {
    if (error) return;
    if (!data || !activeTracker) return;

    const { locations } = data;
    for (const location of locations) {
      activeTracker.processLocationObject(location);
    }
  },
);

/**
 * LocationTracker provides adaptive background location tracking.
 *
 * Modes:
 * - **active**: polls every 5-10s when user is walking (speed > 0.5 m/s)
 * - **sleeping**: reduces updates when user is stationary (speed < 0.3 m/s for > 2 min)
 * - auto-resumes when movement is detected
 */
export class LocationTracker {
  private config: LocationTrackerConfig;
  private callback: LocationCallback | null = null;
  private state: TrackerState = 'idle';
  private foregroundSubscription: Location.LocationSubscription | null = null;
  private lastMovementTimestamp: number = 0;
  private lastLocation: LocationUpdate | null = null;

  constructor(config?: Partial<LocationTrackerConfig>) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Request foreground and background location permissions.
   * Returns true if both are granted.
   */
  async requestPermissions(): Promise<boolean> {
    const { status: fgStatus } = await Location.requestForegroundPermissionsAsync();
    if (fgStatus !== Location.PermissionStatus.GRANTED) {
      return false;
    }

    const { status: bgStatus } = await Location.requestBackgroundPermissionsAsync();
    return bgStatus === Location.PermissionStatus.GRANTED;
  }

  /**
   * Check if both foreground and background permissions are granted.
   */
  async hasPermissions(): Promise<boolean> {
    const fg = await Location.getForegroundPermissionsAsync();
    if (fg.status !== Location.PermissionStatus.GRANTED) return false;

    const bg = await Location.getBackgroundPermissionsAsync();
    return bg.status === Location.PermissionStatus.GRANTED;
  }

  /**
   * Set the callback invoked on each location update.
   */
  setCallback(callback: LocationCallback): void {
    this.callback = callback;
  }

  /**
   * Start location tracking (foreground + background).
   * Throws if permissions are not granted.
   */
  async start(): Promise<void> {
    if (this.state !== 'idle') return;

    const granted = await this.hasPermissions();
    if (!granted) {
      throw new Error('Location permissions not granted');
    }

    activeTracker = this; // eslint-disable-line @typescript-eslint/no-this-alias
    this.state = 'active';
    this.lastMovementTimestamp = Date.now();

    await this.startForegroundTracking();
    await this.startBackgroundTracking();
  }

  /**
   * Stop all location tracking and clean up.
   */
  async stop(): Promise<void> {
    if (this.state === 'idle') return;

    this.state = 'idle';
    this.lastLocation = null;

    if (this.foregroundSubscription) {
      this.foregroundSubscription.remove();
      this.foregroundSubscription = null;
    }

    const isRegistered = await TaskManager.isTaskRegisteredAsync(BACKGROUND_LOCATION_TASK);
    if (isRegistered) {
      await Location.stopLocationUpdatesAsync(BACKGROUND_LOCATION_TASK);
    }

    if (activeTracker === this) {
      activeTracker = null;
    }
  }

  /** Current tracker state */
  getState(): TrackerState {
    return this.state;
  }

  /** Whether tracking is active (either active or sleeping) */
  isTracking(): boolean {
    return this.state !== 'idle';
  }

  /** Whether the tracker is in sleep mode */
  isSleeping(): boolean {
    return this.state === 'sleeping';
  }

  /** Last received location update */
  getLastLocation(): LocationUpdate | null {
    return this.lastLocation;
  }

  /**
   * Process a raw expo-location object. Called from both foreground and background.
   * Public so the background task can invoke it.
   */
  processLocationObject(locationObj: Location.LocationObject): void {
    const speed = Math.max(locationObj.coords.speed ?? 0, 0);
    const heading = locationObj.coords.heading ?? -1;

    const update: LocationUpdate = {
      lat: locationObj.coords.latitude,
      lng: locationObj.coords.longitude,
      heading,
      speed,
      timestamp: locationObj.timestamp,
    };

    this.lastLocation = update;

    const wasSleeping = this.state === 'sleeping';
    this.updateAdaptiveMode(speed);

    if (this.state === 'sleeping') {
      // In sleep mode, don't emit updates
      return;
    }

    // If we just woke up, updateAdaptiveMode already emitted the location
    if (wasSleeping) return;

    this.callback?.(update);
  }

  /**
   * Adaptive mode logic:
   * - If speed > movingSpeedThreshold → mark as moving, reset timer
   * - If speed < stoppedSpeedThreshold for > sleepThresholdMs → enter sleep
   * - If sleeping and speed > movingSpeedThreshold → wake up
   */
  private updateAdaptiveMode(speed: number): void {
    const now = Date.now();

    if (speed >= this.config.movingSpeedThreshold) {
      this.lastMovementTimestamp = now;

      if (this.state === 'sleeping') {
        this.state = 'active';
        // Re-emit the location that woke us up
        if (this.lastLocation) {
          this.callback?.(this.lastLocation);
        }
      }
      return;
    }

    if (speed < this.config.stoppedSpeedThreshold && this.state === 'active') {
      const stationaryDuration = now - this.lastMovementTimestamp;
      if (stationaryDuration >= this.config.sleepThresholdMs) {
        this.state = 'sleeping';
      }
    }
  }

  private async startForegroundTracking(): Promise<void> {
    this.foregroundSubscription = await Location.watchPositionAsync(
      {
        accuracy: Location.Accuracy.High,
        timeInterval: this.config.activeIntervalMs,
        distanceInterval: this.config.distanceFilterM,
      },
      (location) => {
        this.processLocationObject(location);
      },
    );
  }

  private async startBackgroundTracking(): Promise<void> {
    const isRegistered = await TaskManager.isTaskRegisteredAsync(BACKGROUND_LOCATION_TASK);
    if (isRegistered) return;

    await Location.startLocationUpdatesAsync(BACKGROUND_LOCATION_TASK, {
      accuracy: Location.Accuracy.Balanced,
      timeInterval: this.config.activeIntervalMs * 2,
      distanceInterval: this.config.distanceFilterM,
      deferredUpdatesInterval: this.config.activeIntervalMs * 2,
      showsBackgroundLocationIndicator: true,
      foregroundService: {
        notificationTitle: 'City Stories Guide',
        notificationBody: 'Listening for stories nearby...',
        notificationColor: '#1A73E8',
      },
      activityType: Location.ActivityType.Fitness,
    });
  }
}

export { BACKGROUND_LOCATION_TASK };
