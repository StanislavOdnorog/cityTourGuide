import { useQuery } from '@tanstack/react-query';
import { listCities } from '../api';
import type { City } from '../types';

export function useCities() {
  return useQuery({
    queryKey: ['cities', 'all'],
    queryFn: async () => {
      const response = await listCities({ page: 1, per_page: 100 });
      return response.data as City[];
    },
    staleTime: 60_000,
  });
}
