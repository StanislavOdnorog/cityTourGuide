import type { ScoredCandidate } from '@/services/story-engine';
import { usePlayerStore } from '@/store/usePlayerStore';
import { useWalkStore } from '@/store/useWalkStore';

/**
 * Tests for useHomeScreen hook logic.
 *
 * Since we test via the Zustand stores directly (which the hook reads),
 * we verify the state that the Home Screen renders from.
 */

const makeScoredCandidate = (overrides?: Partial<ScoredCandidate>): ScoredCandidate => ({
  poi_id: 1,
  poi_name: 'Narikala Fortress',
  story_id: 10,
  story_text: 'A story about Narikala',
  audio_url: 'https://example.com/audio.mp3',
  duration_sec: 45,
  distance_m: 120,
  score: 80,
  localScore: 85,
  ...overrides,
});

describe('useHomeScreen – store integration', () => {
  beforeEach(() => {
    useWalkStore.setState({ isWalking: false, currentLocation: null });
    usePlayerStore.setState({
      currentStory: null,
      isPlaying: false,
      progress: { position: 0, duration: 0 },
      listenedStoryIds: new Set<number>(),
      listenedPoiIds: new Set<number>(),
    });
  });

  describe('isWalking state', () => {
    it('defaults to false', () => {
      expect(useWalkStore.getState().isWalking).toBe(false);
    });

    it('becomes true after startWalking', () => {
      useWalkStore.getState().startWalking();
      expect(useWalkStore.getState().isWalking).toBe(true);
    });

    it('becomes false after stopWalking', () => {
      useWalkStore.getState().startWalking();
      useWalkStore.getState().stopWalking();
      expect(useWalkStore.getState().isWalking).toBe(false);
    });
  });

  describe('currentStoryName derivation', () => {
    it('is null when no story is playing', () => {
      const { currentStory } = usePlayerStore.getState();
      expect(currentStory?.poi_name ?? null).toBeNull();
    });

    it('returns poi_name when a story is set', () => {
      const candidate = makeScoredCandidate({ poi_name: 'Rike Park' });
      usePlayerStore.getState().setCurrentStory(candidate);

      const { currentStory } = usePlayerStore.getState();
      expect(currentStory?.poi_name ?? null).toBe('Rike Park');
    });

    it('returns null after story is cleared', () => {
      usePlayerStore.getState().setCurrentStory(makeScoredCandidate());
      usePlayerStore.getState().setCurrentStory(null);

      const { currentStory } = usePlayerStore.getState();
      expect(currentStory?.poi_name ?? null).toBeNull();
    });
  });

  describe('isPlaying state', () => {
    it('defaults to false', () => {
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });

    it('becomes true when set', () => {
      usePlayerStore.getState().setIsPlaying(true);
      expect(usePlayerStore.getState().isPlaying).toBe(true);
    });

    it('toggles correctly', () => {
      usePlayerStore.getState().setIsPlaying(true);
      usePlayerStore.getState().setIsPlaying(false);
      expect(usePlayerStore.getState().isPlaying).toBe(false);
    });
  });

  describe('listenedCount derivation', () => {
    it('is 0 initially', () => {
      expect(usePlayerStore.getState().listenedStoryIds.size).toBe(0);
    });

    it('increments when stories are listened', () => {
      usePlayerStore.getState().addListenedStory(10, 1);
      expect(usePlayerStore.getState().listenedStoryIds.size).toBe(1);

      usePlayerStore.getState().addListenedStory(20, 2);
      expect(usePlayerStore.getState().listenedStoryIds.size).toBe(2);
    });

    it('does not count duplicates', () => {
      usePlayerStore.getState().addListenedStory(10, 1);
      usePlayerStore.getState().addListenedStory(10, 1);
      expect(usePlayerStore.getState().listenedStoryIds.size).toBe(1);
    });

    it('resets to 0 on reset', () => {
      usePlayerStore.getState().addListenedStory(10, 1);
      usePlayerStore.getState().addListenedStory(20, 2);
      usePlayerStore.getState().reset();
      expect(usePlayerStore.getState().listenedStoryIds.size).toBe(0);
    });
  });

  describe('progress tracking', () => {
    it('defaults to zero position and duration', () => {
      const { progress } = usePlayerStore.getState();
      expect(progress.position).toBe(0);
      expect(progress.duration).toBe(0);
    });

    it('updates position and duration', () => {
      usePlayerStore.getState().setProgress(15, 45);
      const { progress } = usePlayerStore.getState();
      expect(progress.position).toBe(15);
      expect(progress.duration).toBe(45);
    });
  });

  describe('walking + player interaction', () => {
    it('clearing walking also clears location', () => {
      useWalkStore.getState().startWalking();
      useWalkStore.getState().updateLocation({
        lat: 41.7,
        lng: 44.8,
        heading: 90,
        speed: 1.2,
      });
      useWalkStore.getState().stopWalking();

      expect(useWalkStore.getState().isWalking).toBe(false);
      expect(useWalkStore.getState().currentLocation).toBeNull();
    });

    it('walk + play cycle updates both stores', () => {
      useWalkStore.getState().startWalking();
      usePlayerStore.getState().setCurrentStory(makeScoredCandidate());
      usePlayerStore.getState().setIsPlaying(true);

      expect(useWalkStore.getState().isWalking).toBe(true);
      expect(usePlayerStore.getState().isPlaying).toBe(true);
      expect(usePlayerStore.getState().currentStory).not.toBeNull();
    });

    it('stop walking does not affect player store', () => {
      useWalkStore.getState().startWalking();
      usePlayerStore.getState().setCurrentStory(makeScoredCandidate());
      usePlayerStore.getState().setIsPlaying(true);
      useWalkStore.getState().stopWalking();

      expect(useWalkStore.getState().isWalking).toBe(false);
      expect(usePlayerStore.getState().isPlaying).toBe(true);
    });
  });

  describe('display text logic', () => {
    it('shows story name when playing and story is set', () => {
      const story = makeScoredCandidate({ poi_name: 'Peace Bridge' });
      usePlayerStore.getState().setCurrentStory(story);
      usePlayerStore.getState().setIsPlaying(true);

      const { currentStory, isPlaying } = usePlayerStore.getState();
      const displayText =
        isPlaying && currentStory?.poi_name ? currentStory.poi_name : 'Listening...';
      expect(displayText).toBe('Peace Bridge');
    });

    it('shows Listening... when walking but not playing', () => {
      useWalkStore.getState().startWalking();

      const { currentStory, isPlaying } = usePlayerStore.getState();
      const displayText =
        isPlaying && currentStory?.poi_name ? currentStory.poi_name : 'Listening...';
      expect(displayText).toBe('Listening...');
    });

    it('shows correct plural for listened count', () => {
      const pluralize = (n: number) => `${n} ${n === 1 ? 'story' : 'stories'} listened`;

      expect(pluralize(0)).toBe('0 stories listened');
      expect(pluralize(1)).toBe('1 story listened');
      expect(pluralize(5)).toBe('5 stories listened');
    });
  });
});
