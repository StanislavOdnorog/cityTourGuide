import type { ScoredCandidate } from '@/services/story-engine';

// --- Mock react-native-track-player ---

let mockState = 'idle';
let mockVolume = 1.0;
let mockTracks: Array<Record<string, unknown>> = [];
let mockProgress = { position: 0, duration: 0, buffered: 0 };

const eventListeners = new Map<string, Array<(...args: unknown[]) => void>>();

const _setupPlayer = jest.fn().mockResolvedValue(undefined);
const _updateOptions = jest.fn().mockResolvedValue(undefined);
const _add = jest.fn().mockImplementation(async (track: Record<string, unknown>) => {
  mockTracks.push(track);
});
const _play = jest.fn().mockImplementation(async () => {
  mockState = 'playing';
});
const _pause = jest.fn().mockImplementation(async () => {
  mockState = 'paused';
});
const _reset = jest.fn().mockImplementation(async () => {
  mockState = 'idle';
  mockTracks = [];
  mockVolume = 1.0;
});
const _setVolume = jest.fn().mockImplementation(async (vol: number) => {
  mockVolume = vol;
});
const _getPlaybackState = jest.fn().mockImplementation(async () => ({ state: mockState }));
const _getProgress = jest.fn().mockImplementation(async () => mockProgress);
const _addEventListener = jest
  .fn()
  .mockImplementation((event: string, handler: (...args: unknown[]) => void) => {
    if (!eventListeners.has(event)) {
      eventListeners.set(event, []);
    }
    eventListeners.get(event)!.push(handler);
    return { remove: jest.fn() };
  });

jest.mock('react-native-track-player', () => ({
  __esModule: true,
  default: {
    setupPlayer: _setupPlayer,
    updateOptions: _updateOptions,
    add: _add,
    play: _play,
    pause: _pause,
    reset: _reset,
    setVolume: _setVolume,
    getPlaybackState: _getPlaybackState,
    getProgress: _getProgress,
    addEventListener: _addEventListener,
  },
  Capability: {
    Play: 'play',
    Pause: 'pause',
    Stop: 'stop',
  },
  Event: {
    PlaybackQueueEnded: 'playback-queue-ended',
    RemotePlay: 'remote-play',
    RemotePause: 'remote-pause',
    RemoteStop: 'remote-stop',
    RemoteDuck: 'remote-duck',
  },
  State: {
    Playing: 'playing',
    Paused: 'paused',
    Stopped: 'stopped',
    None: 'idle',
    Buffering: 'buffering',
  },
  AppKilledPlaybackBehavior: {
    ContinuePlayback: 'continue-playback',
  },
  IOSCategoryMode: {
    SpokenAudio: 'spoken-audio',
  },
  IOSCategory: {
    Playback: 'playback',
  },
}));

import { AudioPlayer } from '../AudioPlayer';

// --- Helpers ---

function makeCandidate(overrides: Partial<ScoredCandidate> = {}): ScoredCandidate {
  return {
    poi_id: 1,
    poi_name: 'Holy Trinity Cathedral',
    story_id: 42,
    story_text: 'A story about the cathedral',
    audio_url: 'https://cdn.example.com/audio/42.mp3',
    duration_sec: 60,
    distance_m: 80,
    score: 75,
    localScore: 85,
    ...overrides,
  };
}

function emitEvent(event: string, data?: unknown): void {
  const handlers = eventListeners.get(event) ?? [];
  for (const handler of handlers) {
    handler(data);
  }
}

function resetMocks(): void {
  mockState = 'idle';
  mockVolume = 1.0;
  mockTracks = [];
  mockProgress = { position: 0, duration: 0, buffered: 0 };
  eventListeners.clear();
  jest.clearAllMocks();
}

// --- Tests ---

