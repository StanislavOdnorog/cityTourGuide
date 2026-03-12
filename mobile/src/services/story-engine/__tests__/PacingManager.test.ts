import { PacingManager } from '../PacingManager';
import type { ScoredCandidate } from '../ScoringAlgorithm';

function makeScored(overrides: Partial<ScoredCandidate> = {}): ScoredCandidate {
  return {
    poi_id: 1,
    poi_name: 'Test POI',
    poi_lat: 41.6875,
    poi_lng: 44.8084,
    story_id: 1,
    story_text: 'A story',
    audio_url: 'https://example.com/audio.mp3',
    duration_sec: 30,
    distance_m: 100,
    score: 50,
    localScore: 50,
    ...overrides,
  };
}

describe('PacingManager', () => {
  let pacing: PacingManager;

  beforeEach(() => {
    pacing = new PacingManager();
  });

  describe('canPlayNext', () => {
    it('allows play when no story has played yet', () => {
      expect(pacing.canPlayNext()).toBe(true);
    });

    it('blocks during cooldown period', () => {
      const now = 1000000;
      pacing.markPlayStarted();
      pacing.markPlayEnded(now);
      // 30 seconds later — still in cooldown (60 sec required)
      expect(pacing.canPlayNext(now + 30_000)).toBe(false);
    });

    it('allows play after cooldown expires', () => {
      const now = 1000000;
      pacing.markPlayStarted();
      pacing.markPlayEnded(now);
      // 61 seconds later — cooldown expired
      expect(pacing.canPlayNext(now + 61_000)).toBe(true);
    });

    it('blocks while a story is playing', () => {
      pacing.markPlayStarted();
      expect(pacing.canPlayNext()).toBe(false);
    });

    it('allows at exactly 60 seconds', () => {
      const now = 1000000;
      pacing.markPlayStarted();
      pacing.markPlayEnded(now);
      expect(pacing.canPlayNext(now + 60_000)).toBe(true);
    });
  });

  describe('cooldownRemaining', () => {
    it('returns 0 when no story has played', () => {
      expect(pacing.cooldownRemaining()).toBe(0);
    });

    it('returns full cooldown while playing', () => {
      pacing.markPlayStarted();
      expect(pacing.cooldownRemaining()).toBe(60_000);
    });

    it('returns remaining time during cooldown', () => {
      const now = 1000000;
      pacing.markPlayStarted();
      pacing.markPlayEnded(now);
      expect(pacing.cooldownRemaining(now + 20_000)).toBe(40_000);
    });

    it('returns 0 after cooldown expires', () => {
      const now = 1000000;
      pacing.markPlayStarted();
      pacing.markPlayEnded(now);
      expect(pacing.cooldownRemaining(now + 70_000)).toBe(0);
    });
  });

  describe('getIsPlaying', () => {
    it('returns false initially', () => {
      expect(pacing.getIsPlaying()).toBe(false);
    });

    it('returns true after markPlayStarted', () => {
      pacing.markPlayStarted();
      expect(pacing.getIsPlaying()).toBe(true);
    });

    it('returns false after markPlayEnded', () => {
      pacing.markPlayStarted();
      pacing.markPlayEnded();
      expect(pacing.getIsPlaying()).toBe(false);
    });
  });

  describe('selectByPace', () => {
    const shortStory = makeScored({ story_id: 1, duration_sec: 15, localScore: 80 });
    const mediumStory = makeScored({ story_id: 2, duration_sec: 25, localScore: 70 });
    const longStory = makeScored({ story_id: 3, duration_sec: 40, localScore: 60 });

    it('returns null for empty candidates', () => {
      expect(pacing.selectByPace([], 1.0)).toBeNull();
    });

    it('prefers short stories during fast walking (> 4 km/h)', () => {
      const candidates = [longStory, mediumStory, shortStory];
      const result = pacing.selectByPace(candidates, 1.5); // ~5.4 km/h
      expect(result?.story_id).toBe(1); // short story
    });

    it('prefers long stories during slow walking (< 3 km/h)', () => {
      const candidates = [shortStory, mediumStory, longStory];
      const result = pacing.selectByPace(candidates, 0.5); // ~1.8 km/h
      expect(result?.story_id).toBe(3); // long story
    });

    it('picks highest scored for moderate speed', () => {
      const candidates = [shortStory, mediumStory, longStory];
      const result = pacing.selectByPace(candidates, 1.0); // ~3.6 km/h
      expect(result?.story_id).toBe(1); // highest scored (first in list)
    });

    it('falls back to highest scored if no short stories for fast walk', () => {
      const candidates = [longStory, mediumStory];
      const result = pacing.selectByPace(candidates, 1.5); // fast walk, no short stories
      expect(result?.story_id).toBe(3); // falls back to first (highest scored)
    });

    it('falls back to highest scored if no long stories for slow walk', () => {
      const candidates = [shortStory, mediumStory];
      const result = pacing.selectByPace(candidates, 0.5); // slow walk, no long stories
      expect(result?.story_id).toBe(1); // falls back to first
    });

    it('handles null duration_sec', () => {
      const nullDuration = makeScored({ story_id: 4, duration_sec: null, localScore: 90 });
      const candidates = [nullDuration, shortStory];
      // Fast walk: null duration is not ≤ 20, so only shortStory matches
      const result = pacing.selectByPace(candidates, 1.5);
      expect(result?.story_id).toBe(1);
    });
  });

  describe('reset', () => {
    it('resets all state', () => {
      pacing.markPlayStarted();
      pacing.markPlayEnded();
      pacing.reset();
      expect(pacing.canPlayNext()).toBe(true);
      expect(pacing.getIsPlaying()).toBe(false);
      expect(pacing.cooldownRemaining()).toBe(0);
    });
  });
});
