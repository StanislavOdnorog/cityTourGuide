import { useSettingsStore } from '../useSettingsStore';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

describe('useSettingsStore', () => {
  beforeEach(() => {
    // Reset store between tests
    useSettingsStore.setState({
      language: 'en',
      onboardingCompleted: false,
      geoNotifications: true,
      contentNotifications: true,
      _hasHydrated: false,
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
