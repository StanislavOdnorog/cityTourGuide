import { create } from 'zustand';
import type { City, CityPOI } from '@/types';

interface CityState {
  selectedCity: City | null;
  pois: CityPOI[];
  totalStories: number;
  isLoading: boolean;
  error: string | null;
}

interface CityActions {
  setSelectedCity: (city: City | null) => void;
  setPois: (pois: CityPOI[], totalStories: number) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const initialState: CityState = {
  selectedCity: null,
  pois: [],
  totalStories: 0,
  isLoading: false,
  error: null,
};

export const useCityStore = create<CityState & CityActions>((set) => ({
  ...initialState,

  setSelectedCity: (city) => set({ selectedCity: city }),
  setPois: (pois, totalStories) => set({ pois, totalStories }),
  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
  reset: () => set(initialState),
}));
