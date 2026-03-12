import type { QueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../store/authStore';

let _queryClient: QueryClient | null = null;

/** Call once from App to register the shared QueryClient instance. */
export function registerQueryClient(qc: QueryClient) {
  _queryClient = qc;
}

/**
 * Centralized auth reset: cancels pending queries, clears the React Query
 * cache, wipes persisted auth state, and resets the Zustand auth store.
 *
 * Safe to call from anywhere (interceptors, UI handlers, error boundaries).
 */
export function resetAuth() {
  // Cancel all in-flight queries so they don't repopulate cache after clear
  if (_queryClient) {
    _queryClient.cancelQueries();
    _queryClient.clear();
  }

  // Clear Zustand store + localStorage
  useAuthStore.getState().logout();
}
