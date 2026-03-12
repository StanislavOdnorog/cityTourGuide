import AsyncStorage from '@react-native-async-storage/async-storage';
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';
import type { PurchaseStatus, PurchaseType } from '@/types';

interface PurchaseState {
  status: PurchaseStatus | null;
  isLoading: boolean;
  paywallVisible: boolean;
  _hasHydrated: boolean;
  _rehydrationError: string | null;
}

interface PurchaseActions {
  setStatus: (status: PurchaseStatus) => void;
  setLoading: (loading: boolean) => void;
  showPaywall: () => void;
  hidePaywall: () => void;
  hasFullAccess: () => boolean;
  hasCityAccess: (cityId: number) => boolean;
  canListenFree: () => boolean;
  decrementFreeStories: () => void;
  setHasHydrated: (value: boolean) => void;
}

export const usePurchaseStore = create<PurchaseState & PurchaseActions>()(
  persist(
    (set, get) => ({
      status: null,
      isLoading: false,
      paywallVisible: false,
      _hasHydrated: false,
      _rehydrationError: null,

      setStatus: (status) => set({ status }),
      setLoading: (loading) => set({ isLoading: loading }),
      showPaywall: () => set({ paywallVisible: true }),
      hidePaywall: () => set({ paywallVisible: false }),

      hasFullAccess: () => {
        const { status } = get();
        return status?.has_full_access ?? false;
      },

      hasCityAccess: (cityId: number) => {
        const { status } = get();
        if (!status) return true; // No status loaded yet — allow
        if (status.has_full_access) return true;
        return (status.city_packs ?? []).some((p) => p.city_id === cityId);
      },

      canListenFree: () => {
        const { status } = get();
        if (!status) return true; // No status loaded yet — allow
        if (status.has_full_access) return true;
        return status.free_stories_left > 0;
      },

      decrementFreeStories: () =>
        set((state) => {
          if (!state.status) return state;
          return {
            status: {
              ...state.status,
              free_stories_used: state.status.free_stories_used + 1,
              free_stories_left: Math.max(0, state.status.free_stories_left - 1),
            },
          };
        }),

      setHasHydrated: (value) => set({ _hasHydrated: value }),
    }),
    {
      name: 'city-stories-purchases',
      storage: createJSONStorage(() => AsyncStorage),
      version: 1,
      migrate: (persisted, version) => {
        const state = (persisted ?? {}) as Partial<Pick<PurchaseState, 'status'>>;
        if (version === 0) {
          return {
            status: state.status ?? null,
          };
        }
        return state as Pick<PurchaseState, 'status'>;
      },
      partialize: (state) => ({
        status: state.status,
      }),
      onRehydrateStorage: () => (_state, error) => {
        if (error) {
          console.warn('usePurchaseStore: rehydration failed', error);
        }
        usePurchaseStore.setState({
          _hasHydrated: true,
          _rehydrationError: error ? 'Failed to restore purchase state.' : null,
        });
      },
    },
  ),
);

// Product IDs for IAP
export const IAP_PRODUCTS = {
  CITY_PACK: 'city_stories_city_pack' as const,
  MONTHLY: 'city_stories_monthly' as const,
  LIFETIME: 'city_stories_lifetime' as const,
} as const;

// Product metadata for display
export interface ProductInfo {
  id: string;
  type: PurchaseType;
  title: string;
  description: string;
  price: string;
}

export const PRODUCT_CATALOG: ProductInfo[] = [
  {
    id: IAP_PRODUCTS.CITY_PACK,
    type: 'city_pack',
    title: 'City Pack',
    description: 'Unlock all stories in one city',
    price: '$4.99',
  },
  {
    id: IAP_PRODUCTS.MONTHLY,
    type: 'subscription',
    title: 'Monthly Pass',
    description: 'All cities, unlimited stories',
    price: '$6.99/mo',
  },
  {
    id: IAP_PRODUCTS.LIFETIME,
    type: 'lifetime',
    title: 'Lifetime Access',
    description: 'All cities, forever. Best value!',
    price: '$19.99',
  },
];
