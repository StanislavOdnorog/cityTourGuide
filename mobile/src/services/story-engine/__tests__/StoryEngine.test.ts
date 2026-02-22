import { StoryEngine } from '../StoryEngine';
import type { StoryFetcher, StoryPlayer, ListeningTracker, LocationUpdate } from '../StoryEngine';
import type { NearbyStoryCandidate } from '@/types';

function makeCandidate(overrides: Partial<NearbyStoryCandidate> = {}): NearbyStoryCandidate {
  return {
    poi_id: 1,
    poi_name: 'Test POI',
    story_id: 1,
    story_text: 'A story',
    audio_url: 'https://example.com/audio.mp3',
    duration_sec: 30,
    distance_m: 100,
    score: 50,
    ...overrides,
  };
}

function makeLocation(overrides: Partial<LocationUpdate> = {}): LocationUpdate {
  return {
    lat: 41.7151,
    lng: 44.8271,
    heading: 90,
    speed: 1.0,
    ...overrides,
  };
}

function createMocks() {
  const playedCandidates: NearbyStoryCandidate[] = [];
  const trackedListenings: Array<{ storyId: number; completed: boolean }> = [];
  const listenedIds = new Set<number>();

  const fetcher: StoryFetcher = {
    fetchNearbyStories: jest.fn().mockResolvedValue([]),
  };

  const player: StoryPlayer = {
    play: jest.fn((candidate) => playedCandidates.push(candidate)),
  };

  const tracker: ListeningTracker = {
    trackListening: jest.fn((storyId, completed) => {
      trackedListenings.push({ storyId, completed });
      listenedIds.add(storyId);
    }),
    getListenedStoryIds: jest.fn(() => listenedIds),
  };

  return { fetcher, player, tracker, playedCandidates, trackedListenings, listenedIds };
}

describe('StoryEngine', () => {
  it('does not trigger when inactive', async () => {
    const { fetcher, player, tracker } = createMocks();
    const engine = new StoryEngine(fetcher, player, tracker);
    // Not started
    const result = await engine.onLocationUpdate(makeLocation());
    expect(result).toBeNull();
    expect(fetcher.fetchNearbyStories).not.toHaveBeenCalled();
  });

  it('fetches stories on location update when active', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 1, score: 50 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    const result = await engine.onLocationUpdate(makeLocation());
    expect(result).not.toBeNull();
    expect(result?.story_id).toBe(1);
    expect(fetcher.fetchNearbyStories).toHaveBeenCalledTimes(1);
  });

  it('plays the selected story', async () => {
    const { fetcher, player, tracker, playedCandidates } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 5, poi_name: 'Cathedral' }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    await engine.onLocationUpdate(makeLocation());
    expect(player.play).toHaveBeenCalledTimes(1);
    expect(playedCandidates[0].story_id).toBe(5);
    expect(playedCandidates[0].poi_name).toBe('Cathedral');
  });

  it('does not play while current story is playing (pacing)', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 1, score: 50 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    // First location update → plays story
    await engine.onLocationUpdate(makeLocation());
    expect(player.play).toHaveBeenCalledTimes(1);

    // Second location update → blocked by pacing (story still playing)
    const result2 = await engine.onLocationUpdate(makeLocation());
    expect(result2).toBeNull();
    expect(player.play).toHaveBeenCalledTimes(1);
  });

  it('tracks listening when story completes', async () => {
    const { fetcher, player, tracker, trackedListenings } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 7 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    await engine.onLocationUpdate(makeLocation());
    engine.onStoryCompleted(true);

    expect(tracker.trackListening).toHaveBeenCalledWith(7, true);
    expect(trackedListenings[0]).toEqual({ storyId: 7, completed: true });
  });

  it('excludes listened stories from candidates', async () => {
    const { fetcher, player, tracker, listenedIds } = createMocks();
    listenedIds.add(1);
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 1, score: 80 }),
      makeCandidate({ story_id: 2, score: 60 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    const result = await engine.onLocationUpdate(makeLocation());
    expect(result?.story_id).toBe(2);
  });

  it('returns null when no candidates from API', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    const result = await engine.onLocationUpdate(makeLocation());
    expect(result).toBeNull();
    expect(player.play).not.toHaveBeenCalled();
  });

  it('returns null when all candidates are listened', async () => {
    const { fetcher, player, tracker, listenedIds } = createMocks();
    listenedIds.add(1);
    listenedIds.add(2);
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 1 }),
      makeCandidate({ story_id: 2 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    const result = await engine.onLocationUpdate(makeLocation());
    expect(result).toBeNull();
  });

  it('passes config params to fetcher', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([]);
    const engine = new StoryEngine(fetcher, player, tracker, {
      radiusM: 200,
      language: 'ru',
      userId: 'user-123',
    });
    engine.start();

    const loc = makeLocation({ lat: 41.7, lng: 44.8, heading: 45, speed: 1.5 });
    await engine.onLocationUpdate(loc);

    expect(fetcher.fetchNearbyStories).toHaveBeenCalledWith({
      lat: 41.7,
      lng: 44.8,
      radius: 200,
      heading: 45,
      speed: 1.5,
      language: 'ru',
      user_id: 'user-123',
    });
  });

  it('sets currentStory while playing and clears on completion', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 3 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    expect(engine.getCurrentStory()).toBeNull();

    await engine.onLocationUpdate(makeLocation());
    expect(engine.getCurrentStory()?.story_id).toBe(3);

    engine.onStoryCompleted(true);
    expect(engine.getCurrentStory()).toBeNull();
  });

  it('start and stop toggle isActive', () => {
    const { fetcher, player, tracker } = createMocks();
    const engine = new StoryEngine(fetcher, player, tracker);

    expect(engine.getIsActive()).toBe(false);
    engine.start();
    expect(engine.getIsActive()).toBe(true);
    engine.stop();
    expect(engine.getIsActive()).toBe(false);
  });

  it('does not trigger after stop', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 1 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();
    engine.stop();

    const result = await engine.onLocationUpdate(makeLocation());
    expect(result).toBeNull();
    expect(fetcher.fetchNearbyStories).not.toHaveBeenCalled();
  });

  it('reset clears all state', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([
      makeCandidate({ story_id: 1 }),
    ]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();
    await engine.onLocationUpdate(makeLocation());

    engine.reset();
    expect(engine.getIsActive()).toBe(false);
    expect(engine.getCurrentStory()).toBeNull();
    expect(engine.getPacingManager().canPlayNext()).toBe(true);
  });

  it('updateConfig changes engine parameters', async () => {
    const { fetcher, player, tracker } = createMocks();
    (fetcher.fetchNearbyStories as jest.Mock).mockResolvedValue([]);
    const engine = new StoryEngine(fetcher, player, tracker);
    engine.start();

    engine.updateConfig({ language: 'ru', radiusM: 300 });
    await engine.onLocationUpdate(makeLocation());

    expect(fetcher.fetchNearbyStories).toHaveBeenCalledWith(
      expect.objectContaining({
        language: 'ru',
        radius: 300,
      }),
    );
  });
});
