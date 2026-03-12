import AsyncStorage from '@react-native-async-storage/async-storage';
import axios from 'axios';
import { setRefreshHandler, setTokens } from '@/api/client';
// eslint-disable-next-line import/order
import type { User } from '@/types';

jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

jest.mock('axios', () => ({
  __esModule: true,
  default: {
    post: jest.fn(),
    isAxiosError: jest.fn(() => false),
  },
  post: jest.fn(),
  isAxiosError: jest.fn(() => false),
}));

jest.mock('@/api/client', () => ({
  setTokens: jest.fn(),
  setRefreshHandler: jest.fn(),
}));

import { bootstrapAnonymousAuth, refreshAuthSession } from '../authBootstrap';
import { resetAuthStore, useAuthStore } from '../useAuthStore';
import { useSettingsStore } from '../useSettingsStore';

const mockAxiosPost = axios.post as jest.Mock;
const mockSetTokens = setTokens as jest.Mock;
const mockSetRefreshHandler = setRefreshHandler as jest.Mock;
const mockGetItem = AsyncStorage.getItem as jest.Mock;

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: 'backend-user-123',
    auth_provider: 'email',
    language_pref: 'en',
    is_anonymous: true,
    is_admin: false,
    created_at: '2026-03-12T00:00:00.000Z',
    updated_at: '2026-03-12T00:00:00.000Z',
    ...overrides,
  };
}

