const mockGeneratedGet = jest.fn();
const mockGeneratedPost = jest.fn();

jest.mock('../client', () => ({
  __esModule: true,
  default: {},
  generatedApiClient: {
    GET: mockGeneratedGet,
    POST: mockGeneratedPost,
    DELETE: jest.fn(),
  },
}));

import {
  ApiRequestError,
  fetchCities,
  fetchCityPOIs,
  fetchNearbyStories,
  reportStory,
} from '../endpoints';
import { AppApiError } from '../errors';

function makeCity(id: number) {
  return {
    id,
    name: `City ${id}`,
    country: 'US',
    lat: 1,
    lng: 2,
    radius_km: 3,
    subscription_tier: 'free',
    status: 'active',
  } as unknown as ReturnType<typeof makeCity>;
}

describe('API endpoints cursor pagination', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('fetches a single cities page', async () => {
    const city = makeCity(1);
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: [city], next_cursor: '', has_more: false },
      error: undefined,
    });

    await expect(fetchCities()).resolves.toEqual([city]);
    expect(mockGeneratedGet).toHaveBeenCalledTimes(1);
    expect(mockGeneratedGet).toHaveBeenCalledWith('/cities', {
      params: { query: { limit: 100 } },
    });
  });

  it('keeps following cursor pages until has_more is false, even across empty pages', async () => {
    const city1 = makeCity(1);
    const city2 = makeCity(2);

    mockGeneratedGet
      .mockResolvedValueOnce({
        data: { items: [city1], next_cursor: 'cursor-1', has_more: true },
        error: undefined,
      })
      .mockResolvedValueOnce({
        data: { items: [], next_cursor: 'cursor-2', has_more: true },
        error: undefined,
      })
      .mockResolvedValueOnce({
        data: { items: [city2], next_cursor: '', has_more: false },
        error: undefined,
      });

    await expect(fetchCities()).resolves.toEqual([city1, city2]);
    expect(mockGeneratedGet).toHaveBeenCalledTimes(3);
    expect(mockGeneratedGet).toHaveBeenNthCalledWith(1, '/cities', {
      params: { query: { limit: 100 } },
    });
    expect(mockGeneratedGet).toHaveBeenNthCalledWith(2, '/cities', {
      params: { query: { cursor: 'cursor-1', limit: 100 } },
    });
    expect(mockGeneratedGet).toHaveBeenNthCalledWith(3, '/cities', {
      params: { query: { cursor: 'cursor-2', limit: 100 } },
    });
  });

  it('returns an empty list when the first page is empty and has_more is false', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: [], next_cursor: '', has_more: false },
      error: undefined,
    });

    await expect(fetchCities()).resolves.toEqual([]);
    expect(mockGeneratedGet).toHaveBeenCalledTimes(1);
  });

  it('throws when has_more is true but next_cursor is missing', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: [makeCity(1)], has_more: true },
      error: undefined,
    });

    await expect(fetchCities()).rejects.toThrow(
      'Cursor pagination response is missing next_cursor',
    );
  });

  it('throws on malformed cursor pages without an items array', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: { has_more: false, next_cursor: '' },
      error: undefined,
    });

    await expect(fetchCities()).rejects.toThrow(
      'Cursor pagination response is missing items array',
    );
  });

  it('throws when the backend repeats next_cursor to avoid an infinite loop', async () => {
    mockGeneratedGet
      .mockResolvedValueOnce({
        data: { items: [makeCity(1)], next_cursor: 'repeat-me', has_more: true },
        error: undefined,
      })
      .mockResolvedValueOnce({
        data: { items: [], next_cursor: 'repeat-me', has_more: true },
        error: undefined,
      });

    await expect(fetchCities()).rejects.toThrow(
      'Cursor pagination response repeated next_cursor: repeat-me',
    );
  });
});

describe('city visibility boundary', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('fetchCities calls the public /cities endpoint, not /admin/cities', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: [makeCity(1)], next_cursor: '', has_more: false },
      error: undefined,
    });

    await fetchCities();

    expect(mockGeneratedGet).toHaveBeenCalledWith('/cities', expect.anything());
    // Ensure no call went to admin endpoint.
    const paths = mockGeneratedGet.mock.calls.map((call: unknown[]) => call[0]);
    expect(paths.every((p: string) => !p.includes('admin'))).toBe(true);
  });

  it('public endpoint only returns what the server sends — no client-side filtering', async () => {
    // The mobile client trusts the server to filter inactive cities.
    // If the server returns only active cities, the client returns exactly those.
    const activeCities = [makeCity(1), makeCity(2)];
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: activeCities, next_cursor: '', has_more: false },
      error: undefined,
    });

    const result = await fetchCities();
    expect(result).toEqual(activeCities);
    expect(result).toHaveLength(2);
  });

  it('fetchCities collects across pages using only the public endpoint', async () => {
    mockGeneratedGet
      .mockResolvedValueOnce({
        data: { items: [makeCity(1)], next_cursor: 'c1', has_more: true },
        error: undefined,
      })
      .mockResolvedValueOnce({
        data: { items: [makeCity(2)], next_cursor: '', has_more: false },
        error: undefined,
      });

    const result = await fetchCities();
    expect(result).toHaveLength(2);

    // Both calls must target the public /cities endpoint.
    expect(mockGeneratedGet).toHaveBeenCalledTimes(2);
    expect(mockGeneratedGet).toHaveBeenNthCalledWith(1, '/cities', {
      params: { query: { limit: 100 } },
    });
    expect(mockGeneratedGet).toHaveBeenNthCalledWith(2, '/cities', {
      params: { query: { cursor: 'c1', limit: 100 } },
    });
  });
});

