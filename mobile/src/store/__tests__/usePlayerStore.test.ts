import type { ScoredCandidate } from '@/services/story-engine';
import { usePlayerStore } from '../usePlayerStore';

const mockCandidate: ScoredCandidate = {
  poi_id: 1,
  poi_name: 'Narikala Fortress',
  story_id: 10,
  story_text: 'A story about...',
  audio_url: 'https://example.com/audio.mp3',
  duration_sec: 30,
  distance_m: 120,
  score: 75,
  localScore: 75,
};

describe('usePlayerStore', () => {
  beforeEach(() => {
    usePlayerStore.setState({
      currentStory: null,
      isPlaying: false,
      progress: { position: 0, duration: 0 },
      listenedStoryIds: new Set<number>(),
      listenedPoiIds: new Set<number>(),
    });
  });

  it('has correct initial state', () => {
    const state = usePlayerStore.getState();
    expect(state.currentStory).toBeNull();
    expect(state.isPlaying).toBe(false);
    expect(state.progress).toEqual({ position: 0, duration: 0 });
    expect(state.listenedStoryIds.size).toBe(0);
  });

  it('setCurrentStory stores the story', () => {
    usePlayerStore.getState().setCurrentStory(mockCandidate);
    expect(usePlayerStore.getState().currentStory).toEqual(mockCandidate);
  });

  it('setCurrentStory with null clears the story', () => {
    usePlayerStore.getState().setCurrentStory(mockCandidate);
    usePlayerStore.getState().setCurrentStory(null);
    expect(usePlayerStore.getState().currentStory).toBeNull();
  });

  it('setIsPlaying toggles playing state', () => {
    usePlayerStore.getState().setIsPlaying(true);
    expect(usePlayerStore.getState().isPlaying).toBe(true);

    usePlayerStore.getState().setIsPlaying(false);
    expect(usePlayerStore.getState().isPlaying).toBe(false);
  });

  it('setProgress updates position and duration', () => {
    usePlayerStore.getState().setProgress(15.5, 30);
    expect(usePlayerStore.getState().progress).toEqual({ position: 15.5, duration: 30 });
  });

  it('addListenedStory adds to both story and POI sets', () => {
    usePlayerStore.getState().addListenedStory(10, 1);
    usePlayerStore.getState().addListenedStory(20, 2);

    const storyIds = usePlayerStore.getState().listenedStoryIds;
    expect(storyIds.has(10)).toBe(true);
    expect(storyIds.has(20)).toBe(true);
    expect(storyIds.size).toBe(2);

    const poiIds = usePlayerStore.getState().listenedPoiIds;
    expect(poiIds.has(1)).toBe(true);
    expect(poiIds.has(2)).toBe(true);
    expect(poiIds.size).toBe(2);
  });

  it('addListenedStory deduplicates', () => {
    usePlayerStore.getState().addListenedStory(10, 1);
    usePlayerStore.getState().addListenedStory(10, 1);

    expect(usePlayerStore.getState().listenedStoryIds.size).toBe(1);
    expect(usePlayerStore.getState().listenedPoiIds.size).toBe(1);
  });

  it('reset clears all state', () => {
    usePlayerStore.getState().setCurrentStory(mockCandidate);
    usePlayerStore.getState().setIsPlaying(true);
    usePlayerStore.getState().setProgress(10, 30);
    usePlayerStore.getState().addListenedStory(10, 1);

    usePlayerStore.getState().reset();

    const state = usePlayerStore.getState();
    expect(state.currentStory).toBeNull();
    expect(state.isPlaying).toBe(false);
    expect(state.progress).toEqual({ position: 0, duration: 0 });
    expect(state.listenedStoryIds.size).toBe(0);
    expect(state.listenedPoiIds.size).toBe(0);
  });
});
