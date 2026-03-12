import axios from 'axios';
import { setRefreshHandler, setTokens } from '@/api/client';
import type { operations } from '@/api/generated';
import { API_BASE_URL } from '@/constants';
import { useAuthStore } from './useAuthStore';
import { useSettingsStore } from './useSettingsStore';

type DeviceAuthResponse =
  operations['deviceAuth']['responses']['200']['content']['application/json'];
type RefreshTokenResponse =
  operations['refreshToken']['responses']['200']['content']['application/json'];

const AUTH_API_URL = `${API_BASE_URL}/api/v1`;
const AUTH_TIMEOUT_MS = 15000;
const AUTH_WAIT_TIMEOUT_MS = 5000;
const SETTINGS_HYDRATION_TIMEOUT_MS = 5000;

/**
 * Waits for useSettingsStore to finish rehydrating from AsyncStorage.
 * Prevents reading a freshly-generated deviceId instead of the persisted one.
 */
async function awaitSettingsHydration(): Promise<void> {
  if (useSettingsStore.getState()._hasHydrated) {
    return;
  }

  return new Promise<void>((resolve, reject) => {
    const unsubscribe = useSettingsStore.subscribe((state) => {
      if (state._hasHydrated) {
        clearTimeout(timer);
        unsubscribe();
        resolve();
      }
    });

    const timer = setTimeout(() => {
      unsubscribe();
      reject(new Error('Settings store hydration timed out.'));
    }, SETTINGS_HYDRATION_TIMEOUT_MS);
  });
}

function getAuthErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    return error.response?.data?.error ?? error.message ?? 'Authentication failed.';
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return 'Authentication failed.';
}

function applyStoredTokens(): void {
  const { accessToken, refreshToken } = useAuthStore.getState();
  setTokens(accessToken, refreshToken);
}

async function authenticateDevice(): Promise<string> {
  const { deviceId, language } = useSettingsStore.getState();
  const response = await axios.post<DeviceAuthResponse>(
    `${AUTH_API_URL}/auth/device`,
    {
      device_id: deviceId,
      language,
    },
    {
      timeout: AUTH_TIMEOUT_MS,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  const user = response.data.data;
  const tokens = response.data.tokens;

  if (!user?.id || !tokens?.access_token || !tokens.refresh_token) {
    throw new Error('Device authentication returned an incomplete session.');
  }

  useAuthStore.getState().setSession({
    user,
    userId: user.id,
    accessToken: tokens.access_token,
    refreshToken: tokens.refresh_token,
  });
  setTokens(tokens.access_token, tokens.refresh_token);

  return user.id;
}

export async function refreshAuthSession(refreshToken: string): Promise<string | null> {
  try {
    const response = await axios.post<RefreshTokenResponse>(
      `${AUTH_API_URL}/auth/refresh`,
      { refresh_token: refreshToken },
      {
        timeout: AUTH_TIMEOUT_MS,
        headers: { 'Content-Type': 'application/json' },
      },
    );

    const tokens = response.data.tokens;
    const currentUser = useAuthStore.getState().user;
    const currentUserId = useAuthStore.getState().userId ?? currentUser?.id ?? null;

    if (!tokens?.access_token || !tokens.refresh_token || !currentUserId) {
      useAuthStore.getState().clearSession();
      setTokens(null, null);
      return null;
    }

    useAuthStore.getState().setSession({
      user: currentUser,
      userId: currentUserId,
      accessToken: tokens.access_token,
      refreshToken: tokens.refresh_token,
    });
    setTokens(tokens.access_token, tokens.refresh_token);

    return tokens.access_token;
  } catch {
    useAuthStore.getState().clearSession();
    setTokens(null, null);
    return null;
  }
}

export function configureApiClientAuth(): void {
  applyStoredTokens();
  setRefreshHandler(refreshAuthSession);
}

export async function bootstrapAnonymousAuth(): Promise<boolean> {
  const authStore = useAuthStore.getState();
  if (authStore.bootstrapStatus === 'loading') {
    return false;
  }

  try {
    await awaitSettingsHydration();
  } catch (error) {
    useAuthStore.getState().setBootstrapState('error', getAuthErrorMessage(error));
    return false;
  }

  configureApiClientAuth();
  useAuthStore.getState().setBootstrapState('loading');

  try {
    const currentRefreshToken = useAuthStore.getState().refreshToken;

    if (currentRefreshToken) {
      const refreshedToken = await refreshAuthSession(currentRefreshToken);
      if (refreshedToken) {
        useAuthStore.getState().setBootstrapState('ready');
        return true;
      }
    }

    await authenticateDevice();
    useAuthStore.getState().setBootstrapState('ready');
    return true;
  } catch (error) {
    useAuthStore.getState().clearSession();
    useAuthStore.getState().setBootstrapState('error', getAuthErrorMessage(error));
    setTokens(null, null);
    return false;
  }
}

export async function waitForAuthBootstrap(timeoutMs = AUTH_WAIT_TIMEOUT_MS): Promise<boolean> {
  const currentState = useAuthStore.getState();
  if (
    currentState._hasHydrated &&
    currentState.bootstrapStatus !== 'idle' &&
    currentState.bootstrapStatus !== 'loading'
  ) {
    return currentState.bootstrapStatus === 'ready' && !!currentState.userId;
  }

  return new Promise((resolve) => {
    const unsubscribe = useAuthStore.subscribe((state) => {
      if (
        state._hasHydrated &&
        state.bootstrapStatus !== 'idle' &&
        state.bootstrapStatus !== 'loading'
      ) {
        clearTimeout(timer);
        unsubscribe();
        resolve(state.bootstrapStatus === 'ready' && !!state.userId);
      }
    });

    const timer = setTimeout(() => {
      unsubscribe();
      const latestState = useAuthStore.getState();
      resolve(latestState.bootstrapStatus === 'ready' && !!latestState.userId);
    }, timeoutMs);
  });
}

export async function getAuthenticatedUserId(
  timeoutMs = AUTH_WAIT_TIMEOUT_MS,
): Promise<string | null> {
  const ready = await waitForAuthBootstrap(timeoutMs);
  if (!ready) {
    return null;
  }

  return useAuthStore.getState().userId;
}
