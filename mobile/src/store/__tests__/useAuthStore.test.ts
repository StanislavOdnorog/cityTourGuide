import AsyncStorage from '@react-native-async-storage/async-storage';
import type { User } from '@/types';
import { resetAuthStore, useAuthStore } from '../useAuthStore';

jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

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

describe('useAuthStore', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    resetAuthStore();
  });

  it('has the expected default state', () => {
    const state = useAuthStore.getState();

    expect(state.userId).toBeNull();
    expect(state.accessToken).toBeNull();
    expect(state.refreshToken).toBeNull();
    expect(state.bootstrapStatus).toBe('idle');
    expect(state.bootstrapError).toBeNull();
  });

  it('returns defaults before hydration completes', () => {
    resetAuthStore();
    const state = useAuthStore.getState();
    expect(state._hasHydrated).toBe(false);
    expect(state.user).toBeNull();
    expect(state.userId).toBeNull();
    expect(state.accessToken).toBeNull();
    expect(state.refreshToken).toBeNull();
    expect(state.bootstrapStatus).toBe('idle');
  });

  it('hydrates a valid persisted auth session', async () => {
    mockGetItem.mockResolvedValueOnce(
      JSON.stringify({
        state: {
          user: makeUser(),
          userId: 'backend-user-123',
          accessToken: 'access-token',
          refreshToken: 'refresh-token',
        },
        version: 0,
      }),
    );

    await useAuthStore.persist.rehydrate();

    const state = useAuthStore.getState();
    expect(state._hasHydrated).toBe(true);
    expect(state.userId).toBe('backend-user-123');
    expect(state.accessToken).toBe('access-token');
    expect(state.refreshToken).toBe('refresh-token');
    expect(state.user?.id).toBe('backend-user-123');
  });

  it('drops invalid persisted auth payloads during hydration', async () => {
    mockGetItem.mockResolvedValueOnce(
      JSON.stringify({
        state: {
          user: { id: 42 },
          userId: 42,
          accessToken: { bad: true },
          refreshToken: ['bad'],
        },
        version: 0,
      }),
    );

    await useAuthStore.persist.rehydrate();

    const state = useAuthStore.getState();
    expect(state._hasHydrated).toBe(true);
    expect(state.user).toBeNull();
    expect(state.userId).toBeNull();
    expect(state.accessToken).toBeNull();
    expect(state.refreshToken).toBeNull();
  });
});
