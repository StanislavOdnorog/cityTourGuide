import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { getAdminStats } from '../api';
import { createQueryClientWrapper, createTestQueryClient } from '../test/queryClient';
import { useCities } from './useCities';
import { useDashboardStats } from './useDashboardStats';

vi.mock('../api', () => ({
  getAdminStats: vi.fn(),
}));

vi.mock('./useCities', () => ({
  useCities: vi.fn(),
}));

const mockCities = () => {
  vi.mocked(useCities).mockReturnValue({
    data: {
      items: [
        {
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
        },
      ],
      next_cursor: '',
      has_more: false,
    },
    isLoading: false,
  } as ReturnType<typeof useCities>);
};

describe('useDashboardStats', () => {
  it('combines the admin summary endpoint with the separate cities query', async () => {
    mockCities();
    vi.mocked(getAdminStats).mockResolvedValue({
      data: {
        cities_count: 1,
        pois_count: 5,
        stories_count: 9,
        reports_count: 2,
        new_reports_count: 1,
      },
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useDashboardStats(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
      expect(result.current.stats).toEqual({
        citiesCount: 1,
        poisCount: 5,
        storiesCount: 9,
        reportsCount: 2,
        newReportsCount: 1,
      });
      expect(result.current.cities).toHaveLength(1);
      expect(result.current.isError).toBe(false);
    });

    expect(getAdminStats).toHaveBeenCalledTimes(1);
  });

  it('returns stats as null and isError true when the API call fails', async () => {
    mockCities();
    vi.mocked(getAdminStats).mockRejectedValue(new Error('Network error'));

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useDashboardStats(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
      expect(result.current.isError).toBe(true);
      expect(result.current.error).toBeInstanceOf(Error);
      expect(result.current.stats).toBeNull();
    });
  });

  it('returns stats as null while loading', () => {
    vi.mocked(useCities).mockReturnValue({
      data: undefined,
      isLoading: true,
    } as ReturnType<typeof useCities>);
    vi.mocked(getAdminStats).mockReturnValue(new Promise(() => {}));

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useDashboardStats(), { wrapper });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.stats).toBeNull();
    expect(result.current.cities).toEqual([]);
  });
});
