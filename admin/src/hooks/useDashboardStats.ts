import { useQuery } from '@tanstack/react-query';
import { listAllCities, listAllPOIs, listAllReports, listAllStories } from '../api';
import type { City } from '../types';

interface DashboardStats {
  citiesCount: number;
  poisCount: number;
  storiesCount: number;
  reportsCount: number;
}

async function fetchPOIsCount(cityIds: number[]): Promise<number> {
  if (cityIds.length === 0) return 0;
  const poiLists = await Promise.all(
    cityIds.map((cityId) =>
      listAllPOIs({
        city_id: cityId,
        limit: 100,
      }),
    ),
  );
  return poiLists.reduce((total, pois) => total + pois.length, 0);
}

async function fetchStoriesCountForCities(cityIds: number[]): Promise<number> {
  if (cityIds.length === 0) return 0;
  const poiLists = await Promise.all(
    cityIds.map((cityId) =>
      listAllPOIs({
        city_id: cityId,
        limit: 100,
      }),
    ),
  );
  const poiIds = poiLists.flatMap((pois) => pois.map((poi) => poi.id));
  if (poiIds.length === 0) return 0;

  const storyLists = await Promise.all(
    poiIds.map((poiId) =>
      listAllStories({
        poi_id: poiId,
        limit: 100,
      }),
    ),
  );

  return storyLists.reduce((total, stories) => total + stories.length, 0);
}

export function useDashboardStats() {
  const citiesQuery = useQuery({
    queryKey: ['cities', 'list-for-stats'],
    queryFn: () => listAllCities({ limit: 100 }) as Promise<City[]>,
    staleTime: 60_000,
  });

  const cities = citiesQuery.data ?? [];
  const cityIds = cities.map((c) => c.id);

  const poisCountQuery = useQuery({
    queryKey: ['pois', 'count', cityIds],
    queryFn: () => fetchPOIsCount(cityIds),
    enabled: citiesQuery.isSuccess,
    staleTime: 60_000,
  });

  const storiesCountQuery = useQuery({
    queryKey: ['stories', 'count', cityIds],
    queryFn: () => fetchStoriesCountForCities(cityIds),
    enabled: citiesQuery.isSuccess,
    staleTime: 60_000,
  });

  const reportsQuery = useQuery({
    queryKey: ['reports', 'count'],
    queryFn: async () => (await listAllReports({ limit: 100 })).length,
    staleTime: 60_000,
  });

  const isLoading =
    citiesQuery.isLoading ||
    poisCountQuery.isLoading ||
    storiesCountQuery.isLoading ||
    reportsQuery.isLoading;

  const stats: DashboardStats = {
    citiesCount: cities.length,
    poisCount: poisCountQuery.data ?? 0,
    storiesCount: storiesCountQuery.data ?? 0,
    reportsCount: reportsQuery.data ?? 0,
  };

  return { stats, isLoading, cities };
}
