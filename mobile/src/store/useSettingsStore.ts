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
  geoNotifications: boolean;
  contentNotifications: boolean;
  pushToken: string | null;
  registeredPushUserId: string | null;
  _hasHydrated: boolean;
  _rehydrationError: string | null;
}

type PersistedSettingsState = Omit<SettingsState, '_hasHydrated' | '_rehydrationError'>;

interface SettingsActions {
  setLanguage: (language: AppLanguage) => void;
  completeOnboarding: () => void;
  setGeoNotifications: (enabled: boolean) => void;
  setContentNotifications: (enabled: boolean) => void;
  setPushToken: (token: string | null) => void;
  setPushRegistration: (token: string | null, userId: string | null) => void;
  clearPushRegistration: () => void;
  setHasHydrated: (value: boolean) => void;
}

function createDefaultState(): SettingsState {
  return {
    language: 'en',
    onboardingCompleted: false,
    deviceId: generateDeviceId(),
    geoNotifications: true,
    contentNotifications: true,
    pushToken: null,
    registeredPushUserId: null,
    _hasHydrated: false,
    _rehydrationError: null,
  };
}

function sanitizePersistedSettingsState(
  value: unknown,
  currentState: PersistedSettingsState,
): PersistedSettingsState {
  if (!value || typeof value !== 'object') {
    return currentState;
  }

  const persisted = value as Partial<PersistedSettingsState>;

  return {
    language:
      persisted.language === 'en' || persisted.language === 'ru'
        ? persisted.language
        : currentState.language,
    onboardingCompleted:
      typeof persisted.onboardingCompleted === 'boolean'
        ? persisted.onboardingCompleted
        : currentState.onboardingCompleted,
    deviceId:
      typeof persisted.deviceId === 'string' && persisted.deviceId.length > 0
        ? persisted.deviceId
        : currentState.deviceId,
    geoNotifications:
      typeof persisted.geoNotifications === 'boolean'
        ? persisted.geoNotifications
        : currentState.geoNotifications,
    contentNotifications:
      typeof persisted.contentNotifications === 'boolean'
        ? persisted.contentNotifications
        : currentState.contentNotifications,
    pushToken: persisted.pushToken ?? currentState.pushToken,
    registeredPushUserId: persisted.registeredPushUserId ?? currentState.registeredPushUserId,
  };
}

export const useSettingsStore = create<SettingsState & SettingsActions>()(
  persist(
    (set) => ({
      ...createDefaultState(),

      setLanguage: (language) => set({ language }),
      completeOnboarding: () => set({ onboardingCompleted: true }),
      setGeoNotifications: (enabled) => set({ geoNotifications: enabled }),
      setContentNotifications: (enabled) => set({ contentNotifications: enabled }),
      setPushToken: (token) => set({ pushToken: token }),
      setPushRegistration: (token, userId) =>
        set({ pushToken: token, registeredPushUserId: userId }),
      clearPushRegistration: () => set({ pushToken: null, registeredPushUserId: null }),
      setHasHydrated: (value) => set({ _hasHydrated: value }),
    }),
    {
      name: 'city-stories-settings',
      storage: createJSONStorage(() => AsyncStorage),
      version: 1,
      migrate: (persisted, version) => {
        const state = (persisted ?? {}) as Partial<PersistedSettingsState>;
        if (version === 0) {
          return {
            language: state.language ?? 'en',
            onboardingCompleted: state.onboardingCompleted ?? false,
            deviceId: state.deviceId ?? generateDeviceId(),
            geoNotifications: state.geoNotifications ?? true,
            contentNotifications: state.contentNotifications ?? true,
            pushToken: state.pushToken ?? null,
            registeredPushUserId: state.registeredPushUserId ?? null,
          };
        }
        return state as PersistedSettingsState;
      },
      partialize: (state) => ({
        language: state.language,
        onboardingCompleted: state.onboardingCompleted,
        deviceId: state.deviceId,
        geoNotifications: state.geoNotifications,
        contentNotifications: state.contentNotifications,
        pushToken: state.pushToken,
        registeredPushUserId: state.registeredPushUserId,
      }),
      merge: (persistedState, currentState) => ({
        ...currentState,
        ...sanitizePersistedSettingsState(persistedState, {
          language: currentState.language,
          onboardingCompleted: currentState.onboardingCompleted,
          deviceId: currentState.deviceId,
          geoNotifications: currentState.geoNotifications,
          contentNotifications: currentState.contentNotifications,
          pushToken: currentState.pushToken,
          registeredPushUserId: currentState.registeredPushUserId,
        }),
      }),
      onRehydrateStorage: () => (_state, error) => {
        if (error) {
          console.warn('useSettingsStore: rehydration failed', error);
        }
        useSettingsStore.setState({
          _hasHydrated: true,
          _rehydrationError: error ? 'Failed to restore settings state.' : null,
        });
      },
    },
  ),
);
