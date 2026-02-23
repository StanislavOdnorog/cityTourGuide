import { useQuery } from '@tanstack/react-query';
import apiClient from '../api/client';
import type { POI, POIStatus, POIType, PaginatedResponse } from '../types';

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
        const { data } = await apiClient.get<PaginatedResponse<POI>>('/api/v1/pois', {
          params: {
            city_id: cityId,
            page,
            per_page: perPage,
            ...(status && { status }),
            ...(type && { type }),
          },
        });
        allPOIs.push(...data.data);
        hasMore = allPOIs.length < data.meta.total;
        page++;
      }

      return allPOIs;
    },
    enabled: cityId !== null,
    staleTime: 30_000,
  });
}
