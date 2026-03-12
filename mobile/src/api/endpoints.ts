import type {
  NearbyStoryCandidate,
  NearbyStoriesParams,
  TrackListeningRequest,
  ReportStoryRequest,
  City,
  CityPOI,
  CityDownloadManifest,
  Purchase,
  VerifyPurchaseRequest,
  PurchaseStatusResponse,
  RegisterDeviceTokenRequest,
} from '@/types';
import { generatedApiClient } from './client';
import { AppApiError, normalizeGeneratedError } from './errors';
import type { operations } from './generated';

/** @deprecated Use AppApiError instead. Kept for backward compatibility. */
export const ApiRequestError = AppApiError;

function throwApiError(
  error: { error?: string; trace_id?: string } | undefined,
  fallback: string,
): never {
  throw normalizeGeneratedError(error, fallback);
}

export type FetchNearbyStoriesParams = NearbyStoriesParams;
type ListCitiesQuery = NonNullable<operations['listCities']['parameters']['query']>;
type CursorQuery = {
  cursor?: string;
  limit?: number;
};
type CursorPage<TItem> = {
  items: TItem[];
  next_cursor?: string;
  has_more: boolean;
};
const DEFAULT_CURSOR_LIMIT = 100;

function normalizeCursorPage<TItem>(data: unknown): CursorPage<TItem> {
  if (!data || typeof data !== 'object') {
    throw new Error('Cursor pagination response must be an object');
  }

  const page = data as Partial<CursorPage<TItem>>;

  if (!Array.isArray(page.items)) {
    throw new Error('Cursor pagination response is missing items array');
  }

  if (typeof page.has_more !== 'boolean') {
    throw new Error('Cursor pagination response is missing has_more boolean');
  }

  if (page.next_cursor !== undefined && typeof page.next_cursor !== 'string') {
    throw new Error('Cursor pagination response has invalid next_cursor');
  }

  return {
    items: page.items,
    next_cursor: page.next_cursor,
    has_more: page.has_more,
  };
}

async function collectCursorItems<TItem, TQuery extends CursorQuery>(
  fetchPage: (query: TQuery) => Promise<unknown>,
  baseQuery: Omit<TQuery, 'cursor'>,
): Promise<TItem[]> {
  const items: TItem[] = [];
  const seenCursors = new Set<string>();
  let cursor: string | undefined;

  while (true) {
    const page = normalizeCursorPage<TItem>(
      await fetchPage({
        ...baseQuery,
        ...(cursor ? { cursor } : {}),
        limit: baseQuery.limit ?? DEFAULT_CURSOR_LIMIT,
      } as TQuery),
    );

    items.push(...page.items);

    if (!page.has_more) {
      return items;
    }

    if (!page.next_cursor) {
      throw new Error('Cursor pagination response is missing next_cursor');
    }

    if (seenCursors.has(page.next_cursor)) {
      throw new Error(`Cursor pagination response repeated next_cursor: ${page.next_cursor}`);
    }

    seenCursors.add(page.next_cursor);
    cursor = page.next_cursor;
  }
}

export async function fetchNearbyStories(
  params: FetchNearbyStoriesParams,
): Promise<NearbyStoryCandidate[]> {
  const { data, error } = await generatedApiClient.GET('/nearby-stories', {
    params: { query: params },
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data.data ?? [];
}

export async function trackListening(request: TrackListeningRequest): Promise<void> {
  const { error } = await generatedApiClient.POST('/listenings', {
    body: request,
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
}

export async function reportStory(request: ReportStoryRequest): Promise<void> {
  const { error } = await generatedApiClient.POST('/reports', {
    body: request,
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
}

async function fetchCitiesPage(query: ListCitiesQuery = {}): Promise<unknown> {
  const { data, error } = await generatedApiClient.GET('/cities', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data;
}

export async function fetchCities(): Promise<City[]> {
  return collectCursorItems<City, ListCitiesQuery>(fetchCitiesPage, {});
}

export async function fetchCityById(id: number): Promise<City> {
  const { data, error } = await generatedApiClient.GET('/cities/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data.data;
}

export async function fetchCityDownloadManifest(
  cityId: number,
  language?: 'en' | 'ru',
): Promise<CityDownloadManifest> {
  const { data, error } = await generatedApiClient.GET('/cities/{id}/download-manifest', {
    params: {
      path: { id: cityId },
      query: { language },
    },
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data;
}

type ListPOIsQuery = NonNullable<operations['listPOIs']['parameters']['query']>;

async function fetchPOIsPage(query: ListPOIsQuery): Promise<unknown> {
  const { data, error } = await generatedApiClient.GET('/pois', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data;
}

export async function fetchCityPOIs(
  cityId: number,
  _language?: string,
): Promise<{ pois: CityPOI[]; totalStories: number }> {
  const pois = await collectCursorItems<CityPOI, ListPOIsQuery>(fetchPOIsPage, {
    city_id: cityId,
  });
  const totalStories = pois.reduce((sum, p) => sum + (p.story_count ?? 0), 0);
  return { pois, totalStories };
}

// Device token endpoints

export async function registerDeviceToken(request: RegisterDeviceTokenRequest): Promise<void> {
  const { error } = await generatedApiClient.POST('/device-tokens', {
    body: request,
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
}

export async function unregisterDeviceToken(token: string): Promise<void> {
  const { error } = await generatedApiClient.DELETE('/device-tokens', {
    body: { token },
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
}

// User account endpoints

export async function deleteAccount(): Promise<void> {
  const { error } = await generatedApiClient.DELETE('/users/me');
  if (error) {
    throwApiError(error, 'Request failed');
  }
}

export async function restoreAccount(): Promise<void> {
  const { error } = await generatedApiClient.POST('/users/me/restore');
  if (error) {
    throwApiError(error, 'Request failed');
  }
}

// Purchase endpoints

export async function verifyPurchase(request: VerifyPurchaseRequest): Promise<Purchase> {
  const { data, error } = await generatedApiClient.POST('/purchases/verify', {
    body: request,
  });
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data.data;
}

export async function fetchPurchaseStatus(): Promise<PurchaseStatusResponse> {
  const { data, error } = await generatedApiClient.GET('/purchases/status');
  if (error) {
    throwApiError(error, 'Request failed');
  }
  return data.data;
}
