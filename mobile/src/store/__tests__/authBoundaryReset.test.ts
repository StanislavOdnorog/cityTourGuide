jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

const mockUnregister = jest.fn().mockResolvedValue(undefined);
jest.mock('@/services/notifications', () => ({
  notificationManager: {
    unregister: mockUnregister,
  },
}));

const mockClearAll = jest.fn().mockResolvedValue(undefined);
jest.mock('@/services/sync', () => ({
  getSyncQueue: jest.fn(() => ({ clearAll: mockClearAll })),
}));

import type { PurchaseStatus, User } from '@/types';
import { subscribeAuthBoundaryReset } from '../authBoundaryReset';
import { resetAuthStore, useAuthStore } from '../useAuthStore';
import { useDownloadStore } from '../useDownloadStore';
import { usePlayerStore } from '../usePlayerStore';
import { usePurchaseStore } from '../usePurchaseStore';
import { useSettingsStore } from '../useSettingsStore';

function makeUser(): User {
  return {
    id: 'user-1',
    auth_provider: 'email',
    language_pref: 'en',
    is_anonymous: false,
    is_admin: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

describe('subscribeAuthBoundaryReset', () => {
  let unsubscribe: () => void;

  beforeEach(() => {
    resetAuthStore();
    useDownloadStore.setState({ downloadsByCityId: {}, downloadedCities: {} });
    usePlayerStore.getState().reset();
    usePurchaseStore.setState({ status: null });
    useSettingsStore.setState({ pushToken: null, registeredPushUserId: null });
    mockUnregister.mockClear();
    mockClearAll.mockClear();
  });

  afterEach(() => {
    unsubscribe?.();
  });

  it('clears download and purchase stores when session is cleared', () => {
    // Set up an active session
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-abc',
      refreshToken: 'refresh-abc',
    });

    // Add user-scoped data
    useDownloadStore.getState().markCityDownloaded({
      cityId: 1,
      language: 'en',
      downloadedAt: Date.now(),
      storyIds: [10],
      totalFiles: 1,
      totalSizeBytes: 1000,
    });
    usePurchaseStore.setState({
      status: { has_full_access: true } as PurchaseStatus,
    });

    // Subscribe AFTER setting session so previousAccessToken is non-null
    unsubscribe = subscribeAuthBoundaryReset();

    // Clear the session (simulates logout)
    useAuthStore.getState().clearSession();

    // Verify user-scoped stores were reset
    expect(useDownloadStore.getState().downloadedCities).toEqual({});
    expect(useDownloadStore.getState().downloadsByCityId).toEqual({});
    expect(usePurchaseStore.getState().status).toBeNull();
  });

  it('does not clear stores when session is set (login)', () => {
    unsubscribe = subscribeAuthBoundaryReset();

    useDownloadStore.getState().markCityDownloaded({
      cityId: 2,
      language: 'en',
      downloadedAt: Date.now(),
      storyIds: [20],
      totalFiles: 1,
      totalSizeBytes: 500,
    });

    // Setting a session should NOT clear stores
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-xyz',
      refreshToken: 'refresh-xyz',
    });

    expect(useDownloadStore.getState().downloadedCities).toHaveProperty('2');
  });

  it('resets player store on session clear', () => {
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-abc',
      refreshToken: 'refresh-abc',
    });

    // Simulate active playback state
    usePlayerStore.setState({
      isPlaying: true,
      listenedStoryIds: new Set([1, 2]),
      listenedPoiIds: new Set([10, 20]),
    });

    unsubscribe = subscribeAuthBoundaryReset();
    useAuthStore.getState().clearSession();

    expect(usePlayerStore.getState().isPlaying).toBe(false);
    expect(usePlayerStore.getState().currentStory).toBeNull();
    expect(usePlayerStore.getState().listenedStoryIds.size).toBe(0);
    expect(usePlayerStore.getState().listenedPoiIds.size).toBe(0);
  });

  it('clears push registration and unregisters notifications on session clear', () => {
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-abc',
      refreshToken: 'refresh-abc',
    });
    useSettingsStore.setState({ pushToken: 'expo-token', registeredPushUserId: 'user-1' });

    unsubscribe = subscribeAuthBoundaryReset();
    useAuthStore.getState().clearSession();

    expect(useSettingsStore.getState().pushToken).toBeNull();
    expect(useSettingsStore.getState().registeredPushUserId).toBeNull();
    expect(mockUnregister).toHaveBeenCalledTimes(1);
  });

  it('clears sync queue on session clear to prevent cross-user replay', () => {
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-abc',
      refreshToken: 'refresh-abc',
    });

    unsubscribe = subscribeAuthBoundaryReset();
    useAuthStore.getState().clearSession();

    expect(mockClearAll).toHaveBeenCalledTimes(1);
  });

  it('does not clear sync queue when session is set (login)', () => {
    unsubscribe = subscribeAuthBoundaryReset();

    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-xyz',
      refreshToken: 'refresh-xyz',
    });

    expect(mockClearAll).not.toHaveBeenCalled();
  });

  it('clears user-scoped stores when userId changes without accessToken going null', () => {
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-abc',
      refreshToken: 'refresh-abc',
    });
    useDownloadStore.getState().markCityDownloaded({
      cityId: 3,
      language: 'en',
      downloadedAt: Date.now(),
      storyIds: [30],
      totalFiles: 1,
      totalSizeBytes: 750,
    });
    usePurchaseStore.setState({
      status: { has_full_access: true } as PurchaseStatus,
    });
    useSettingsStore.setState({ pushToken: 'expo-token', registeredPushUserId: 'user-1' });

    unsubscribe = subscribeAuthBoundaryReset();

    useAuthStore.getState().setSession({
      user: { ...makeUser(), id: 'user-2' },
      userId: 'user-2',
      accessToken: 'token-abc',
      refreshToken: 'refresh-next',
    });

    expect(useDownloadStore.getState().downloadedCities).toEqual({});
    expect(useDownloadStore.getState().downloadsByCityId).toEqual({});
    expect(usePurchaseStore.getState().status).toBeNull();
    expect(useSettingsStore.getState().pushToken).toBeNull();
    expect(useSettingsStore.getState().registeredPushUserId).toBeNull();
    expect(mockUnregister).toHaveBeenCalledTimes(1);
    expect(mockClearAll).toHaveBeenCalledTimes(1);
  });

  it('does not clear user-scoped stores when only the token rotates for the same user', () => {
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-abc',
      refreshToken: 'refresh-abc',
    });
    useDownloadStore.getState().markCityDownloaded({
      cityId: 4,
      language: 'en',
      downloadedAt: Date.now(),
      storyIds: [40],
      totalFiles: 1,
      totalSizeBytes: 900,
    });
    usePurchaseStore.setState({
      status: { has_full_access: true } as PurchaseStatus,
    });

    unsubscribe = subscribeAuthBoundaryReset();

    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-rotated',
      refreshToken: 'refresh-rotated',
    });

    expect(useDownloadStore.getState().downloadedCities).toHaveProperty('4');
    expect(usePurchaseStore.getState().status).not.toBeNull();
    expect(mockUnregister).not.toHaveBeenCalled();
    expect(mockClearAll).not.toHaveBeenCalled();
  });

  it('returns an unsubscribe function that stops the listener', () => {
    useAuthStore.getState().setSession({
      user: makeUser(),
      userId: 'user-1',
      accessToken: 'token-1',
      refreshToken: 'refresh-1',
    });

    usePurchaseStore.setState({
      status: { has_full_access: false } as PurchaseStatus,
    });

    unsubscribe = subscribeAuthBoundaryReset();
    unsubscribe();

    // After unsubscribing, clearing session should NOT reset stores
    useAuthStore.getState().clearSession();

    expect(usePurchaseStore.getState().status).not.toBeNull();
  });
});
