import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { listCities, createCity, updateCity } from '../api';
import type { City } from '../types';
import { createQueryClientWrapper, createTestQueryClient } from '../test/queryClient';
import { useCities } from './useCities';
import { useCityManagement } from './useCityManagement';

vi.mock('../api', () => ({
  listCities: vi.fn(),
  createCity: vi.fn(),
  updateCity: vi.fn(),
}));

const city1: City = {
  id: 1,
  name: 'Tbilisi',
  name_ru: null,
  country: 'Georgia',
  center_lat: 41.7151,
  center_lng: 44.8271,
  radius_km: 15,
  is_active: true,
  download_size_mb: 120,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  deleted_at: null,
};

const city2: City = {
  id: 2,
  name: 'Draft City',
  name_ru: null,
  country: 'Georgia',
  center_lat: 42.0,
  center_lng: 43.0,
  radius_km: 10,
  is_active: false,
  download_size_mb: 50,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  deleted_at: null,
};

const city3: City = {
  id: 3,
  name: 'Batumi',
  name_ru: null,
  country: 'Georgia',
  center_lat: 41.6,
  center_lng: 41.6,
  radius_km: 8,
  is_active: true,
  download_size_mb: 80,
  created_at: '2026-01-02T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  deleted_at: null,
};

describe('useCities – cursor pagination', () => {
  it('fetches a single cursor page using limit and optional cursor', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [city1, city2],
      next_cursor: 'cursor-abc',
      has_more: true,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useCities({ limit: 2 }), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(listCities).toHaveBeenCalledWith({ limit: 2 });
    expect(result.current.data?.items).toEqual([city1, city2]);
    expect(result.current.data?.has_more).toBe(true);
    expect(result.current.data?.next_cursor).toBe('cursor-abc');
  });

  it('passes cursor to fetch the next page', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [city3],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(
      () => useCities({ cursor: 'cursor-abc', limit: 2 }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(listCities).toHaveBeenCalledWith({ limit: 2, cursor: 'cursor-abc' });
    expect(result.current.data?.items).toEqual([city3]);
    expect(result.current.data?.has_more).toBe(false);
  });

  it('includes inactive cities in results (admin endpoint)', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [city1, city2],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useCities({ limit: 20 }), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const inactive = result.current.data!.items.filter((c) => !c.is_active);
    expect(inactive).toHaveLength(1);
    expect(inactive[0].name).toBe('Draft City');
  });

  it('defaults limit to 20 when not specified', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    renderHook(() => useCities(), { wrapper });

    await waitFor(() => {
      expect(listCities).toHaveBeenCalledWith({ limit: 20 });
    });
  });

  it('passes include_deleted for admin pages that need soft-deleted rows', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [{ ...city2, deleted_at: '2026-01-03T00:00:00Z' }],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    renderHook(() => useCities({ limit: 20, includeDeleted: true }), { wrapper });

    await waitFor(() => {
      expect(listCities).toHaveBeenCalledWith({ limit: 20, include_deleted: true });
    });
  });
});

describe('useCityManagement – mutation-driven refresh', () => {
  it('invalidates city queries after create mutation', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [city1],
      next_cursor: '',
      has_more: false,
    });
    vi.mocked(createCity).mockResolvedValue({ data: city3 });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);

    const { result: citiesResult } = renderHook(() => useCities({ limit: 20 }), { wrapper });
    const { result: mgmtResult } = renderHook(() => useCityManagement(), { wrapper });

    await waitFor(() => expect(citiesResult.current.isSuccess).toBe(true));
    expect(listCities).toHaveBeenCalledTimes(1);

    await act(async () => {
      await mgmtResult.current.createCity.mutateAsync({
        name: 'Batumi',
        country: 'Georgia',
        center_lat: 41.6,
        center_lng: 41.6,
        radius_km: 8,
        is_active: true,
        download_size_mb: 80,
      });
    });

    await waitFor(() => {
      expect(listCities).toHaveBeenCalledTimes(2);
    });
  });

  it('invalidates city queries after update mutation', async () => {
    vi.mocked(listCities).mockResolvedValue({
      items: [city1],
      next_cursor: '',
      has_more: false,
    });
    vi.mocked(updateCity).mockResolvedValue({ data: { ...city1, name: 'Tbilisi Updated' } });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);

    const { result: citiesResult } = renderHook(() => useCities({ limit: 20 }), { wrapper });
    const { result: mgmtResult } = renderHook(() => useCityManagement(), { wrapper });

    await waitFor(() => expect(citiesResult.current.isSuccess).toBe(true));
    expect(listCities).toHaveBeenCalledTimes(1);

    await act(async () => {
      await mgmtResult.current.updateCity.mutateAsync({
        id: city1.id,
        body: {
          name: 'Tbilisi Updated',
          country: 'Georgia',
          center_lat: 41.7151,
          center_lng: 44.8271,
          radius_km: 15,
          is_active: true,
          download_size_mb: 120,
        },
      });
    });

    await waitFor(() => {
      expect(listCities).toHaveBeenCalledTimes(2);
    });
  });
});
