import AsyncStorage from '@react-native-async-storage/async-storage';
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type AppLanguage = 'en' | 'ru';

interface SettingsState {
  language: AppLanguage;
  onboardingCompleted: boolean;
  _hasHydrated: boolean;
}

interface SettingsActions {
  setLanguage: (language: AppLanguage) => void;
  completeOnboarding: () => void;
  setHasHydrated: (value: boolean) => void;
}

export const useSettingsStore = create<SettingsState & SettingsActions>()(
  persist(
    (set) => ({
      language: 'en',
      onboardingCompleted: false,
      _hasHydrated: false,

      setLanguage: (language) => set({ language }),
      completeOnboarding: () => set({ onboardingCompleted: true }),
      setHasHydrated: (value) => set({ _hasHydrated: value }),
    }),
    {
      name: 'city-stories-settings',
      storage: createJSONStorage(() => AsyncStorage),
      partialize: (state) => ({
        language: state.language,
        onboardingCompleted: state.onboardingCompleted,
      }),
      onRehydrateStorage: () => () => {
        useSettingsStore.setState({ _hasHydrated: true });
      },
    },
  ),
);
