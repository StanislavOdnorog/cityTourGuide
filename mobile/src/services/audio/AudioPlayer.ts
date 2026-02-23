import TrackPlayer, {
  Capability,
  Event,
  State,
  AppKilledPlaybackBehavior,
  IOSCategoryMode,
  IOSCategory,
} from 'react-native-track-player';
import type { ScoredCandidate } from '@/services/story-engine';

export type AudioPlayerEvent = 'complete' | 'error';
export type OnCompleteCallback = (completed: boolean) => void;
export type OnErrorCallback = (error: Error) => void;

export interface AudioPlayerConfig {
  fadeInDurationMs: number;
  fadeOutDurationMs: number;
  duckingVolume: number;
}

const DEFAULT_CONFIG: AudioPlayerConfig = {
  fadeInDurationMs: 700,
  fadeOutDurationMs: 700,
  duckingVolume: 0.3,
};

const FADE_STEP_MS = 50;

export class AudioPlayer {
  private config: AudioPlayerConfig;
  private initialized = false;
  private currentCandidate: ScoredCandidate | null = null;
  private onComplete: OnCompleteCallback | null = null;
  private onError: OnErrorCallback | null = null;
  private fadeTimer: ReturnType<typeof setInterval> | null = null;
  private playbackEndSubscription: { remove: () => void } | null = null;
  private remotePlaySubscription: { remove: () => void } | null = null;
  private remotePauseSubscription: { remove: () => void } | null = null;
  private remoteStopSubscription: { remove: () => void } | null = null;
  private remoteDuckSubscription: { remove: () => void } | null = null;

  constructor(config?: Partial<AudioPlayerConfig>) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  async setup(): Promise<void> {
    if (this.initialized) return;

    await TrackPlayer.setupPlayer({
      iosCategory: IOSCategory.Playback,
      iosCategoryMode: IOSCategoryMode.SpokenAudio,
    });

    await TrackPlayer.updateOptions({
      capabilities: [Capability.Play, Capability.Pause, Capability.Stop],
      compactCapabilities: [Capability.Play, Capability.Pause],
      android: {
        appKilledPlaybackBehavior: AppKilledPlaybackBehavior.ContinuePlayback,
      },
      notificationCapabilities: [Capability.Play, Capability.Pause],
    });

    this.subscribeToEvents();
    this.initialized = true;
  }

  private subscribeToEvents(): void {
    this.playbackEndSubscription = TrackPlayer.addEventListener(Event.PlaybackQueueEnded, () => {
      this.handlePlaybackComplete(true);
    });

    this.remotePlaySubscription = TrackPlayer.addEventListener(Event.RemotePlay, async () => {
      await TrackPlayer.play();
    });

    this.remotePauseSubscription = TrackPlayer.addEventListener(Event.RemotePause, async () => {
      await TrackPlayer.pause();
    });

    this.remoteStopSubscription = TrackPlayer.addEventListener(Event.RemoteStop, async () => {
      await this.stop();
    });

    this.remoteDuckSubscription = TrackPlayer.addEventListener(Event.RemoteDuck, async (event) => {
      if (event.permanent) {
        await this.stop();
      } else if (event.paused) {
        await TrackPlayer.pause();
      } else {
        // Duck ended — resume playback and restore volume
        await TrackPlayer.setVolume(1.0);
        await TrackPlayer.play();
      }
    });
  }

  async play(candidate: ScoredCandidate): Promise<void> {
    if (!this.initialized) {
      await this.setup();
    }

    if (!candidate.audio_url) {
      this.onError?.(new Error('No audio URL provided'));
      return;
    }

    this.clearFadeTimer();

    await TrackPlayer.reset();

    this.currentCandidate = candidate;

    await TrackPlayer.add({
      id: String(candidate.story_id),
      url: candidate.audio_url,
      title: candidate.poi_name,
      artist: 'City Stories Guide',
      duration: candidate.duration_sec ?? undefined,
    });

    await TrackPlayer.setVolume(0);
    await TrackPlayer.play();
    this.fadeIn();
  }

  async pause(): Promise<void> {
    const state = await TrackPlayer.getPlaybackState();
    if (state.state === State.Playing) {
      await TrackPlayer.pause();
    }
  }

  async resume(): Promise<void> {
    const state = await TrackPlayer.getPlaybackState();
    if (state.state === State.Paused) {
      await TrackPlayer.play();
    }
  }

  async stop(): Promise<void> {
    this.clearFadeTimer();

    const state = await TrackPlayer.getPlaybackState();
    if (state.state === State.Playing || state.state === State.Paused) {
      this.handlePlaybackComplete(false);
    }

    await TrackPlayer.reset();
    this.currentCandidate = null;
  }

  async getIsPlaying(): Promise<boolean> {
    const state = await TrackPlayer.getPlaybackState();
    return state.state === State.Playing;
  }

  async getProgress(): Promise<{ position: number; duration: number }> {
    const progress = await TrackPlayer.getProgress();
    return { position: progress.position, duration: progress.duration };
  }

  getCurrentCandidate(): ScoredCandidate | null {
    return this.currentCandidate;
  }

  setOnComplete(callback: OnCompleteCallback | null): void {
    this.onComplete = callback;
  }

  setOnError(callback: OnErrorCallback | null): void {
    this.onError = callback;
  }

  getIsInitialized(): boolean {
    return this.initialized;
  }

  async destroy(): Promise<void> {
    this.clearFadeTimer();
    this.playbackEndSubscription?.remove();
    this.remotePlaySubscription?.remove();
    this.remotePauseSubscription?.remove();
    this.remoteStopSubscription?.remove();
    this.remoteDuckSubscription?.remove();
    this.playbackEndSubscription = null;
    this.remotePlaySubscription = null;
    this.remotePauseSubscription = null;
    this.remoteStopSubscription = null;
    this.remoteDuckSubscription = null;
    this.onComplete = null;
    this.onError = null;
    this.currentCandidate = null;

    if (this.initialized) {
      await TrackPlayer.reset();
      this.initialized = false;
    }
  }

  private async fadeIn(): Promise<void> {
    const steps = Math.ceil(this.config.fadeInDurationMs / FADE_STEP_MS);
    if (steps <= 0) {
      await TrackPlayer.setVolume(1.0);
      return;
    }

    let currentStep = 0;
    return new Promise<void>((resolve) => {
      this.fadeTimer = setInterval(async () => {
        currentStep++;
        const volume = Math.min(currentStep / steps, 1.0);
        await TrackPlayer.setVolume(volume);

        if (currentStep >= steps) {
          this.clearFadeTimer();
          resolve();
        }
      }, FADE_STEP_MS);
    });
  }

  private async fadeOut(): Promise<void> {
    const steps = Math.ceil(this.config.fadeOutDurationMs / FADE_STEP_MS);
    if (steps <= 0) {
      await TrackPlayer.setVolume(0);
      return;
    }

    let currentStep = 0;
    return new Promise<void>((resolve) => {
      this.fadeTimer = setInterval(async () => {
        currentStep++;
        const volume = Math.max(1.0 - currentStep / steps, 0);
        await TrackPlayer.setVolume(volume);

        if (currentStep >= steps) {
          this.clearFadeTimer();
          resolve();
        }
      }, FADE_STEP_MS);
    });
  }

  private clearFadeTimer(): void {
    if (this.fadeTimer !== null) {
      clearInterval(this.fadeTimer);
      this.fadeTimer = null;
    }
  }

  private handlePlaybackComplete(completed: boolean): void {
    this.onComplete?.(completed);
  }
}
