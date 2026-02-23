import type {
  NearbyStoryCandidate,
  NearbyStoriesResponse,
  TrackListeningRequest,
  ReportStoryRequest,
  City,
  CityPOI,
  CityPOIsResponse,
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
