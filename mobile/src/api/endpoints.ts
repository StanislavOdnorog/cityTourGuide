import type {
  NearbyStoryCandidate,
  NearbyStoriesResponse,
  TrackListeningRequest,
  ReportStoryRequest,
  City,
  CityPOI,
  CityPOIsResponse,
  CityDownloadManifest,
  Purchase,
  VerifyPurchaseRequest,
  PurchaseStatusResponse,
} from '@/types';
import apiClient from './client';

export interface FetchNearbyStoriesParams {
  lat: number;
  lng: number;
  radius?: number;
  heading?: number;
  speed?: number;
  language?: string;
  user_id?: string;
}

export async function fetchNearbyStories(
  params: FetchNearbyStoriesParams,
): Promise<NearbyStoryCandidate[]> {
  const response = await apiClient.get<NearbyStoriesResponse>('/api/v1/nearby-stories', {
    params,
  });
  return response.data.data;
}

export async function trackListening(request: TrackListeningRequest): Promise<void> {
  await apiClient.post('/api/v1/listenings', request);
}

export async function reportStory(request: ReportStoryRequest): Promise<void> {
  await apiClient.post('/api/v1/reports', request);
}

export async function fetchCities(): Promise<City[]> {
  const response = await apiClient.get<{ data: City[] }>('/api/v1/cities');
  return response.data.data;
}

export async function fetchCityById(id: number): Promise<City> {
  const response = await apiClient.get<{ data: City }>(`/api/v1/cities/${id}`);
  return response.data.data;
}

export async function fetchCityDownloadManifest(
  cityId: number,
  language?: string,
): Promise<CityDownloadManifest> {
  const response = await apiClient.get<CityDownloadManifest>(
    `/api/v1/cities/${cityId}/download-manifest`,
    { params: { language } },
  );
  return response.data;
}

export async function fetchCityPOIs(
  cityId: number,
  language?: string,
): Promise<{ pois: CityPOI[]; totalStories: number }> {
  const response = await apiClient.get<CityPOIsResponse>(`/api/v1/cities/${cityId}/pois`, {
    params: { language },
  });
  return {
    pois: response.data.data,
    totalStories: response.data.total_stories,
  };
}

// Device token endpoints

export async function registerDeviceToken(
  userId: string,
  token: string,
  platform: 'ios' | 'android',
): Promise<void> {
  await apiClient.post('/api/v1/device-tokens', {
    user_id: userId,
    token,
    platform,
  });
}

export async function unregisterDeviceToken(token: string): Promise<void> {
  await apiClient.delete('/api/v1/device-tokens', {
    data: { token },
  });
}

// User account endpoints

export async function deleteAccount(): Promise<void> {
  await apiClient.delete('/api/v1/users/me');
}

export async function restoreAccount(): Promise<void> {
  await apiClient.post('/api/v1/users/me/restore');
}

// Purchase endpoints

export async function verifyPurchase(request: VerifyPurchaseRequest): Promise<Purchase> {
  const response = await apiClient.post<{ data: Purchase }>('/api/v1/purchases/verify', request);
  return response.data.data;
}

export async function fetchPurchaseStatus(): Promise<PurchaseStatusResponse> {
  const response = await apiClient.get<{ data: PurchaseStatusResponse }>(
    '/api/v1/purchases/status',
  );
  return response.data.data;
}
