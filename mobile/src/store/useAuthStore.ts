import AsyncStorage from '@react-native-async-storage/async-storage';
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';
import type { User } from '@/types';

export type AuthBootstrapStatus = 'idle' | 'loading' | 'ready' | 'error';

interface PersistedAuthState {
  user: User | null;
  userId: string | null;
  accessToken: string | null;
  refreshToken: string | null;
}

interface AuthState extends PersistedAuthState {
  _hasHydrated: boolean;
  bootstrapStatus: AuthBootstrapStatus;
  bootstrapError: string | null;
}

interface AuthActions {
  setSession: (session: PersistedAuthState) => void;
  clearSession: () => void;
  setHasHydrated: (value: boolean) => void;
  setBootstrapState: (status: AuthBootstrapStatus, error?: string | null) => void;
}

const DEFAULT_STATE: AuthState = {
  user: null,
  userId: null,
  accessToken: null,
  refreshToken: null,
  _hasHydrated: false,
  bootstrapStatus: 'idle',
  bootstrapError: null,
};

function isStringOrNull(value: unknown): value is string | null {
  return typeof value === 'string' || value === null || value === undefined;
}

function sanitizePersistedAuthState(value: unknown): PersistedAuthState {
  if (!value || typeof value !== 'object') {
    return {
      user: DEFAULT_STATE.user,
      userId: DEFAULT_STATE.userId,
      accessToken: DEFAULT_STATE.accessToken,
      refreshToken: DEFAULT_STATE.refreshToken,
    };
  }

  const persisted = value as Partial<PersistedAuthState>;
  const user =
    persisted.user && typeof persisted.user === 'object' && typeof persisted.user.id === 'string'
      ? (persisted.user as User)
      : null;
  const userId = typeof persisted.userId === 'string' ? persisted.userId : (user?.id ?? null);
  const accessToken = typeof persisted.accessToken === 'string' ? persisted.accessToken : null;
  const refreshToken = typeof persisted.refreshToken === 'string' ? persisted.refreshToken : null;

  if (
    !isStringOrNull(persisted.userId) ||
    !isStringOrNull(persisted.accessToken) ||
    !isStringOrNull(persisted.refreshToken)
  ) {
    return {
      user: null,
      userId: null,
      accessToken: null,
      refreshToken: null,
    };
  }

  return {
    user,
    userId,
    accessToken,
    refreshToken,
  };
}

export const useAuthStore = create<AuthState & AuthActions>()(
  persist(
    (set) => ({
      ...DEFAULT_STATE,
      setSession: ({ user, userId, accessToken, refreshToken }) =>
        set({
          user,
          userId,
          accessToken,
          refreshToken,
          bootstrapStatus: 'ready',
          bootstrapError: null,
        }),
      clearSession: () =>
        set({
          user: null,
          userId: null,
          accessToken: null,
          refreshToken: null,
        }),
      setHasHydrated: (value) => set({ _hasHydrated: value }),
      setBootstrapState: (status, error = null) =>
        set({ bootstrapStatus: status, bootstrapError: error ?? null }),
    }),
    {
      name: 'city-stories-auth',
      storage: createJSONStorage(() => AsyncStorage),
      version: 1,
      migrate: (persisted, version) => {
        const state = (persisted ?? {}) as Partial<PersistedAuthState>;
        if (version === 0) {
          return {
            user: state.user ?? null,
            userId: state.userId ?? null,
            accessToken: state.accessToken ?? null,
            refreshToken: state.refreshToken ?? null,
          };
        }
        return state as PersistedAuthState;
      },
      partialize: (state) => ({
        user: state.user,
        userId: state.userId,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
      }),
      merge: (persistedState, currentState) => ({
        ...currentState,
        ...sanitizePersistedAuthState(persistedState),
      }),
      onRehydrateStorage: () => (_state, error) => {
        useAuthStore.setState({
          _hasHydrated: true,
          bootstrapStatus: 'idle',
          bootstrapError: error ? 'Failed to restore authentication state.' : null,
        });
      },
    },
  ),
);

export function resetAuthStore(): void {
  useAuthStore.setState(DEFAULT_STATE);
}
