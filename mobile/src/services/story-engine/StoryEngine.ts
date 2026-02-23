import type { NearbyStoryCandidate } from '@/types';
import { PacingManager } from './PacingManager';
import { scoreAndRankCandidates, type ScoredCandidate } from './ScoringAlgorithm';

export interface LocationUpdate {
  lat: number;
  lng: number;
  heading: number;
  speed: number;
}

export interface StoryFetcher {
  fetchNearbyStories(params: {
    lat: number;
    lng: number;
    radius?: number;
    heading?: number;
    speed?: number;
    language?: string;
    user_id?: string;
  }): Promise<NearbyStoryCandidate[]>;
}

export interface StoryPlayer {
  play(candidate: ScoredCandidate): void;
}

export interface ListeningTracker {
  trackListening(storyId: number, completed: boolean): void;
  getListenedStoryIds(): Set<number>;
}

export interface CachePrefetcher {
  prefetchAhead(
    candidates: NearbyStoryCandidate[],
    lat: number,
    lng: number,
    heading: number,
  ): void;
}

export interface StoryEngineConfig {
  radiusM: number;
  language: string;
  userId: string;
}

const DEFAULT_RADIUS_M = 150;
const DEFAULT_LANGUAGE = 'en';

/**
 * StoryEngine orchestrates the full pipeline:
 * location update → API fetch → scoring → pacing → playback trigger.
 */
export class StoryEngine {
  private readonly pacing: PacingManager;
  private readonly fetcher: StoryFetcher;
  private readonly player: StoryPlayer;
  private readonly tracker: ListeningTracker;
  private readonly cachePrefetcher: CachePrefetcher | null;
  private config: StoryEngineConfig;

  private currentStory: ScoredCandidate | null = null;
  private isActive = false;

  constructor(
    fetcher: StoryFetcher,
    player: StoryPlayer,
    tracker: ListeningTracker,
    config?: Partial<StoryEngineConfig>,
    cachePrefetcher?: CachePrefetcher,
  ) {
    this.fetcher = fetcher;
    this.player = player;
    this.tracker = tracker;
    this.cachePrefetcher = cachePrefetcher ?? null;
    this.pacing = new PacingManager();
    this.config = {
      radiusM: config?.radiusM ?? DEFAULT_RADIUS_M,
      language: config?.language ?? DEFAULT_LANGUAGE,
      userId: config?.userId ?? '',
    };
  }

  /**
   * Start the story engine. Location updates will trigger story evaluation.
   */
  start(): void {
    this.isActive = true;
  }

  /**
   * Stop the story engine. No new stories will be triggered.
   */
  stop(): void {
    this.isActive = false;
  }

  getIsActive(): boolean {
    return this.isActive;
  }

  getCurrentStory(): ScoredCandidate | null {
    return this.currentStory;
  }

  getPacingManager(): PacingManager {
    return this.pacing;
  }

  updateConfig(config: Partial<StoryEngineConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * Called on each location update. Evaluates whether to fetch and play a story.
   * Returns the selected candidate if a story was triggered, null otherwise.
   */
  async onLocationUpdate(location: LocationUpdate): Promise<ScoredCandidate | null> {
    if (!this.isActive) return null;
    if (!this.pacing.canPlayNext()) return null;

    const candidates = await this.fetcher.fetchNearbyStories({
      lat: location.lat,
      lng: location.lng,
      radius: this.config.radiusM,
      heading: location.heading,
      speed: location.speed,
      language: this.config.language,
      user_id: this.config.userId,
    });

    // Trigger background pre-fetching for nearby stories ahead
    this.cachePrefetcher?.prefetchAhead(candidates, location.lat, location.lng, location.heading);

    if (candidates.length === 0) return null;

    const listenedIds = this.tracker.getListenedStoryIds();
    const scored = scoreAndRankCandidates(candidates, listenedIds);
    if (scored.length === 0) return null;

    const selected = this.pacing.selectByPace(scored, location.speed);
    if (!selected) return null;

    this.currentStory = selected;
    this.pacing.markPlayStarted();
    this.player.play(selected);

    return selected;
  }

  /**
   * Called when the current story finishes playing.
   */
  onStoryCompleted(completed: boolean): void {
    if (this.currentStory) {
      this.tracker.trackListening(this.currentStory.story_id, completed);
    }
    this.pacing.markPlayEnded();
    this.currentStory = null;
  }

  /**
   * Reset engine state (for testing or session restart).
   */
  reset(): void {
    this.pacing.reset();
    this.currentStory = null;
    this.isActive = false;
  }
}
