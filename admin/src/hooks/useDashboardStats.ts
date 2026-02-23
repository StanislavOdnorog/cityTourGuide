import { useQueries } from '@tanstack/react-query';
import apiClient from '../api/client';
import type { City, PaginatedResponse, Report } from '../types';

interface DashboardStats {
  citiesCount: number;
  poisCount: number;
  storiesCount: number;
  reportsCount: number;
}

async function fetchReportsCount(): Promise<number> {
  const { data } = await apiClient.get<PaginatedResponse<Report>>('/api/v1/admin/reports', {
    params: { page: 1, per_page: 1 },
  });
  return data.meta.total;
}

async function fetchCitiesCount(): Promise<number> {
  const { data } = await apiClient.get<PaginatedResponse<City>>('/api/v1/cities', {
    params: { page: 1, per_page: 1 },
  });
  return data.meta.total;
}

async function fetchPOIsCount(cityIds: number[]): Promise<number> {
  if (cityIds.length === 0) return 0;
  let total = 0;
  for (const cityId of cityIds) {
    const { data } = await apiClient.get<PaginatedResponse<unknown>>('/api/v1/pois', {
      params: { city_id: cityId, page: 1, per_page: 1 },
    });
    total += data.meta.total;
  }
  return total;
}

async function fetchStoriesCountForCities(cityIds: number[]): Promise<number> {
  if (cityIds.length === 0) return 0;
  // First get all POI IDs across all cities
  let allPOIIds: number[] = [];
  for (const cityId of cityIds) {
    const { data } = await apiClient.get<PaginatedResponse<{ id: number }>>('/api/v1/pois', {
      params: { city_id: cityId, page: 1, per_page: 100 },
    });
    allPOIIds = allPOIIds.concat(data.data.map((p) => p.id));
    // If there are more pages, fetch them
    const totalPages = Math.ceil(data.meta.total / 100);
    for (let page = 2; page <= totalPages; page++) {
      const { data: pageData } = await apiClient.get<PaginatedResponse<{ id: number }>>(
        '/api/v1/pois',
        { params: { city_id: cityId, page, per_page: 100 } },
      );
      allPOIIds = allPOIIds.concat(pageData.data.map((p) => p.id));
    }
  }

  // Get story count for each POI (sample first few to get total)
  let totalStories = 0;
  for (const poiId of allPOIIds) {
    const { data } = await apiClient.get<PaginatedResponse<unknown>>('/api/v1/stories', {
      params: { poi_id: poiId, page: 1, per_page: 1 },
    });
    totalStories += data.meta.total;
  }
  return totalStories;
}

export function useDashboardStats() {
  const results = useQueries({
    queries: [
      {
        queryKey: ['cities', 'count'],
        queryFn: fetchCitiesCount,
        staleTime: 60_000,
      },
      {
        queryKey: ['cities', 'list-for-stats'],
        queryFn: async () => {
          const { data } = await apiClient.get<PaginatedResponse<City>>('/api/v1/cities', {
            params: { page: 1, per_page: 100 },
          });
          return data.data;
        },
        staleTime: 60_000,
      },
    ],
  });

  const [citiesCountResult, citiesListResult] = results;
  const cities = citiesListResult.data ?? [];
  const cityIds = cities.map((c) => c.id);

  const dependentResults = useQueries({
    queries: [
      {
        queryKey: ['pois', 'count', cityIds],
        queryFn: () => fetchPOIsCount(cityIds),
        enabled: citiesListResult.isSuccess && cityIds.length > 0,
        staleTime: 60_000,
      },
      {
        queryKey: ['stories', 'count', cityIds],
        queryFn: () => fetchStoriesCountForCities(cityIds),
        enabled: citiesListResult.isSuccess && cityIds.length > 0,
        staleTime: 60_000,
      },
      {
        queryKey: ['reports', 'count'],
        queryFn: fetchReportsCount,
        staleTime: 60_000,
      },
    ],
  });

  const [poisCountResult, storiesCountResult, reportsCountResult] = dependentResults;

  const isLoading =
    citiesCountResult.isLoading ||
    citiesListResult.isLoading ||
    poisCountResult.isLoading ||
    storiesCountResult.isLoading ||
    reportsCountResult.isLoading;

  const stats: DashboardStats = {
    citiesCount: citiesCountResult.data ?? 0,
    poisCount: poisCountResult.data ?? 0,
    storiesCount: storiesCountResult.data ?? 0,
    reportsCount: reportsCountResult.data ?? 0,
  };

  return { stats, isLoading, cities };
}
