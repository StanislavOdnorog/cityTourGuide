import { create } from 'zustand';

export interface WalkLocation {
  lat: number;
  lng: number;
  heading: number;
  speed: number;
}

interface WalkState {
  isWalking: boolean;
  currentLocation: WalkLocation | null;
}

interface WalkActions {
  startWalking: () => void;
  stopWalking: () => void;
  updateLocation: (location: WalkLocation) => void;
}

export const useWalkStore = create<WalkState & WalkActions>((set) => ({
  isWalking: false,
  currentLocation: null,

  startWalking: () => set({ isWalking: true }),
  stopWalking: () => set({ isWalking: false, currentLocation: null }),
  updateLocation: (location) => set({ currentLocation: location }),
}));
