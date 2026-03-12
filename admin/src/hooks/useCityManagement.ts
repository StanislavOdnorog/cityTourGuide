import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createCity, updateCity, deleteCity, restoreCity } from '../api';
import { handleMutationError } from '../api/errors';
import type { ApiRequestError } from '../api/endpoints';
import type { City } from '../types';
import type { operations } from '../types';
import { cityQueryKeys } from './cityQueryKeys';

type CreateCityRequest =
  operations['adminCreateCity']['requestBody']['content']['application/json'];
type UpdateCityRequest =
  operations['adminUpdateCity']['requestBody']['content']['application/json'];
type CreateCityResponse = operations['adminCreateCity']['responses']['201']['content']['application/json'];
type UpdateCityResponse = operations['adminUpdateCity']['responses']['200']['content']['application/json'];

export function useCityManagement() {
  const queryClient = useQueryClient();

  const createCityMutation = useMutation<CreateCityResponse, ApiRequestError, CreateCityRequest>({
    mutationFn: (body: CreateCityRequest) => createCity(body) as Promise<CreateCityResponse>,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cityQueryKeys.all });
    },
    onError: handleMutationError,
  });

  const updateCityMutation = useMutation<
    UpdateCityResponse,
    ApiRequestError,
    { id: number; body: UpdateCityRequest }
  >({
    mutationFn: ({ id, body }: { id: number; body: UpdateCityRequest }) =>
      updateCity(id, body) as Promise<UpdateCityResponse>,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cityQueryKeys.all });
    },
    onError: handleMutationError,
  });

  const deleteCityMutation = useMutation<unknown, ApiRequestError, number>({
    mutationFn: (id: number) => deleteCity(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cityQueryKeys.all });
    },
    onError: handleMutationError,
  });

  const restoreCityMutation = useMutation<unknown, ApiRequestError, number>({
    mutationFn: (id: number) => restoreCity(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cityQueryKeys.all });
    },
    onError: handleMutationError,
  });

  const toUpdateRequest = (city: City): UpdateCityRequest => ({
    name: city.name,
    name_ru: city.name_ru,
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
    deleteCity: deleteCityMutation,
    restoreCity: restoreCityMutation,
    toUpdateRequest,
  };
}
