import type { operations, POI, Story } from './generated';
import { generatedApiClient } from './client';

type ListCitiesQuery = operations['listCities']['parameters']['query'];
type ListPOIsQuery = operations['listPOIs']['parameters']['query'];
type ListStoriesQuery = operations['listStories']['parameters']['query'];
type ListReportsQuery = operations['adminListReports']['parameters']['query'];
type UpdateReportStatusRequest =
  operations['adminUpdateReportStatus']['requestBody']['content']['application/json'];
type UpdatePOIRequest = operations['adminUpdatePOI']['requestBody']['content']['application/json'];
type UpdateStoryRequest =
  operations['adminUpdateStory']['requestBody']['content']['application/json'];
type CreateCityRequest =
  operations['adminCreateCity']['requestBody']['content']['application/json'];
type UpdateCityRequest =
  operations['adminUpdateCity']['requestBody']['content']['application/json'];

function getApiErrorMessage(error: { error?: string } | undefined, fallback: string): string {
  return typeof error?.error === 'string' ? error.error : fallback;
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

export async function listCities(query: ListCitiesQuery) {
  const { data, error } = await generatedApiClient.GET('/cities', {
    params: { query },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch cities'));
  }
  return data;
}

export async function listPOIs(query: ListPOIsQuery) {
  const { data, error } = await generatedApiClient.GET('/pois', {
    params: { query },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch POIs'));
  }
  return data;
}

export async function getPOI(id: number) {
  const { data, error } = await generatedApiClient.GET('/pois/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch POI'));
  }
  return data;
}

export async function listStories(query: ListStoriesQuery) {
  const { data, error } = await generatedApiClient.GET('/stories', {
    params: { query },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch stories'));
  }
  return data;
}

export async function getStory(id: number) {
  const { data, error } = await generatedApiClient.GET('/stories/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch story'));
  }
  return data;
}

export async function listReports(query: ListReportsQuery) {
  const { data, error } = await generatedApiClient.GET('/admin/reports', {
    params: { query },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch reports'));
  }
  return data;
}

export async function listReportsByPOI(id: number) {
  const { data, error } = await generatedApiClient.GET('/admin/pois/{id}/reports', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch POI reports'));
  }
  return data;
}

export async function listInflationJobsByPOI(id: number) {
  const { data, error } = await generatedApiClient.GET('/admin/pois/{id}/inflation-jobs', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to fetch inflation jobs'));
  }
  return data;
}

export async function updateReportStatus(id: number, body: UpdateReportStatusRequest) {
  const { data, error } = await generatedApiClient.PUT('/admin/reports/{id}', {
    params: { path: { id } },
    body,
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to update report'));
  }
  return data;
}

export async function updatePOI(id: number, current: POI, updates: Partial<POI>) {
  const { data, error } = await generatedApiClient.PUT('/admin/pois/{id}', {
    params: { path: { id } },
    body: toUpdatePOIRequest(current, updates),
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to update POI'));
  }
  return data;
}

export async function updateStory(id: number, current: Story, updates: Partial<Story>) {
  const { data, error } = await generatedApiClient.PUT('/admin/stories/{id}', {
    params: { path: { id } },
    body: toUpdateStoryRequest(current, updates),
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to update story'));
  }
  return data;
}

export async function triggerInflation(id: number) {
  const { data, error } = await generatedApiClient.POST('/admin/pois/{id}/inflate', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to trigger inflation'));
  }
  return data;
}

export async function createCity(body: CreateCityRequest) {
  const { data, error } = await generatedApiClient.POST('/admin/cities', {
    body,
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to create city'));
  }
  return data;
}

export async function updateCity(id: number, body: UpdateCityRequest) {
  const { data, error } = await generatedApiClient.PUT('/admin/cities/{id}', {
    params: { path: { id } },
    body,
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to update city'));
  }
  return data;
}

export async function deleteCity(id: number) {
  const { data, error } = await generatedApiClient.DELETE('/admin/cities/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(getApiErrorMessage(error, 'Failed to delete city'));
  }
  return data;
}
