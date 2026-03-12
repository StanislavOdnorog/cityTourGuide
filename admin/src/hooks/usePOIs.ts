import { useQuery } from '@tanstack/react-query';
import { listAllPOIs } from '../api';
import type { POI, POIStatus, POIType } from '../types';

interface UsePOIsParams {
  cityId: number | null;
  status?: POIStatus;
  type?: POIType;
}

export function usePOIs({ cityId, status, type }: UsePOIsParams) {
  return useQuery({
    queryKey: ['pois', cityId, status, type],
    queryFn: () =>
      listAllPOIs({
        city_id: cityId as number,
        limit: 100,
        ...(status ? { status } : {}),
        ...(type ? { type } : {}),
      }) as Promise<POI[]>,
    enabled: cityId !== null,
    staleTime: 30_000,
  });
}
