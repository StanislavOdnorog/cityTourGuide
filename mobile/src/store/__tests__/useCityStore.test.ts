import type { City, CityPOI } from '@/types';
import { useCityStore } from '../useCityStore';

const mockCity: City = {
  id: 1,
  name: 'Tbilisi',
  name_ru: 'Тбилиси',
  country: 'Georgia',
  center_lat: 41.7151,
  center_lng: 44.8271,
  radius_km: 10,
  is_active: true,
  download_size_mb: 50,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const mockPOIs: CityPOI[] = [
  {
    id: 1,
    city_id: 1,
    name: 'Narikala Fortress',
    name_ru: 'Крепость Нарикала',
    lat: 41.6875,
    lng: 44.8078,
    type: 'monument',
    tags: null,
    address: null,
    interest_score: 90,
    status: 'active',
    story_count: 3,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 2,
    city_id: 1,
    name: 'Peace Bridge',
    name_ru: 'Мост Мира',
    lat: 41.6932,
    lng: 44.8065,
    type: 'bridge',
    tags: null,
    address: null,
    interest_score: 85,
    status: 'active',
    story_count: 2,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
];

describe('useCityStore', () => {
  beforeEach(() => {
    useCityStore.getState().reset();
  });

  it('has correct initial state', () => {
    const state = useCityStore.getState();
    expect(state.selectedCity).toBeNull();
    expect(state.pois).toEqual([]);
    expect(state.totalStories).toBe(0);
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it('setSelectedCity stores the city', () => {
    useCityStore.getState().setSelectedCity(mockCity);
    expect(useCityStore.getState().selectedCity).toEqual(mockCity);
  });

  it('setSelectedCity with null clears the city', () => {
    useCityStore.getState().setSelectedCity(mockCity);
    useCityStore.getState().setSelectedCity(null);
    expect(useCityStore.getState().selectedCity).toBeNull();
  });

  it('setPois stores POIs and total stories', () => {
    useCityStore.getState().setPois(mockPOIs, 5);
    expect(useCityStore.getState().pois).toEqual(mockPOIs);
    expect(useCityStore.getState().totalStories).toBe(5);
  });

  it('setLoading toggles loading state', () => {
    useCityStore.getState().setLoading(true);
    expect(useCityStore.getState().isLoading).toBe(true);

    useCityStore.getState().setLoading(false);
    expect(useCityStore.getState().isLoading).toBe(false);
  });

  it('setError stores error message', () => {
    useCityStore.getState().setError('Network error');
    expect(useCityStore.getState().error).toBe('Network error');
  });

  it('setError with null clears error', () => {
    useCityStore.getState().setError('Network error');
    useCityStore.getState().setError(null);
    expect(useCityStore.getState().error).toBeNull();
  });

  it('reset clears all state', () => {
    useCityStore.getState().setSelectedCity(mockCity);
    useCityStore.getState().setPois(mockPOIs, 5);
    useCityStore.getState().setLoading(true);
    useCityStore.getState().setError('error');

    useCityStore.getState().reset();

    const state = useCityStore.getState();
    expect(state.selectedCity).toBeNull();
    expect(state.pois).toEqual([]);
    expect(state.totalStories).toBe(0);
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });
});