describe('AudioPlayer', () => {
  beforeEach(() => {
    resetMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    // Clear any pending fade timers
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  describe('constructor', () => {
    it('creates with default config', () => {
      const player = new AudioPlayer();
      expect(player.getIsInitialized()).toBe(false);
      expect(player.getCurrentCandidate()).toBeNull();
    });

    it('accepts custom config overrides', () => {
      const player = new AudioPlayer({ fadeInDurationMs: 500, duckingVolume: 0.5 });
      expect(player.getIsInitialized()).toBe(false);
    });
  });

  describe('setup', () => {
    it('initializes TrackPlayer with iOS spoken audio category', async () => {
      const player = new AudioPlayer();
      await player.setup();

      expect(_setupPlayer).toHaveBeenCalledWith(
        expect.objectContaining({
          iosCategory: 'playback',
          iosCategoryMode: 'spoken-audio',
        }),
      );
      expect(player.getIsInitialized()).toBe(true);
    });

    it('configures play/pause/stop capabilities', async () => {
      const player = new AudioPlayer();
      await player.setup();

      expect(_updateOptions).toHaveBeenCalledWith(
        expect.objectContaining({
          capabilities: ['play', 'pause', 'stop'],
          compactCapabilities: ['play', 'pause'],
        }),
      );
    });

    it('subscribes to all required events', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const subscribedEvents = _addEventListener.mock.calls.map(
        (call: [string, unknown]) => call[0],
      );
      expect(subscribedEvents).toContain('playback-queue-ended');
      expect(subscribedEvents).toContain('remote-play');
      expect(subscribedEvents).toContain('remote-pause');
      expect(subscribedEvents).toContain('remote-stop');
      expect(subscribedEvents).toContain('remote-duck');
    });

    it('no-ops on second setup call', async () => {
      const player = new AudioPlayer();
      await player.setup();
      await player.setup();

      expect(_setupPlayer).toHaveBeenCalledTimes(1);
    });
  });

  describe('play', () => {
    it('adds track and starts playback', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const candidate = makeCandidate();
      await player.play(candidate);

      expect(_reset).toHaveBeenCalled();
      expect(_add).toHaveBeenCalledWith(
        expect.objectContaining({
          id: '42',
          url: 'https://cdn.example.com/audio/42.mp3',
          title: 'Holy Trinity Cathedral',
          artist: 'City Stories Guide',
          duration: 60,
        }),
      );
      expect(_play).toHaveBeenCalled();
      expect(player.getCurrentCandidate()).toBe(candidate);
    });

    it('auto-initializes if setup not called', async () => {
      const player = new AudioPlayer();
      await player.play(makeCandidate());

      expect(_setupPlayer).toHaveBeenCalled();
      expect(player.getIsInitialized()).toBe(true);
    });

    it('calls onError if audio_url is null', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const onError = jest.fn();
      player.setOnError(onError);

      await player.play(makeCandidate({ audio_url: null }));

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
      expect(onError.mock.calls[0][0].message).toBe('No audio URL provided');
      expect(_add).not.toHaveBeenCalled();
    });

    it('starts volume at 0 for fade-in', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate());

      // First setVolume call should be 0 (before fade starts)
      expect(_setVolume.mock.calls[0][0]).toBe(0);
    });

    it('handles duration_sec null gracefully', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate({ duration_sec: null }));

      expect(_add).toHaveBeenCalledWith(
        expect.objectContaining({
          duration: undefined,
        }),
      );
    });

    it('resets previous track before playing new one', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate({ story_id: 1 }));
      _reset.mockClear();

      await player.play(makeCandidate({ story_id: 2 }));
      expect(_reset).toHaveBeenCalledTimes(1);
      expect(player.getCurrentCandidate()?.story_id).toBe(2);
    });
  });

  describe('pause and resume', () => {
    it('pauses when playing', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'playing';
      await player.pause();

      expect(_pause).toHaveBeenCalled();
    });

    it('does not pause when not playing', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'paused';
      await player.pause();

      expect(_pause).not.toHaveBeenCalled();
    });

    it('resumes when paused', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'paused';
      await player.resume();

      expect(_play).toHaveBeenCalled();
    });

    it('does not resume when not paused', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'idle';
      await player.resume();

      expect(_play).not.toHaveBeenCalled();
    });
  });

  describe('stop', () => {
    it('calls onComplete with false and clears candidate', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const onComplete = jest.fn();
      player.setOnComplete(onComplete);

      await player.play(makeCandidate());
      // State is now 'playing'

      await player.stop();

      expect(onComplete).toHaveBeenCalledWith(false);
      expect(player.getCurrentCandidate()).toBeNull();
    });

    it('resets TrackPlayer', async () => {
      const player = new AudioPlayer();
      await player.setup();
      await player.play(makeCandidate());

      _reset.mockClear();
      await player.stop();

      expect(_reset).toHaveBeenCalled();
    });

    it('no-ops onComplete when already idle', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const onComplete = jest.fn();
      player.setOnComplete(onComplete);

      mockState = 'idle';
      await player.stop();

      expect(onComplete).not.toHaveBeenCalled();
    });

    it('stops from paused state', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const onComplete = jest.fn();
      player.setOnComplete(onComplete);

      await player.play(makeCandidate());
      mockState = 'paused';

      await player.stop();

      expect(onComplete).toHaveBeenCalledWith(false);
    });
  });

  describe('fade in', () => {
    it('starts fade-in by setting initial volume to 0', async () => {
      const player = new AudioPlayer({ fadeInDurationMs: 500 });
      await player.setup();

      await player.play(makeCandidate());

      // Volume starts at 0
      expect(_setVolume.mock.calls[0][0]).toBe(0);
      // play() is called after setVolume(0)
      expect(_play).toHaveBeenCalled();
    });

    it('with zero fade duration sets volume to 1 immediately', async () => {
      const player = new AudioPlayer({ fadeInDurationMs: 0 });
      await player.setup();

      await player.play(makeCandidate());

      const calls = _setVolume.mock.calls.map((c: [number]) => c[0]);
      expect(calls).toContain(1.0);
    });

    it('clears fade timer on stop', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate());
      // Fade-in timer is running. Stop should clear it.
      await player.stop();

      // No error thrown, timer was cleared cleanly
      expect(player.getCurrentCandidate()).toBeNull();
    });
  });

  describe('playback events', () => {
    it('fires onComplete(true) when queue ends naturally', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const onComplete = jest.fn();
      player.setOnComplete(onComplete);

      await player.play(makeCandidate());

      // Simulate track finishing
      emitEvent('playback-queue-ended');

      expect(onComplete).toHaveBeenCalledWith(true);
    });

    it('handles RemotePlay event', async () => {
      const player = new AudioPlayer();
      await player.setup();

      _play.mockClear();
      emitEvent('remote-play');

      expect(_play).toHaveBeenCalled();
    });

    it('handles RemotePause event', async () => {
      const player = new AudioPlayer();
      await player.setup();

      emitEvent('remote-pause');

      expect(_pause).toHaveBeenCalled();
    });

    it('handles RemoteStop — stops playback', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate());

      const onComplete = jest.fn();
      player.setOnComplete(onComplete);

      // Emit remote stop — the handler calls this.stop() which is async
      emitEvent('remote-stop');
      // Flush multiple microtask ticks for stop()'s await chain
      for (let i = 0; i < 5; i++) {
        jest.advanceTimersByTime(0);
        await Promise.resolve();
      }

      expect(player.getCurrentCandidate()).toBeNull();
    });
  });

  describe('audio ducking (RemoteDuck)', () => {
    it('pauses on duck with paused flag', async () => {
      const player = new AudioPlayer();
      await player.setup();

      emitEvent('remote-duck', { paused: true, permanent: false });

      expect(_pause).toHaveBeenCalled();
    });

    it('stops on permanent interruption', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate());

      emitEvent('remote-duck', { paused: false, permanent: true });
      // Flush multiple microtask ticks for stop()'s await chain
      for (let i = 0; i < 5; i++) {
        jest.advanceTimersByTime(0);
        await Promise.resolve();
      }

      expect(player.getCurrentCandidate()).toBeNull();
    });

    it('resumes and restores volume when duck ends', async () => {
      const player = new AudioPlayer();
      await player.setup();

      _setVolume.mockClear();
      _play.mockClear();

      emitEvent('remote-duck', { paused: false, permanent: false });
      // Handler is async, wait for it to settle
      jest.advanceTimersByTime(0);
      await Promise.resolve();

      expect(_setVolume).toHaveBeenCalledWith(1.0);
      expect(_play).toHaveBeenCalled();
    });
  });

  describe('getIsPlaying', () => {
    it('returns true when playing', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'playing';
      expect(await player.getIsPlaying()).toBe(true);
    });

    it('returns false when paused', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'paused';
      expect(await player.getIsPlaying()).toBe(false);
    });

    it('returns false when idle', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockState = 'idle';
      expect(await player.getIsPlaying()).toBe(false);
    });
  });

  describe('getProgress', () => {
    it('returns position and duration', async () => {
      const player = new AudioPlayer();
      await player.setup();

      mockProgress = { position: 15.5, duration: 60, buffered: 30 };
      const progress = await player.getProgress();

      expect(progress).toEqual({ position: 15.5, duration: 60 });
    });
  });

  describe('callbacks', () => {
    it('setOnComplete replaces previous callback', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const first = jest.fn();
      const second = jest.fn();

      player.setOnComplete(first);
      player.setOnComplete(second);

      await player.play(makeCandidate());
      emitEvent('playback-queue-ended');

      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledWith(true);
    });

    it('setOnComplete(null) removes callback', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const onComplete = jest.fn();
      player.setOnComplete(onComplete);
      player.setOnComplete(null);

      await player.play(makeCandidate());
      emitEvent('playback-queue-ended');

      expect(onComplete).not.toHaveBeenCalled();
    });

    it('setOnError replaces previous callback', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const first = jest.fn();
      const second = jest.fn();

      player.setOnError(first);
      player.setOnError(second);

      await player.play(makeCandidate({ audio_url: null }));

      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledWith(expect.any(Error));
    });
  });

  describe('destroy', () => {
    it('resets all state and cleans up', async () => {
      const player = new AudioPlayer();
      await player.setup();

      await player.play(makeCandidate());

      await player.destroy();

      expect(player.getIsInitialized()).toBe(false);
      expect(player.getCurrentCandidate()).toBeNull();
      expect(_reset).toHaveBeenCalled();
    });

    it('is safe to call when not initialized', async () => {
      const player = new AudioPlayer();
      await player.destroy();

      expect(player.getIsInitialized()).toBe(false);
    });
  });

  describe('StoryPlayer interface compatibility', () => {
    it('play method accepts ScoredCandidate', async () => {
      const player = new AudioPlayer();
      await player.setup();

      const candidate = makeCandidate();
      await player.play(candidate);

      expect(player.getCurrentCandidate()).toBe(candidate);
    });
  });

  describe('lock screen controls', () => {
    it('configures lock screen capabilities', async () => {
      const player = new AudioPlayer();
      await player.setup();

      expect(_updateOptions).toHaveBeenCalledWith(
        expect.objectContaining({
          capabilities: expect.arrayContaining(['play', 'pause', 'stop']),
          compactCapabilities: expect.arrayContaining(['play', 'pause']),
        }),
      );
    });

    it('configures Android to continue playback when app killed', async () => {
      const player = new AudioPlayer();
      await player.setup();

      expect(_updateOptions).toHaveBeenCalledWith(
        expect.objectContaining({
          android: expect.objectContaining({
            appKilledPlaybackBehavior: 'continue-playback',
          }),
        }),
      );
    });
  });
});
