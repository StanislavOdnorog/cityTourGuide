import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { createCity, deleteCity, restoreCity, updateCity } from '../api';
import type { City } from '../types';
import { createQueryClientWrapper, createTestQueryClient } from '../test/queryClient';
import { cityQueryKeys } from './cityQueryKeys';
import { useCityManagement } from './useCityManagement';

vi.mock('../api', () => ({
  createCity: vi.fn(),
  deleteCity: vi.fn(),
  restoreCity: vi.fn(),
  updateCity: vi.fn(),
}));

const city: City = {
  id: 1,
  name: 'Tbilisi',
  name_ru: null,
  country: 'Georgia',
  center_lat: 41.7151,
  center_lng: 44.8271,
  radius_km: 15,
  is_active: true,
  download_size_mb: 120.5,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  deleted_at: null,
};

describe('useCityManagement', () => {
  it('invalidates cities queries after a successful create mutation', async () => {
    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    vi.mocked(createCity).mockResolvedValue({ data: city });

    const { result } = renderHook(() => useCityManagement(), { wrapper });

    await act(async () => {
      await result.current.createCity.mutateAsync({
        name: city.name,
        name_ru: city.name_ru,
        country: city.country,
        center_lat: city.center_lat,
        center_lng: city.center_lng,
        radius_km: city.radius_km,
        is_active: city.is_active,
        download_size_mb: city.download_size_mb,
      });
    });

    await waitFor(() => {
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: cityQueryKeys.all });
    });
  });

  it('invalidates cities queries after a successful update mutation', async () => {
    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    vi.mocked(updateCity).mockResolvedValue({ data: { ...city, is_active: false } });

    const { result } = renderHook(() => useCityManagement(), { wrapper });
    const body = result.current.toUpdateRequest(city);

    await act(async () => {
      await result.current.updateCity.mutateAsync({ id: city.id, body });
    });

    await waitFor(() => {
      expect(updateCity).toHaveBeenCalledWith(city.id, body);
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: cityQueryKeys.all });
    });
  });

  it('invalidates cities queries after a successful delete mutation', async () => {
    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    vi.mocked(deleteCity).mockResolvedValue({ message: 'city deleted' });

    const { result } = renderHook(() => useCityManagement(), { wrapper });

    await act(async () => {
      await result.current.deleteCity.mutateAsync(city.id);
    });

    await waitFor(() => {
      expect(deleteCity).toHaveBeenCalledWith(city.id);
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: cityQueryKeys.all });
    });
  });

  it('invalidates cities queries after a successful restore mutation', async () => {
    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');
    vi.mocked(restoreCity).mockResolvedValue({ data: city });

    const { result } = renderHook(() => useCityManagement(), { wrapper });

    await act(async () => {
      await result.current.restoreCity.mutateAsync(city.id);
    });

    await waitFor(() => {
      expect(restoreCity).toHaveBeenCalledWith(city.id);
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: cityQueryKeys.all });
    });
  });

  it('preserves nullable optional fields when shaping update payloads', () => {
    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);

    const { result } = renderHook(() => useCityManagement(), { wrapper });

    expect(result.current.toUpdateRequest(city)).toMatchObject({
      name_ru: null,
    });
    expect(
      result.current.toUpdateRequest({
        ...city,
        name_ru: 'Тбилиси',
      }),
    ).toMatchObject({
      name_ru: 'Тбилиси',
    });
  });
});
