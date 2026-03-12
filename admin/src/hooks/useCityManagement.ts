import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createCity, updateCity } from '../api';
import type { City } from '../types';
import type { operations } from '../types';

type CreateCityRequest =
  operations['adminCreateCity']['requestBody']['content']['application/json'];
type UpdateCityRequest =
  operations['adminUpdateCity']['requestBody']['content']['application/json'];

export function useCityManagement() {
  const queryClient = useQueryClient();

  const createCityMutation = useMutation({
    mutationFn: (body: CreateCityRequest) => createCity(body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cities'] });
    },
  });

  const updateCityMutation = useMutation({
    mutationFn: ({ id, body }: { id: number; body: UpdateCityRequest }) => updateCity(id, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cities'] });
    },
  });

  const toUpdateRequest = (city: City): UpdateCityRequest => ({
    name: city.name,
    name_ru: city.name_ru ?? undefined,
    country: city.country,
    center_lat: city.center_lat,
    center_lng: city.center_lng,
    radius_km: city.radius_km,
    is_active: city.is_active,
    download_size_mb: city.download_size_mb,
  });

  return {
    createCity: createCityMutation,
    updateCity: updateCityMutation,
    toUpdateRequest,
  };
}
