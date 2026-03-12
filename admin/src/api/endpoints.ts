import type { operations, POI, Story } from './generated';
import { generatedApiClient } from './client';
import { ApiRequestError, createApiRequestError } from './errors';

type ListCitiesQuery = NonNullable<operations['adminListCities']['parameters']['query']>;
type ListCitiesResponse = operations['adminListCities']['responses']['200']['content']['application/json'];
type ListPOIsQuery = NonNullable<operations['adminListPOIs']['parameters']['query']>;
type ListStoriesQuery = NonNullable<operations['adminListStories']['parameters']['query']>;
type ListAuditLogsQuery = NonNullable<operations['adminListAuditLogs']['parameters']['query']>;
type ListAuditLogsResponse =
  operations['adminListAuditLogs']['responses']['200']['content']['application/json'];
type ListReportsQuery = NonNullable<operations['adminListReports']['parameters']['query']>;
type ListReportsResponse =
  operations['adminListReports']['responses']['200']['content']['application/json'];
type AdminStatsResponse = operations['adminGetStats']['responses']['200']['content']['application/json'];
type CursorQuery = {
  cursor?: string;
  limit?: number;
};
type CursorResponse<TItem> = {
  items: TItem[];
  next_cursor: string;
  has_more: boolean;
};
type UpdateReportStatusRequest =
  operations['adminUpdateReportStatus']['requestBody']['content']['application/json'];
type DisableReportedStoryResponse =
  operations['adminDisableReportedStory']['responses']['200']['content']['application/json'];
type UpdatePOIRequest = operations['adminUpdatePOI']['requestBody']['content']['application/json'];
type UpdateStoryRequest =
  operations['adminUpdateStory']['requestBody']['content']['application/json'];
type CreateCityRequest =
  operations['adminCreateCity']['requestBody']['content']['application/json'];
type UpdateCityRequest =
  operations['adminUpdateCity']['requestBody']['content']['application/json'];
type RestoreCityResponse = operations['adminRestoreCity']['responses']['200']['content']['application/json'];

export { ApiRequestError };

function throwApiError(error: unknown, fallback: string, status?: number): never {
  throw createApiRequestError(error, fallback, status);
}

async function collectCursorItems<TItem, TQuery extends CursorQuery>(
  fetchPage: (query: TQuery) => Promise<CursorResponse<TItem>>,
  baseQuery: Omit<TQuery, 'cursor'>,
): Promise<TItem[]> {
  const items: TItem[] = [];
  let cursor: string | undefined;

  while (true) {
    const page = await fetchPage({
      ...baseQuery,
      ...(cursor ? { cursor } : {}),
      limit: baseQuery.limit ?? 100,
    } as TQuery);

    items.push(...page.items);

    if (!page.has_more) {
      return items;
    }

    if (!page.next_cursor) {
      throw new Error('Cursor pagination response is missing next_cursor');
    }

    cursor = page.next_cursor;
  }
}

function toUpdatePOIRequest(current: POI, updates: Partial<POI>): UpdatePOIRequest {
  const next = { ...current, ...updates };
  return {
    city_id: next.city_id,
    name: next.name,
    name_ru: next.name_ru ?? undefined,
    lat: next.lat,
    lng: next.lng,
    type: next.type,
    tags: next.tags ?? undefined,
    address: next.address ?? undefined,
    interest_score: next.interest_score,
    status: next.status,
  };
}

function toUpdateStoryRequest(current: Story, updates: Partial<Story>): UpdateStoryRequest {
  const next = { ...current, ...updates };
  return {
    poi_id: next.poi_id,
    language: next.language,
    text: next.text,
    audio_url: next.audio_url ?? undefined,
    duration_sec: next.duration_sec ?? undefined,
    layer_type: next.layer_type,
    order_index: next.order_index,
    is_inflation: next.is_inflation,
    confidence: next.confidence,
    sources: next.sources ?? undefined,
    status: next.status,
  };
}

export async function listCities(query: ListCitiesQuery = {}): Promise<ListCitiesResponse> {
  const { data, error } = await generatedApiClient.GET('/admin/cities', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch cities');
  }
  return data as ListCitiesResponse;
}

export async function listAllCities(query: ListCitiesQuery = {}) {
  return collectCursorItems(listCities, query);
}

export async function listPOIs(query: ListPOIsQuery) {
  const { data, error } = await generatedApiClient.GET('/admin/pois', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch POIs');
  }
  return data;
}

export async function listAllPOIs(query: ListPOIsQuery) {
  return collectCursorItems(listPOIs, query);
}

