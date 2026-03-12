import type {
  NearbyStoryCandidate,
  NearbyStoriesParams,
  TrackListeningRequest,
  ReportStoryRequest,
  City,
  CityPOI,
  CityPOIsResponse,
  CityDownloadManifest,
  Purchase,
  VerifyPurchaseRequest,
  PurchaseStatusResponse,
  RegisterDeviceTokenRequest,
} from '@/types';
import apiClient, { generatedApiClient } from './client';

export type FetchNearbyStoriesParams = NearbyStoriesParams;

export async function fetchNearbyStories(
  params: FetchNearbyStoriesParams,
): Promise<NearbyStoryCandidate[]> {
  const { data, error } = await generatedApiClient.GET('/nearby-stories', {
    params: { query: params },
  });
  if (error) {
    throw new Error(error.error);
  }
  return data.data ?? [];
}

export async function trackListening(request: TrackListeningRequest): Promise<void> {
  const { error } = await generatedApiClient.POST('/listenings', {
    body: request,
  });
  if (error) {
    throw new Error(error.error);
  }
}

export async function reportStory(request: ReportStoryRequest): Promise<void> {
  const { error } = await generatedApiClient.POST('/reports', {
    body: request,
  });
  if (error) {
    throw new Error(error.error);
  }
}

export async function fetchCities(): Promise<City[]> {
  const { data, error } = await generatedApiClient.GET('/cities');
  if (error) {
    throw new Error(error.error);
  }
  return data.data;
}

export async function fetchCityById(id: number): Promise<City> {
  const { data, error } = await generatedApiClient.GET('/cities/{id}', {
    params: { path: { id } },
  });
  if (error) {
    throw new Error(error.error);
  }
  return data.data;
}

export async function fetchCityDownloadManifest(
  cityId: number,
  language?: string,
): Promise<CityDownloadManifest> {
  const { data, error } = await generatedApiClient.GET('/cities/{id}/download-manifest', {
    params: {
      path: { id: cityId },
      query: { language },
    },
  });
  if (error) {
    throw new Error(error.error);
  }
  return data;
}

export async function fetchCityPOIs(
  cityId: number,
  language?: string,
): Promise<{ pois: CityPOI[]; totalStories: number }> {
  const response = await apiClient.get<CityPOIsResponse>(`/cities/${cityId}/pois`, {
    params: { language },
  });
  return {
    pois: response.data.data,
    totalStories: response.data.total_stories,
  };
}

// Device token endpoints

export async function registerDeviceToken(request: RegisterDeviceTokenRequest): Promise<void> {
  const { error } = await generatedApiClient.POST('/device-tokens', {
    body: request,
  });
  if (error) {
    throw new Error(error.error);
  }
}

export async function unregisterDeviceToken(token: string): Promise<void> {
  const { error } = await generatedApiClient.DELETE('/device-tokens', {
    body: { token },
  });
  if (error) {
    throw new Error(error.error);
  }
}

// User account endpoints

export async function deleteAccount(): Promise<void> {
  const { error } = await generatedApiClient.DELETE('/users/me');
  if (error) {
    throw new Error(error.error);
  }
}

export async function restoreAccount(): Promise<void> {
  const { error } = await generatedApiClient.POST('/users/me/restore');
  if (error) {
    throw new Error(error.error);
  }
}

// Purchase endpoints

export async function verifyPurchase(request: VerifyPurchaseRequest): Promise<Purchase> {
  const { data, error } = await generatedApiClient.POST('/purchases/verify', {
    body: request,
  });
  if (error) {
    throw new Error(error.error);
  }
  return data.data;
}

export async function fetchPurchaseStatus(): Promise<PurchaseStatusResponse> {
  const { data, error } = await generatedApiClient.GET('/purchases/status');
  if (error) {
    throw new Error(error.error);
  }
  return data.data;
}
