import { useQuery } from '@tanstack/react-query';
import { listCities } from '../api';
import { cityQueryKeys } from './cityQueryKeys';

interface UseCitiesOptions {
  cursor?: string;
  limit?: number;
  includeDeleted?: boolean;
}

export function useCities({ cursor, limit = 20, includeDeleted = false }: UseCitiesOptions = {}) {
  return useQuery({
    queryKey: cityQueryKeys.list(cursor, limit, includeDeleted),
    queryFn: () =>
      listCities({
        limit,
        ...(cursor ? { cursor } : {}),
        ...(includeDeleted ? { include_deleted: true } : {}),
      }),
    staleTime: 60_000,
  });
}
