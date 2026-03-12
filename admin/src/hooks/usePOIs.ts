import { useQuery } from '@tanstack/react-query';
import { listPOIs } from '../api';
import type { POI, POIStatus, POIType } from '../types';

interface UsePOIsParams {
  cityId: number | null;
  status?: POIStatus;
  type?: POIType;
}

export function usePOIs({ cityId, status, type }: UsePOIsParams) {
  return useQuery({
    queryKey: ['pois', cityId, status, type],
    queryFn: async () => {
      const allPOIs: POI[] = [];
      let page = 1;
      const perPage = 100;

      let hasMore = true;
      while (hasMore) {
        const response = await listPOIs({
          city_id: cityId,
          page,
          per_page: perPage,
          ...(status ? { status } : {}),
          ...(type ? { type } : {}),
        });
        allPOIs.push(...(response.data as POI[]));
        hasMore = allPOIs.length < response.meta.total;
        page++;
      }

      return allPOIs;
    },
    enabled: cityId !== null,
    staleTime: 30_000,
  });
}
