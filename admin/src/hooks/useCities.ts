import { useQuery } from '@tanstack/react-query';
import apiClient from '../api/client';
import type { City, PaginatedResponse } from '../types';

export function useCities() {
  return useQuery({
    queryKey: ['cities', 'all'],
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedResponse<City>>('/api/v1/cities', {
        params: { page: 1, per_page: 100 },
      });
      return data.data;
    },
    staleTime: 60_000,
  });
}
