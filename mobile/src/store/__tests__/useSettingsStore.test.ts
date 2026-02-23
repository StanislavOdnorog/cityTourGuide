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
      _hasHydrated: false,
    });
  });

  it('has correct initial state', () => {
    const state = useSettingsStore.getState();
    expect(state.language).toBe('en');
    expect(state.onboardingCompleted).toBe(false);
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
});
