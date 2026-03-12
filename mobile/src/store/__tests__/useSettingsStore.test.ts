import AsyncStorage from '@react-native-async-storage/async-storage';
import { useSettingsStore } from '../useSettingsStore';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

const mockGetItem = AsyncStorage.getItem as jest.Mock;

describe('useSettingsStore', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    const deviceId = useSettingsStore.getState().deviceId;
    // Reset store between tests
    useSettingsStore.setState({
      language: 'en',
      onboardingCompleted: false,
      deviceId,
      geoNotifications: true,
      contentNotifications: true,
      pushToken: null,
      registeredPushUserId: null,
      _hasHydrated: false,
      _rehydrationError: null,
    });
  });

  it('has correct initial state', () => {
    const state = useSettingsStore.getState();
    expect(state.language).toBe('en');
    expect(state.onboardingCompleted).toBe(false);
    expect(state.geoNotifications).toBe(true);
    expect(state.contentNotifications).toBe(true);
  });

  it('setLanguage changes language to ru', () => {
    useSettingsStore.getState().setLanguage('ru');
    expect(useSettingsStore.getState().language).toBe('ru');
  });

  it('setLanguage changes language back to en', () => {
    useSettingsStore.getState().setLanguage('ru');
    useSettingsStore.getState().setLanguage('en');
    expect(useSettingsStore.getState().language).toBe('en');
  });

  it('completeOnboarding sets onboardingCompleted to true', () => {
    useSettingsStore.getState().completeOnboarding();
    expect(useSettingsStore.getState().onboardingCompleted).toBe(true);
  });

  it('completeOnboarding persists after language change', () => {
    useSettingsStore.getState().completeOnboarding();
    useSettingsStore.getState().setLanguage('ru');

    const state = useSettingsStore.getState();
    expect(state.onboardingCompleted).toBe(true);
    expect(state.language).toBe('ru');
  });

  it('setHasHydrated updates hydration state', () => {
    useSettingsStore.getState().setHasHydrated(true);
    expect(useSettingsStore.getState()._hasHydrated).toBe(true);
  });

  it('generates a deviceId on creation', () => {
    const state = useSettingsStore.getState();
    expect(state.deviceId).toBeDefined();
    expect(typeof state.deviceId).toBe('string');
    expect(state.deviceId.length).toBeGreaterThan(0);
  });

  it('deviceId persists across state changes', () => {
    const originalId = useSettingsStore.getState().deviceId;
    useSettingsStore.getState().setLanguage('ru');
    expect(useSettingsStore.getState().deviceId).toBe(originalId);
  });

  it('setGeoNotifications disables geo notifications', () => {
    useSettingsStore.getState().setGeoNotifications(false);
    expect(useSettingsStore.getState().geoNotifications).toBe(false);
  });

  it('setGeoNotifications re-enables geo notifications', () => {
    useSettingsStore.getState().setGeoNotifications(false);
    useSettingsStore.getState().setGeoNotifications(true);
    expect(useSettingsStore.getState().geoNotifications).toBe(true);
  });

  it('setContentNotifications disables content notifications', () => {
    useSettingsStore.getState().setContentNotifications(false);
    expect(useSettingsStore.getState().contentNotifications).toBe(false);
  });

  it('setContentNotifications re-enables content notifications', () => {
    useSettingsStore.getState().setContentNotifications(false);
    useSettingsStore.getState().setContentNotifications(true);
    expect(useSettingsStore.getState().contentNotifications).toBe(true);
  });

  describe('pre-hydration defaults', () => {
    it('returns defaults before hydration completes', () => {
      useSettingsStore.setState({ _hasHydrated: false });
      const state = useSettingsStore.getState();
      expect(state._hasHydrated).toBe(false);
      expect(state.language).toBe('en');
      expect(state.onboardingCompleted).toBe(false);
      expect(state.geoNotifications).toBe(true);
      expect(state.contentNotifications).toBe(true);
      expect(state.pushToken).toBeNull();
      expect(state.registeredPushUserId).toBeNull();
    });
  });

  describe('corrupted AsyncStorage data', () => {
    it('handles non-JSON garbage gracefully and sets rehydration error', async () => {
      mockGetItem.mockResolvedValueOnce('not valid json {{{');
      await useSettingsStore.persist.rehydrate();

      const state = useSettingsStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state._rehydrationError).toBe('Failed to restore settings state.');
      expect(state.language).toBe('en');
      expect(state.geoNotifications).toBe(true);
    });

    it('handles null persisted state gracefully without error', async () => {
      mockGetItem.mockResolvedValueOnce(null);
      await useSettingsStore.persist.rehydrate();

      const state = useSettingsStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state._rehydrationError).toBeNull();
      expect(state.language).toBe('en');
    });

    it('handles state with wrong types gracefully', async () => {
      const defaultDeviceId = useSettingsStore.getState().deviceId;

      mockGetItem.mockResolvedValueOnce(
        JSON.stringify({
          state: {
            language: 12345,
            deviceId: '',
            onboardingCompleted: 'not-a-bool',
            geoNotifications: null,
            contentNotifications: 'false',
          },
          version: 1,
        }),
      );
      await useSettingsStore.persist.rehydrate();

      const state = useSettingsStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state.language).toBe('en');
      expect(state.deviceId).toBe(defaultDeviceId);
      expect(state.onboardingCompleted).toBe(false);
      expect(state.geoNotifications).toBe(true);
      expect(state.contentNotifications).toBe(true);
    });

    it('handles empty object persisted state gracefully', async () => {
      mockGetItem.mockResolvedValueOnce(JSON.stringify({ state: {}, version: 1 }));
      await useSettingsStore.persist.rehydrate();

      const state = useSettingsStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state.deviceId).toBeDefined();
    });

    it('preserves deviceId from valid persisted state even with other corrupt fields', async () => {
      mockGetItem.mockResolvedValueOnce(
        JSON.stringify({
          state: {
            deviceId: 'persisted-device-id',
            language: null,
          },
          version: 1,
        }),
      );
      await useSettingsStore.persist.rehydrate();

      const state = useSettingsStore.getState();
      expect(state._hasHydrated).toBe(true);
      expect(state.deviceId).toBe('persisted-device-id');
      expect(state.language).toBe('en');
    });

    it('falls back to the generated deviceId when persisted deviceId is null or not a string', async () => {
      const defaultDeviceId = useSettingsStore.getState().deviceId;

      mockGetItem.mockResolvedValueOnce(
        JSON.stringify({
          state: {
            deviceId: null,
            language: 'ru',
            onboardingCompleted: true,
          },
          version: 1,
        }),
      );

      await useSettingsStore.persist.rehydrate();

      let state = useSettingsStore.getState();
      expect(state.deviceId).toBe(defaultDeviceId);
      expect(state.language).toBe('ru');
      expect(state.onboardingCompleted).toBe(true);

      useSettingsStore.setState({
        language: 'en',
        onboardingCompleted: false,
        deviceId: defaultDeviceId,
        geoNotifications: true,
        contentNotifications: true,
        pushToken: null,
        registeredPushUserId: null,
        _hasHydrated: false,
        _rehydrationError: null,
      });

      mockGetItem.mockResolvedValueOnce(
        JSON.stringify({
          state: {
            deviceId: 12345,
          },
          version: 1,
        }),
      );

      await useSettingsStore.persist.rehydrate();

      state = useSettingsStore.getState();
      expect(state.deviceId).toBe(defaultDeviceId);
    });

    it('falls back to defaults for invalid language and boolean hydration values', async () => {
      mockGetItem.mockResolvedValueOnce(
        JSON.stringify({
          state: {
            language: 'de',
            onboardingCompleted: 1,
            geoNotifications: 'yes',
            contentNotifications: 0,
          },
          version: 1,
        }),
      );

      await useSettingsStore.persist.rehydrate();

      const state = useSettingsStore.getState();
      expect(state.language).toBe('en');
      expect(state.onboardingCompleted).toBe(false);
      expect(state.geoNotifications).toBe(true);
      expect(state.contentNotifications).toBe(true);
    });
  });

  it('notification settings persist across language changes', () => {
    useSettingsStore.getState().setGeoNotifications(false);
    useSettingsStore.getState().setContentNotifications(false);
    useSettingsStore.getState().setLanguage('ru');

    const state = useSettingsStore.getState();
    expect(state.geoNotifications).toBe(false);
    expect(state.contentNotifications).toBe(false);
    expect(state.language).toBe('ru');
  });
});
