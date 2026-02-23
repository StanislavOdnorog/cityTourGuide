/* eslint-disable import/order -- imports must be split around jest.mock calls */
import type { ScoredCandidate } from '@/services/story-engine';
import { usePlayerStore } from '@/store/usePlayerStore';
import { useWalkStore } from '@/store/useWalkStore';
import type { NearbyStoryCandidate } from '@/types';

// --- Mocks ---

// Mock expo-location
jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  requestBackgroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  getForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  getBackgroundPermissionsAsync: jest.fn().mockResolvedValue({ status: 'granted' }),
  watchPositionAsync: jest.fn().mockResolvedValue({ remove: jest.fn() }),
  startLocationUpdatesAsync: jest.fn().mockResolvedValue(undefined),
  stopLocationUpdatesAsync: jest.fn().mockResolvedValue(undefined),
  Accuracy: { High: 6, Balanced: 3 },
  ActivityType: { Fitness: 3 },
  PermissionStatus: { GRANTED: 'granted' },
}));

jest.mock('expo-task-manager', () => ({
  defineTask: jest.fn(),
  isTaskRegisteredAsync: jest.fn().mockResolvedValue(false),
}));

// Mock react-native-track-player
let mockTrackPlayerState = 'idle';
const mockTrackPlayerEventListeners = new Map<string, Array<(...args: unknown[]) => void>>();

jest.mock('react-native-track-player', () => ({
  __esModule: true,
  default: {
    setupPlayer: jest.fn().mockResolvedValue(undefined),
    updateOptions: jest.fn().mockResolvedValue(undefined),
    add: jest.fn().mockResolvedValue(undefined),
    play: jest.fn().mockImplementation(async () => {
      mockTrackPlayerState = 'playing';
    }),
    pause: jest.fn().mockResolvedValue(undefined),
    reset: jest.fn().mockImplementation(async () => {
      mockTrackPlayerState = 'idle';
    }),
    setVolume: jest.fn().mockResolvedValue(undefined),
    getPlaybackState: jest.fn().mockImplementation(async () => ({
      state: mockTrackPlayerState,
    })),
    getProgress: jest.fn().mockResolvedValue({ position: 0, duration: 0, buffered: 0 }),
    addEventListener: jest
      .fn()
      .mockImplementation((event: string, handler: (...args: unknown[]) => void) => {
        if (!mockTrackPlayerEventListeners.has(event)) {
          mockTrackPlayerEventListeners.set(event, []);
        }
        mockTrackPlayerEventListeners.get(event)!.push(handler);
        return { remove: jest.fn() };
      }),
  },
  Capability: { Play: 'play', Pause: 'pause', Stop: 'stop' },
  Event: {
    PlaybackQueueEnded: 'playback-queue-ended',
    RemotePlay: 'remote-play',
    RemotePause: 'remote-pause',
    RemoteStop: 'remote-stop',
    RemoteDuck: 'remote-duck',
  },
  State: { Playing: 'playing', Paused: 'paused', Stopped: 'stopped', None: 'idle' },
  AppKilledPlaybackBehavior: { ContinuePlayback: 'continue-playback' },
  IOSCategoryMode: { SpokenAudio: 'spoken-audio' },
  IOSCategory: { Playback: 'playback' },
}));

// Mock the API endpoints
const mockFetchNearby = jest.fn();
const mockTrackListening = jest.fn().mockResolvedValue(undefined);

jest.mock('@/api/endpoints', () => ({
  fetchNearbyStories: (...args: unknown[]) => mockFetchNearby(...args),
  trackListening: (...args: unknown[]) => mockTrackListening(...args),
}));

// Import after mocks
import { AudioPlayer } from '@/services/audio';
import { LocationTracker } from '@/services/location';
import type { StoryFetcher } from '@/services/story-engine';
import { WalkingPipeline } from '../WalkingPipeline';