export async function getPOI(id: number) {
  const { data, error } = await generatedApiClient.GET('/pois/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch POI');
  }
  return data;
}

export async function listStories(query: ListStoriesQuery) {
  const { data, error } = await generatedApiClient.GET('/admin/stories', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch stories');
  }
  return data;
}

export async function listAllStories(query: ListStoriesQuery) {
  return collectCursorItems(listStories, query);
}

export async function getAdminStats() {
  const { data, error } = await generatedApiClient.GET('/admin/stats');
  if (error) {
    throwApiError(error, 'Failed to fetch admin stats');
  }
  return data as AdminStatsResponse;
}

export async function listAuditLogs(query: ListAuditLogsQuery = {}): Promise<ListAuditLogsResponse> {
  const { data, error } = await generatedApiClient.GET('/admin/audit-logs', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch audit logs');
  }
  return data as ListAuditLogsResponse;
}

export async function listAllAuditLogs(query: ListAuditLogsQuery = {}) {
  return collectCursorItems(listAuditLogs, query);
}

export async function getStory(id: number) {
  const { data, error, response } = await generatedApiClient.GET('/stories/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch story', response.status);
  }
  return data;
}

export async function listReports(query: ListReportsQuery = {}): Promise<ListReportsResponse> {
  const { data, error } = await generatedApiClient.GET('/admin/reports', {
    params: { query },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch reports');
  }
  return data as ListReportsResponse;
}

export async function listReportsPageItems(query: ListReportsQuery = {}) {
  return (await listReports(query)).items;
}

export async function listAllReports(query: ListReportsQuery = {}) {
  return collectCursorItems(listReports, query);
}

export async function listReportsByPOI(id: number) {
  const { data, error } = await generatedApiClient.GET('/admin/pois/{id}/reports', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch POI reports');
  }
  return data;
}

export async function listInflationJobsByPOI(id: number) {
  const { data, error } = await generatedApiClient.GET('/admin/pois/{id}/inflation-jobs', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to fetch inflation jobs');
  }
  return data;
}

export async function updateReportStatus(id: number, body: UpdateReportStatusRequest) {
  const { data, error } = await generatedApiClient.PUT('/admin/reports/{id}', {
    params: { path: { id } },
    body,
  });
  if (error) {
    throwApiError(error, 'Failed to update report');
  }
  return data;
}

export async function disableReportedStory(id: number) {
  const { data, error, response } = await generatedApiClient.POST('/admin/reports/{id}/disable-story', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to moderate report', response.status);
  }
  return data as DisableReportedStoryResponse;
}

export async function updatePOI(id: number, current: POI, updates: Partial<POI>) {
  const { data, error } = await generatedApiClient.PUT('/admin/pois/{id}', {
    params: { path: { id } },
    body: toUpdatePOIRequest(current, updates),
  });
  if (error) {
    throwApiError(error, 'Failed to update POI');
  }
  return data;
}

export async function updateStory(id: number, current: Story, updates: Partial<Story>) {
  const { data, error } = await generatedApiClient.PUT('/admin/stories/{id}', {
    params: { path: { id } },
    body: toUpdateStoryRequest(current, updates),
  });
  if (error) {
    throwApiError(error, 'Failed to update story');
  }
  return data;
}

export async function triggerInflation(id: number) {
  const { data, error } = await generatedApiClient.POST('/admin/pois/{id}/inflate', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to trigger inflation');
  }
  return data;
}

export async function createCity(body: CreateCityRequest) {
  const { data, error, response } = await generatedApiClient.POST('/admin/cities', {
    body,
  });
  if (error) {
    throwApiError(error, 'Failed to create city', response.status);
  }
  return data;
}

export async function updateCity(id: number, body: UpdateCityRequest) {
  const { data, error, response } = await generatedApiClient.PUT('/admin/cities/{id}', {
    params: { path: { id } },
    body,
  });
  if (error) {
    throwApiError(error, 'Failed to update city', response.status);
  }
  return data;
}

export async function deleteCity(id: number) {
  const { data, error, response } = await generatedApiClient.DELETE('/admin/cities/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to delete city', response.status);
  }
  return data;
}

export async function restoreCity(id: number): Promise<RestoreCityResponse> {
  const { data, error, response } = await generatedApiClient.POST('/admin/cities/{id}/restore', {
    params: { path: { id } },
  });
  if (error) {
    throwApiError(error, 'Failed to restore city', response.status);
  }
  return data as RestoreCityResponse;
}
