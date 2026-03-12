import { describe, expect, it } from 'vitest';
import type { User } from '../types';
import { useAuthStore } from './authStore';

const TOKEN_KEY = 'csg_admin_token';
const REFRESH_KEY = 'csg_admin_refresh';
const USER_KEY = 'csg_admin_user';

const user: User = {
  id: '550e8400-e29b-41d4-a716-446655440000',
  email: 'admin@example.com',
  name: 'Admin User',
  auth_provider: 'email',
  provider_id: null,
  language_pref: 'en',
  is_anonymous: false,
  is_admin: true,
  deleted_at: null,
  deletion_scheduled_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

describe('authStore', () => {
  it('persists auth state with setAuth', () => {
    useAuthStore.getState().setAuth('access-token', 'refresh-token', user);

    expect(useAuthStore.getState()).toMatchObject({
      token: 'access-token',
      refreshToken: 'refresh-token',
      user,
      isAuthenticated: true,
    });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('access-token');
    expect(localStorage.getItem(REFRESH_KEY)).toBe('refresh-token');
    expect(localStorage.getItem(USER_KEY)).toBe(JSON.stringify(user));
  });

  it('hydrates a stored session', () => {
    localStorage.setItem(TOKEN_KEY, 'stored-access');
    localStorage.setItem(REFRESH_KEY, 'stored-refresh');
    localStorage.setItem(USER_KEY, JSON.stringify(user));

    useAuthStore.getState().hydrateFromStorage();

    expect(useAuthStore.getState()).toMatchObject({
      token: 'stored-access',
      refreshToken: 'stored-refresh',
      user,
      isAuthenticated: true,
      isHydrated: true,
    });
  });

  it('cleans up malformed stored user data', () => {
    localStorage.setItem(TOKEN_KEY, 'stored-access');
    localStorage.setItem(REFRESH_KEY, 'stored-refresh');
    localStorage.setItem(USER_KEY, '{bad-json');

    useAuthStore.getState().hydrateFromStorage();

    expect(useAuthStore.getState()).toMatchObject({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isHydrated: true,
    });
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(localStorage.getItem(REFRESH_KEY)).toBeNull();
    expect(localStorage.getItem(USER_KEY)).toBeNull();
  });

  it('updates and persists refreshed tokens', () => {
    useAuthStore.getState().setAuth('access-token', 'refresh-token', user);

    useAuthStore.getState().setTokens('next-access', 'next-refresh');

    expect(useAuthStore.getState()).toMatchObject({
      token: 'next-access',
      refreshToken: 'next-refresh',
      user,
      isAuthenticated: true,
    });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('next-access');
    expect(localStorage.getItem(REFRESH_KEY)).toBe('next-refresh');
  });

  it('removes auth state on logout', () => {
    useAuthStore.getState().setAuth('access-token', 'refresh-token', user);

    useAuthStore.getState().logout();

    expect(useAuthStore.getState()).toMatchObject({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
    });
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(localStorage.getItem(REFRESH_KEY)).toBeNull();
    expect(localStorage.getItem(USER_KEY)).toBeNull();
  });
});
