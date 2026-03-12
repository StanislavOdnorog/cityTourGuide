import { useQuery } from '@tanstack/react-query';
import { getAdminStats } from '../api';
import { useCities } from './useCities';

export interface DashboardStats {
  citiesCount: number;
  poisCount: number;
  storiesCount: number;
  reportsCount: number;
  newReportsCount: number;
}

export function useDashboardStats() {
  const citiesQuery = useCities();
  const statsQuery = useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: getAdminStats,
    staleTime: 60_000,
  });

  const isLoading = citiesQuery.isLoading || statsQuery.isLoading;
  const statsData = statsQuery.data?.data;

  const stats: DashboardStats | null = statsData
    ? {
        citiesCount: statsData.cities_count,
        poisCount: statsData.pois_count,
        storiesCount: statsData.stories_count,
        reportsCount: statsData.reports_count,
        newReportsCount: statsData.new_reports_count,
      }
    : null;

  return {
    stats,
    isLoading,
    isError: statsQuery.isError,
    error: statsQuery.error,
    cities: citiesQuery.data?.items ?? [],
  };
}
