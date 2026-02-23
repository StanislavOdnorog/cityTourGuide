import AsyncStorage from '@react-native-async-storage/async-storage';
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type AppLanguage = 'en' | 'ru';

function generateDeviceId(): string {
  const s4 = () =>
    Math.floor((1 + Math.random()) * 0x10000)
      .toString(16)
      .substring(1);
  return `${s4()}${s4()}-${s4()}-${s4()}-${s4()}-${s4()}${s4()}${s4()}`;
}

interface SettingsState {
  language: AppLanguage;
  onboardingCompleted: boolean;
  deviceId: string;
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
      deviceId: generateDeviceId(),
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
        deviceId: state.deviceId,
      }),
      onRehydrateStorage: () => () => {
        useSettingsStore.setState({ _hasHydrated: true });
      },
    },
  ),
);
