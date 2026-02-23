import { fetchNearbyStories } from '@/api/endpoints';
import { trackListening as trackListeningApi } from '@/api/endpoints';
import { AudioPlayer } from '@/services/audio';
import { LocationTracker, type LocationUpdate as TrackerLocationUpdate } from '@/services/location';
import {
  StoryEngine,
  type StoryFetcher,
  type StoryPlayer,
  type ListeningTracker,
  type ScoredCandidate,
} from '@/services/story-engine';
import { usePlayerStore } from '@/store/usePlayerStore';
import { useWalkStore } from '@/store/useWalkStore';

export interface WalkingPipelineConfig {
  radiusM: number;
  language: string;
  userId: string;
}

const DEFAULT_CONFIG: WalkingPipelineConfig = {
  radiusM: 150,
  language: 'en',
  userId: '',
};

/**
 * WalkingPipeline connects LocationTracker → StoryEngine → AudioPlayer
 * and keeps Zustand stores in sync.
 *
 * Full cycle:
 * 1. LocationTracker fires location update
 * 2. StoryEngine evaluates nearby stories via API
 * 3. AudioPlayer plays the selected story
 * 4. On story completion → trackListening API call
 * 5. Zustand stores updated at each step
 */
export class WalkingPipeline {
  private readonly locationTracker: LocationTracker;
  private readonly audioPlayer: AudioPlayer;
  private readonly storyEngine: StoryEngine;
  private config: WalkingPipelineConfig;
  private running = false;

  constructor(
    locationTracker: LocationTracker,
    audioPlayer: AudioPlayer,
    fetcher: StoryFetcher,
    config?: Partial<WalkingPipelineConfig>,
  ) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.locationTracker = locationTracker;
    this.audioPlayer = audioPlayer;

    const playerAdapter: StoryPlayer = {
      play: (candidate: ScoredCandidate) => {
        void this.audioPlayer.play(candidate);
        usePlayerStore.getState().setCurrentStory(candidate);
        usePlayerStore.getState().setIsPlaying(true);
      },
    };

    const trackerAdapter: ListeningTracker = {
      trackListening: (storyId: number, completed: boolean) => {
        usePlayerStore.getState().addListenedStory(storyId);
        const location = useWalkStore.getState().currentLocation;
        void trackListeningApi({
          user_id: this.config.userId,
          story_id: storyId,
          completed,
          lat: location?.lat,
          lng: location?.lng,
        });
      },
      getListenedStoryIds: () => usePlayerStore.getState().listenedStoryIds,
    };

    this.storyEngine = new StoryEngine(fetcher, playerAdapter, trackerAdapter, {
      radiusM: this.config.radiusM,
      language: this.config.language,
      userId: this.config.userId,
    });
  }

  isRunning(): boolean {
    return this.running;
  }

  getStoryEngine(): StoryEngine {
    return this.storyEngine;
  }

  getLocationTracker(): LocationTracker {
    return this.locationTracker;
  }

  getAudioPlayer(): AudioPlayer {
    return this.audioPlayer;
  }

  /**
   * Start the full pipeline: location tracking + story engine.
   * Requests location permissions if not granted.
   */
  async start(): Promise<void> {
    if (this.running) return;

    this.running = true;
    useWalkStore.getState().startWalking();

    this.audioPlayer.setOnComplete((completed) => {
      this.storyEngine.onStoryCompleted(completed);
      usePlayerStore.getState().setCurrentStory(null);
      usePlayerStore.getState().setIsPlaying(false);
      usePlayerStore.getState().setProgress(0, 0);
    });

    this.audioPlayer.setOnError(() => {
      usePlayerStore.getState().setIsPlaying(false);
    });

    this.locationTracker.setCallback((update: TrackerLocationUpdate) => {
      useWalkStore.getState().updateLocation({
        lat: update.lat,
        lng: update.lng,
        heading: update.heading,
        speed: update.speed,
      });

      void this.storyEngine.onLocationUpdate({
        lat: update.lat,
        lng: update.lng,
        heading: update.heading,
        speed: update.speed,
      });
    });

    this.storyEngine.start();
    await this.locationTracker.start();
  }

  /**
   * Stop the full pipeline: location tracking + story engine + audio.
   */
  async stop(): Promise<void> {
    if (!this.running) return;

    this.running = false;
    this.storyEngine.stop();
    await this.audioPlayer.stop();
    await this.locationTracker.stop();

    this.audioPlayer.setOnComplete(null);
    this.audioPlayer.setOnError(null);

    useWalkStore.getState().stopWalking();
    usePlayerStore.getState().setCurrentStory(null);
    usePlayerStore.getState().setIsPlaying(false);
    usePlayerStore.getState().setProgress(0, 0);
  }

  /**
   * Update pipeline configuration (language, radius, userId).
   */
  updateConfig(config: Partial<WalkingPipelineConfig>): void {
    this.config = { ...this.config, ...config };
    this.storyEngine.updateConfig(config);
  }

  /**
   * Full teardown: stop + destroy audio player.
   */
  async destroy(): Promise<void> {
    await this.stop();
    await this.audioPlayer.destroy();
    usePlayerStore.getState().reset();
  }
}

/**
 * Factory: creates a WalkingPipeline with default real dependencies.
 */
export function createWalkingPipeline(config?: Partial<WalkingPipelineConfig>): WalkingPipeline {
  const locationTracker = new LocationTracker();
  const audioPlayer = new AudioPlayer();
  const fetcher: StoryFetcher = { fetchNearbyStories };

  return new WalkingPipeline(locationTracker, audioPlayer, fetcher, config);
}
