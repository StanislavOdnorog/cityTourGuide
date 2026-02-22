import type { ScoredCandidate } from './ScoringAlgorithm';

const MIN_COOLDOWN_MS = 60_000;
const FAST_WALK_SPEED_MS = 4 / 3.6; // 4 km/h → ~1.11 m/s
const SLOW_WALK_SPEED_MS = 3 / 3.6; // 3 km/h → ~0.83 m/s

const SHORT_STORY_MAX_SEC = 20;
const LONG_STORY_MIN_SEC = 30;

export class PacingManager {
  private lastPlayEndTime = 0;
  private isPlaying = false;

  /**
   * Returns true if enough time has passed since the last story ended
   * and no story is currently playing.
   */
  canPlayNext(now: number = Date.now()): boolean {
    if (this.isPlaying) return false;
    if (this.lastPlayEndTime === 0) return true;
    return now - this.lastPlayEndTime >= MIN_COOLDOWN_MS;
  }

  /**
   * Returns milliseconds remaining until next story can play.
   * Returns 0 if ready to play.
   */
  cooldownRemaining(now: number = Date.now()): number {
    if (this.isPlaying) return MIN_COOLDOWN_MS;
    if (this.lastPlayEndTime === 0) return 0;
    const elapsed = now - this.lastPlayEndTime;
    return Math.max(0, MIN_COOLDOWN_MS - elapsed);
  }

  /**
   * Mark that a story has started playing.
   */
  markPlayStarted(): void {
    this.isPlaying = true;
  }

  /**
   * Mark that the current story has finished playing.
   */
  markPlayEnded(now: number = Date.now()): void {
    this.isPlaying = false;
    this.lastPlayEndTime = now;
  }

  /**
   * Returns true if a story is currently playing.
   */
  getIsPlaying(): boolean {
    return this.isPlaying;
  }

  /**
   * Select the best candidate based on walking speed:
   * - Fast walk (> 4 km/h): prefer short stories (≤ 20 sec)
   * - Slow walk (< 3 km/h): prefer longer stories (≥ 30 sec)
   * - In between: no preference, pick highest scored
   *
   * Returns null if no candidates available.
   */
  selectByPace(candidates: ScoredCandidate[], speedMs: number): ScoredCandidate | null {
    if (candidates.length === 0) return null;

    if (speedMs > FAST_WALK_SPEED_MS) {
      const short = candidates.filter(
        (c) => c.duration_sec !== null && c.duration_sec <= SHORT_STORY_MAX_SEC,
      );
      if (short.length > 0) return short[0];
    }

    if (speedMs < SLOW_WALK_SPEED_MS) {
      const long = candidates.filter(
        (c) => c.duration_sec !== null && c.duration_sec >= LONG_STORY_MIN_SEC,
      );
      if (long.length > 0) return long[0];
    }

    return candidates[0];
  }

  /**
   * Reset internal state (for testing or session restart).
   */
  reset(): void {
    this.lastPlayEndTime = 0;
    this.isPlaying = false;
  }
}