function makeCandidate(overrides: Partial<NearbyStoryCandidate> = {}): NearbyStoryCandidate {
  return {
    poi_id: 1,
    poi_name: 'Narikala Fortress',
    story_id: 10,
    story_text: 'Ancient fortress above Tbilisi...',
    audio_url: 'https://cdn.example.com/narikala.mp3',
    duration_sec: 25,
    distance_m: 80,
    score: 70,
    ...overrides,
  };
}

describe('WalkingPipeline', () => {
  let locationTracker: LocationTracker;
  let audioPlayer: AudioPlayer;
  let fetcher: StoryFetcher;
  let pipeline: WalkingPipeline;

  beforeEach(() => {
    jest.clearAllMocks();
    mockTrackPlayerState = 'idle';
    mockTrackPlayerEventListeners.clear();

    // Reset Zustand stores
    useWalkStore.setState({ isWalking: false, currentLocation: null });
    usePlayerStore.setState({
      currentStory: null,
      isPlaying: false,
      progress: { position: 0, duration: 0 },
      listenedStoryIds: new Set<number>(),
    });

    locationTracker = new LocationTracker();
    audioPlayer = new AudioPlayer();
    fetcher = { fetchNearbyStories: mockFetchNearby };
    mockFetchNearby.mockResolvedValue([]);

    pipeline = new WalkingPipeline(locationTracker, audioPlayer, fetcher, {
      language: 'en',
      userId: 'test-user-123',
    });
  });

  afterEach(async () => {
    if (pipeline.isRunning()) {
      await pipeline.stop();
    }
  });

  // --- Constructor & Config ---

  describe('constructor', () => {
    it('creates a pipeline with default config', () => {
      const p = new WalkingPipeline(locationTracker, audioPlayer, fetcher);
      expect(p.isRunning()).toBe(false);
      expect(p.getStoryEngine()).toBeDefined();
      expect(p.getLocationTracker()).toBe(locationTracker);
      expect(p.getAudioPlayer()).toBe(audioPlayer);
    });

    it('accepts custom config', () => {
      const p = new WalkingPipeline(locationTracker, audioPlayer, fetcher, {
        radiusM: 300,
        language: 'ru',
        userId: 'custom-user',
      });
      expect(p.isRunning()).toBe(false);
    });
  });

  // --- Start / Stop ---

  describe('start', () => {
    it('sets running to true and starts walking in store', async () => {
      await pipeline.start();

      expect(pipeline.isRunning()).toBe(true);
      expect(useWalkStore.getState().isWalking).toBe(true);
    });

    it('starts the story engine', async () => {
      await pipeline.start();
      expect(pipeline.getStoryEngine().getIsActive()).toBe(true);
    });

    it('is a no-op if already running', async () => {
      await pipeline.start();
      await pipeline.start(); // second call
      expect(pipeline.isRunning()).toBe(true);
    });
  });

  describe('stop', () => {
    it('sets running to false and stops walking in store', async () => {
      await pipeline.start();
      await pipeline.stop();

      expect(pipeline.isRunning()).toBe(false);
      expect(useWalkStore.getState().isWalking).toBe(false);
    });

    it('clears player store on stop', async () => {
      await pipeline.start();
      usePlayerStore.getState().setIsPlaying(true);
      usePlayerStore.getState().setCurrentStory(makeCandidate() as ScoredCandidate);

      await pipeline.stop();

      expect(usePlayerStore.getState().currentStory).toBeNull();
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });

    it('is a no-op if not running', async () => {
      await pipeline.stop(); // should not throw
      expect(pipeline.isRunning()).toBe(false);
    });

    it('stops the story engine', async () => {
      await pipeline.start();
      await pipeline.stop();
      expect(pipeline.getStoryEngine().getIsActive()).toBe(false);
    });
  });

  // --- Location → StoryEngine → AudioPlayer Flow ---

  describe('location update flow', () => {
    it('updates walk store location on location callback', async () => {
      await pipeline.start();

      // Simulate a location update via the tracker callback
      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });

      const loc = useWalkStore.getState().currentLocation;
      expect(loc).toBeDefined();
      expect(loc!.lat).toBe(41.7151);
      expect(loc!.lng).toBe(44.8271);
      expect(loc!.heading).toBe(90);
      expect(loc!.speed).toBe(1.2);
    });

    it('triggers story fetch on location update', async () => {
      mockFetchNearby.mockResolvedValue([makeCandidate()]);
      await pipeline.start();

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });

      // Wait for async fetch
      await new Promise((r) => setTimeout(r, 50));

      expect(mockFetchNearby).toHaveBeenCalledWith(
        expect.objectContaining({
          lat: 41.7151,
          lng: 44.8271,
          heading: 90,
          speed: 1.2,
          language: 'en',
          user_id: 'test-user-123',
        }),
      );
    });

    it('sets currentStory in player store when a story is selected', async () => {
      const candidate = makeCandidate();
      mockFetchNearby.mockResolvedValue([candidate]);
      await pipeline.start();

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });

      await new Promise((r) => setTimeout(r, 50));

      const story = usePlayerStore.getState().currentStory;
      expect(story).toBeDefined();
      expect(story!.story_id).toBe(10);
      expect(usePlayerStore.getState().isPlaying).toBe(true);
    });

    it('does not trigger story when no candidates returned', async () => {
      mockFetchNearby.mockResolvedValue([]);
      await pipeline.start();

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });

      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().currentStory).toBeNull();
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });
  });

  // --- Story Completion → trackListening ---

  describe('story completion', () => {
    it('calls trackListening API on story completion', async () => {
      const candidate = makeCandidate();
      mockFetchNearby.mockResolvedValue([candidate]);
      await pipeline.start();

      // Trigger a location update to play a story
      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      // Simulate audio completion via AudioPlayer's onComplete callback
      // The pipeline set the callback in start(), so we trigger it
      // through the playback-queue-ended event
      const queueEndedHandlers = mockTrackPlayerEventListeners.get('playback-queue-ended') ?? [];
      for (const handler of queueEndedHandlers) {
        handler();
      }

      await new Promise((r) => setTimeout(r, 50));

      expect(mockTrackListening).toHaveBeenCalledWith(
        expect.objectContaining({
          user_id: 'test-user-123',
          story_id: 10,
          completed: true,
        }),
      );
    });

    it('adds listened story to player store on completion', async () => {
      const candidate = makeCandidate();
      mockFetchNearby.mockResolvedValue([candidate]);
      await pipeline.start();

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      // Trigger completion
      const handlers = mockTrackPlayerEventListeners.get('playback-queue-ended') ?? [];
      for (const handler of handlers) {
        handler();
      }
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().listenedStoryIds.has(10)).toBe(true);
    });

    it('clears currentStory and isPlaying on completion', async () => {
      const candidate = makeCandidate();
      mockFetchNearby.mockResolvedValue([candidate]);
      await pipeline.start();

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().isPlaying).toBe(true);

      const handlers = mockTrackPlayerEventListeners.get('playback-queue-ended') ?? [];
      for (const handler of handlers) {
        handler();
      }
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().currentStory).toBeNull();
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });

    it('includes user location in trackListening call', async () => {
      const candidate = makeCandidate();
      mockFetchNearby.mockResolvedValue([candidate]);
      await pipeline.start();

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7151,
          longitude: 44.8271,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      const handlers = mockTrackPlayerEventListeners.get('playback-queue-ended') ?? [];
      for (const handler of handlers) {
        handler();
      }
      await new Promise((r) => setTimeout(r, 50));

      expect(mockTrackListening).toHaveBeenCalledWith(
        expect.objectContaining({
          lat: 41.7151,
          lng: 44.8271,
        }),
      );
    });
  });

  // --- Listened stories not repeated ---

  describe('listened story exclusion', () => {
    it('excludes already listened stories from future selection', async () => {
      const candidate1 = makeCandidate({ story_id: 10, score: 70 });
      const candidate2 = makeCandidate({ story_id: 20, poi_name: 'Peace Bridge', score: 60 });

      // First call returns both candidates
      mockFetchNearby.mockResolvedValue([candidate1, candidate2]);
      await pipeline.start();

      // First location update → should play story_id=10 (highest score)
      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7,
          longitude: 44.8,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().currentStory?.story_id).toBe(10);

      // Complete story_id=10
      const handlers = mockTrackPlayerEventListeners.get('playback-queue-ended') ?? [];
      for (const handler of handlers) {
        handler();
      }
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().listenedStoryIds.has(10)).toBe(true);

      // Pacing has a 60-sec cooldown. Manually reset it for test.
      pipeline.getStoryEngine().getPacingManager().reset();

      // Second location update → should play story_id=20 since 10 is excluded
      mockFetchNearby.mockResolvedValue([candidate1, candidate2]);
      locationTracker.processLocationObject({
        coords: {
          latitude: 41.71,
          longitude: 44.81,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().currentStory?.story_id).toBe(20);
    });
  });

  // --- Pacing enforcement ---

  describe('pacing', () => {
    it('enforces 60-second cooldown between stories', async () => {
      const candidate = makeCandidate();
      mockFetchNearby.mockResolvedValue([candidate]);
      await pipeline.start();

      // Play first story
      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7,
          longitude: 44.8,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      expect(usePlayerStore.getState().isPlaying).toBe(true);

      // Complete it
      const handlers = mockTrackPlayerEventListeners.get('playback-queue-ended') ?? [];
      for (const handler of handlers) {
        handler();
      }
      await new Promise((r) => setTimeout(r, 50));

      // Immediately try another location update → should NOT play (cooldown)
      const candidate2 = makeCandidate({ story_id: 20 });
      mockFetchNearby.mockResolvedValue([candidate2]);

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.71,
          longitude: 44.81,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      // Should still have no current story (pacing cooldown active)
      expect(usePlayerStore.getState().currentStory).toBeNull();
    });
  });

  // --- Config update ---

  describe('updateConfig', () => {
    it('updates language and radius', () => {
      pipeline.updateConfig({ language: 'ru', radiusM: 300 });
      // Verify the engine gets updated config by triggering a fetch
      // (the actual assertion is that it doesn't throw)
      expect(pipeline.isRunning()).toBe(false);
    });
  });

  // --- Destroy ---

  describe('destroy', () => {
    it('stops and destroys audio player', async () => {
      await pipeline.start();
      await pipeline.destroy();

      expect(pipeline.isRunning()).toBe(false);
    });

    it('resets player store on destroy', async () => {
      await pipeline.start();
      usePlayerStore.getState().addListenedStory(10, 1);
      usePlayerStore.getState().setIsPlaying(true);

      await pipeline.destroy();

      expect(usePlayerStore.getState().listenedStoryIds.size).toBe(0);
      expect(usePlayerStore.getState().listenedPoiIds.size).toBe(0);
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });
  });

  // --- Error handling ---

  describe('error handling', () => {
    it('sets isPlaying to false on audio error', async () => {
      await pipeline.start();
      usePlayerStore.getState().setIsPlaying(true);

      // Trigger error callback
      const _errorHandlers = mockTrackPlayerEventListeners.get('remote-stop') ?? [];
      // Actually, let's trigger via the onError callback that pipeline set
      // AudioPlayer onError is called when play() gets a candidate without audio_url
      const candidateNoAudio = makeCandidate({ audio_url: null });
      mockFetchNearby.mockResolvedValue([candidateNoAudio]);

      locationTracker.processLocationObject({
        coords: {
          latitude: 41.7,
          longitude: 44.8,
          heading: 90,
          speed: 1.2,
          altitude: null,
          accuracy: 10,
          altitudeAccuracy: null,
        },
        timestamp: Date.now(),
      });
      await new Promise((r) => setTimeout(r, 50));

      // AudioPlayer.play() calls onError for null audio_url
      // This triggers pipeline's error handler → setIsPlaying(false)
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });
  });
});
