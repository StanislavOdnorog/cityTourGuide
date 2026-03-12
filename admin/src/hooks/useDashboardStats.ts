import { useQueries } from '@tanstack/react-query';
import { listCities, listPOIs, listReports, listStories } from '../api';

interface DashboardStats {
  citiesCount: number;
  poisCount: number;
  storiesCount: number;
  reportsCount: number;
}

async function fetchReportsCount(): Promise<number> {
  const response = await listReports({ page: 1, per_page: 1 });
  return response.meta.total;
}

async function fetchCitiesCount(): Promise<number> {
  const response = await listCities({ page: 1, per_page: 1 });
  return response.meta.total;
}

async function fetchPOIsCount(cityIds: number[]): Promise<number> {
  if (cityIds.length === 0) return 0;
  let total = 0;
  for (const cityId of cityIds) {
    const response = await listPOIs({ city_id: cityId, page: 1, per_page: 1 });
    total += response.meta.total;
  }
  return total;
}

async function fetchStoriesCountForCities(cityIds: number[]): Promise<number> {
  if (cityIds.length === 0) return 0;
  // First get all POI IDs across all cities
  let allPOIIds: number[] = [];
  for (const cityId of cityIds) {
    const response = await listPOIs({ city_id: cityId, page: 1, per_page: 100 });
    allPOIIds = allPOIIds.concat(response.data.map((p) => p.id));
    // If there are more pages, fetch them
    const totalPages = Math.ceil(response.meta.total / 100);
    for (let page = 2; page <= totalPages; page++) {
      const pageData = await listPOIs({ city_id: cityId, page, per_page: 100 });
      allPOIIds = allPOIIds.concat(pageData.data.map((p) => p.id));
    }
  }

  // Get story count for each POI (sample first few to get total)
  let totalStories = 0;
  for (const poiId of allPOIIds) {
    const response = await listStories({ poi_id: poiId, page: 1, per_page: 1 });
    totalStories += response.meta.total;
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
          const response = await listCities({ page: 1, per_page: 100 });
          return response.data;
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