describe('ApiRequestError backward compatibility', () => {
  it('ApiRequestError is aliased to AppApiError', () => {
    expect(ApiRequestError).toBe(AppApiError);
  });

  it('instances pass instanceof checks for both names', () => {
    const err = new ApiRequestError({ category: 'unknown', message: 'test' });
    expect(err).toBeInstanceOf(AppApiError);
    expect(err).toBeInstanceOf(ApiRequestError);
  });
});

describe('trace ID propagation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('includes traceId on ApiRequestError when backend returns trace_id', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'no stories nearby', trace_id: 'mobile-trace-1' },
    });

    try {
      await fetchNearbyStories({ lat: 0, lng: 0, radius: 100, language: 'en' });
      fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as AppApiError;
      expect(apiErr.message).toBe('no stories nearby');
      expect(apiErr.traceId).toBe('mobile-trace-1');
    }
  });

  it('omits traceId when backend does not return trace_id', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'server error' },
    });

    try {
      await fetchNearbyStories({ lat: 0, lng: 0, radius: 100, language: 'en' });
      fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      expect((err as AppApiError).traceId).toBeUndefined();
    }
  });

  it('propagates traceId from POST endpoints', async () => {
    mockGeneratedPost.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'report failed', trace_id: 'post-trace-7' },
    });

    try {
      await reportStory({ story_id: 1, user_id: 'u1', type: 'wrong_fact' });
      fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      expect((err as AppApiError).traceId).toBe('post-trace-7');
    }
  });

  it('uses fallback message when error.error is missing', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: undefined,
      error: { trace_id: 'fallback-trace' },
    });

    try {
      await fetchNearbyStories({ lat: 0, lng: 0, radius: 100, language: 'en' });
      fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as AppApiError;
      expect(apiErr.message).toBe('Request failed');
      expect(apiErr.traceId).toBe('fallback-trace');
    }
  });
});

describe('fetchCityPOIs', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  function makePOI(id: number, storyCount = 0) {
    return {
      id,
      city_id: 1,
      name: `POI ${id}`,
      lat: 41.0,
      lng: 44.0,
      type: 'monument',
      interest_score: 80,
      status: 'active',
      story_count: storyCount,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    } as unknown as ReturnType<typeof makePOI>;
  }

  it('calls the generated /pois endpoint with city_id query param', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: [makePOI(1, 2)], next_cursor: '', has_more: false },
      error: undefined,
    });

    const result = await fetchCityPOIs(42);

    expect(mockGeneratedGet).toHaveBeenCalledWith('/pois', {
      params: { query: { city_id: 42, limit: 100 } },
    });
    expect(result.pois).toHaveLength(1);
    expect(result.pois[0].id).toBe(1);
  });

  it('returns the correct public shape with totalStories computed from story_count', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: {
        items: [makePOI(1, 3), makePOI(2, 2)],
        next_cursor: '',
        has_more: false,
      },
      error: undefined,
    });

    const result = await fetchCityPOIs(1);

    expect(result.pois).toHaveLength(2);
    expect(result.totalStories).toBe(5);
  });

  it('collects POIs across multiple cursor pages', async () => {
    mockGeneratedGet
      .mockResolvedValueOnce({
        data: { items: [makePOI(1, 1)], next_cursor: 'c1', has_more: true },
        error: undefined,
      })
      .mockResolvedValueOnce({
        data: { items: [makePOI(2, 4)], next_cursor: '', has_more: false },
        error: undefined,
      });

    const result = await fetchCityPOIs(1);

    expect(result.pois).toHaveLength(2);
    expect(result.totalStories).toBe(5);
    expect(mockGeneratedGet).toHaveBeenCalledTimes(2);
  });

  it('returns zero totalStories when POIs have no story_count', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: { items: [makePOI(1), makePOI(2)], next_cursor: '', has_more: false },
      error: undefined,
    });

    const result = await fetchCityPOIs(1);
    expect(result.totalStories).toBe(0);
  });

  it('throws ApiRequestError on error response', async () => {
    mockGeneratedGet.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'invalid city_id', trace_id: 'trace-poi' },
    });

    try {
      await fetchCityPOIs(999);
      fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      expect((err as AppApiError).message).toBe('invalid city_id');
    }
  });
});
