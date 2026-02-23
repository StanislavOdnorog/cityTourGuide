import { useCallback, useEffect } from 'react';
import { fetchCityById, fetchCityPOIs } from '@/api';
import { useCityStore } from '@/store/useCityStore';
import { usePlayerStore } from '@/store/usePlayerStore';
import { useSettingsStore } from '@/store/useSettingsStore';
import type { CityPOI } from '@/types';

export type MarkerColor = 'green' | 'blue' | 'grey';

export interface CityMarker {
  poi: CityPOI;
  color: MarkerColor;
}

export interface CityScreenState {
  cityName: string | null;
  markers: CityMarker[];
  totalStories: number;
  listenedCount: number;
  centerLat: number;
  centerLng: number;
  isLoading: boolean;
  error: string | null;
}

export interface CityScreenActions {
  loadCity: (cityId: number) => Promise<void>;
  refresh: () => Promise<void>;
}

export function getMarkerColor(
  poi: CityPOI,
  listenedPoiIds: Set<number>,
  isPurchased: boolean,
): MarkerColor {
  if (!isPurchased && poi.story_count > 0) {
    return 'grey';
  }
  if (listenedPoiIds.has(poi.id)) {
    return 'green';
  }
  return 'blue';
}

export function useCityScreen(cityId: number): CityScreenState & CityScreenActions {
  const selectedCity = useCityStore((s) => s.selectedCity);
  const pois = useCityStore((s) => s.pois);
  const totalStories = useCityStore((s) => s.totalStories);
  const isLoading = useCityStore((s) => s.isLoading);
  const error = useCityStore((s) => s.error);
  const { setSelectedCity, setPois, setLoading, setError } = useCityStore();

  const listenedStoryIds = usePlayerStore((s) => s.listenedStoryIds);
  const listenedPoiIds = usePlayerStore((s) => s.listenedPoiIds);
  const language = useSettingsStore((s) => s.language);

  // Freemium check is handled by TASK-047; for now all POIs are accessible
  const isPurchased = true;

  const markers: CityMarker[] = pois.map((poi) => ({
    poi,
    color: getMarkerColor(poi, listenedPoiIds, isPurchased),
  }));

  const loadCity = useCallback(
    async (id: number) => {
      setLoading(true);
      setError(null);
      try {
        const [city, { pois: cityPois, totalStories: total }] = await Promise.all([
          fetchCityById(id),
          fetchCityPOIs(id, language),
        ]);
        setSelectedCity(city);
        setPois(cityPois, total);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to load city data';
        setError(message);
      } finally {
        setLoading(false);
      }
    },
    [language, setSelectedCity, setPois, setLoading, setError],
  );

  const refresh = useCallback(async () => {
    await loadCity(cityId);
  }, [cityId, loadCity]);

  useEffect(() => {
    void loadCity(cityId);
  }, [cityId, loadCity]);

  return {
    cityName: selectedCity?.name ?? null,
    markers,
    totalStories,
    listenedCount: listenedStoryIds.size,
    centerLat: selectedCity?.center_lat ?? 0,
    centerLng: selectedCity?.center_lng ?? 0,
    isLoading,
    error,
    loadCity,
    refresh,
  };
}
