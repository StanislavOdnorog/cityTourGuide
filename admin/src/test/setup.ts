import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach, beforeEach, vi } from 'vitest';
import { useAuthStore } from '../store/authStore';

function resetAuthStore() {
  useAuthStore.setState({
    token: null,
    refreshToken: null,
    user: null,
    isAuthenticated: false,
    isHydrated: false,
  });
}

beforeEach(() => {
  localStorage.clear();
  resetAuthStore();
});

afterEach(() => {
  cleanup();
  localStorage.clear();
  resetAuthStore();
  vi.clearAllMocks();
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});
