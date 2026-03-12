import { useQuery } from '@tanstack/react-query';
import { listAllCities } from '../api';
import type { City } from '../types';

export function useCities() {
  return useQuery({
    queryKey: ['cities', 'all'],
    queryFn: () => listAllCities({ limit: 100 }) as Promise<City[]>,
    staleTime: 60_000,
  });
}