describe('auth bootstrap', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetItem.mockResolvedValue(null);
    resetAuthStore();
    useSettingsStore.setState({
      language: 'en',
      onboardingCompleted: false,
      deviceId: 'device-abc',
      geoNotifications: true,
      contentNotifications: true,
      pushToken: null,
      _hasHydrated: true,
    });
    useAuthStore.setState({
      user: null,
      userId: null,
      accessToken: null,
      refreshToken: null,
      _hasHydrated: true,
      bootstrapStatus: 'idle',
      bootstrapError: null,
    });
  });

  it('authenticates a fresh install through /auth/device and stores tokens', async () => {
    mockAxiosPost.mockResolvedValueOnce({
      data: {
        data: makeUser(),
        tokens: {
          access_token: 'access-1',
          refresh_token: 'refresh-1',
          expires_in: 3600,
        },
      },
    });

    const result = await bootstrapAnonymousAuth();

    expect(result).toBe(true);
    expect(mockSetRefreshHandler).toHaveBeenCalledWith(expect.any(Function));
    expect(mockSetTokens).toHaveBeenLastCalledWith('access-1', 'refresh-1');
    expect(mockAxiosPost).toHaveBeenCalledWith(
      expect.stringContaining('/auth/device'),
      expect.objectContaining({
        device_id: 'device-abc',
        language: 'en',
      }),
      expect.any(Object),
    );
    expect(useAuthStore.getState().userId).toBe('backend-user-123');
    expect(useAuthStore.getState().bootstrapStatus).toBe('ready');
  });

  it('refreshes an existing session before falling back to device auth', async () => {
    useAuthStore.setState({
      user: makeUser(),
      userId: 'backend-user-123',
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      _hasHydrated: true,
      bootstrapStatus: 'idle',
      bootstrapError: null,
    });
    mockAxiosPost.mockResolvedValueOnce({
      data: {
        tokens: {
          access_token: 'new-access',
          refresh_token: 'new-refresh',
          expires_in: 3600,
        },
      },
    });

    const result = await bootstrapAnonymousAuth();

    expect(result).toBe(true);
    expect(mockAxiosPost).toHaveBeenCalledTimes(1);
    expect(mockAxiosPost).toHaveBeenCalledWith(
      expect.stringContaining('/auth/refresh'),
      { refresh_token: 'old-refresh' },
      expect.any(Object),
    );
    expect(mockSetTokens).toHaveBeenLastCalledWith('new-access', 'new-refresh');
    expect(useAuthStore.getState().accessToken).toBe('new-access');
    expect(useAuthStore.getState().refreshToken).toBe('new-refresh');
  });

  it('exposes a controlled failure state when refresh and device auth both fail', async () => {
    useAuthStore.setState({
      user: makeUser(),
      userId: 'backend-user-123',
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      _hasHydrated: true,
      bootstrapStatus: 'idle',
      bootstrapError: null,
    });
    mockAxiosPost.mockRejectedValueOnce(new Error('refresh expired'));
    mockAxiosPost.mockRejectedValueOnce(new Error('device auth unavailable'));

    const result = await bootstrapAnonymousAuth();

    expect(result).toBe(false);
    expect(useAuthStore.getState().userId).toBeNull();
    expect(useAuthStore.getState().bootstrapStatus).toBe('error');
    expect(useAuthStore.getState().bootstrapError).toBe('device auth unavailable');
    expect(mockSetTokens).toHaveBeenLastCalledWith(null, null);
  });

  it('waits for settings hydration before proceeding', async () => {
    // Start with settings NOT hydrated
    useSettingsStore.setState({ _hasHydrated: false, deviceId: 'stale-device' });

    mockAxiosPost.mockResolvedValueOnce({
      data: {
        data: makeUser(),
        tokens: {
          access_token: 'access-1',
          refresh_token: 'refresh-1',
          expires_in: 3600,
        },
      },
    });

    // Start bootstrap — it should block on hydration
    const bootstrapPromise = bootstrapAnonymousAuth();

    // Simulate hydration completing with the persisted deviceId
    await new Promise((r) => setTimeout(r, 10));
    useSettingsStore.setState({ _hasHydrated: true, deviceId: 'persisted-device' });

    const result = await bootstrapPromise;

    expect(result).toBe(true);
    expect(mockAxiosPost).toHaveBeenCalledWith(
      expect.stringContaining('/auth/device'),
      expect.objectContaining({ device_id: 'persisted-device' }),
      expect.any(Object),
    );
  });

  it('fails with error when settings hydration times out', async () => {
    useSettingsStore.setState({ _hasHydrated: false });

    jest.useFakeTimers();

    const bootstrapPromise = bootstrapAnonymousAuth();

    // Advance past the hydration timeout (5000ms) and flush microtasks
    await jest.advanceTimersByTimeAsync(5001);

    const result = await bootstrapPromise;

    expect(result).toBe(false);
    expect(useAuthStore.getState().bootstrapStatus).toBe('error');
    expect(useAuthStore.getState().bootstrapError).toBe('Settings store hydration timed out.');

    jest.useRealTimers();
  });

  it('proceeds immediately when settings are already hydrated', async () => {
    // _hasHydrated is already true from beforeEach
    mockAxiosPost.mockResolvedValueOnce({
      data: {
        data: makeUser(),
        tokens: {
          access_token: 'access-1',
          refresh_token: 'refresh-1',
          expires_in: 3600,
        },
      },
    });

    const result = await bootstrapAnonymousAuth();

    expect(result).toBe(true);
    expect(useAuthStore.getState().bootstrapStatus).toBe('ready');
  });

  it('updates the stored session through the registered refresh flow', async () => {
    useAuthStore.setState({
      user: makeUser(),
      userId: 'backend-user-123',
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      _hasHydrated: true,
      bootstrapStatus: 'ready',
      bootstrapError: null,
    });
    mockAxiosPost.mockResolvedValueOnce({
      data: {
        tokens: {
          access_token: 'rotated-access',
          refresh_token: 'rotated-refresh',
          expires_in: 3600,
        },
      },
    });

    const token = await refreshAuthSession('old-refresh');

    expect(token).toBe('rotated-access');
    expect(mockAxiosPost).toHaveBeenCalledWith(
      expect.stringContaining('/auth/refresh'),
      { refresh_token: 'old-refresh' },
      expect.any(Object),
    );
    expect(mockSetTokens).toHaveBeenLastCalledWith('rotated-access', 'rotated-refresh');
    expect(useAuthStore.getState().accessToken).toBe('rotated-access');
    expect(useAuthStore.getState().refreshToken).toBe('rotated-refresh');
  });
});
