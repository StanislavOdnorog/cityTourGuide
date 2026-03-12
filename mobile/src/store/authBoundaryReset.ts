import { notificationManager } from '@/services/notifications';
import { getSyncQueue } from '@/services/sync';
import { useAuthStore } from './useAuthStore';
import { useDownloadStore } from './useDownloadStore';
import { usePlayerStore } from './usePlayerStore';
import { usePurchaseStore } from './usePurchaseStore';
import { useSettingsStore } from './useSettingsStore';

/**
 * Subscribes to auth identity changes and resets user-scoped stores.
 * Call once at app startup (e.g. in _layout.tsx).
 * Returns the unsubscribe function.
 */
export function subscribeAuthBoundaryReset(): () => void {
  let previousAccessToken = useAuthStore.getState().accessToken;
  let previousUserId = useAuthStore.getState().userId;

  const resetUserScopedStores = () => {
    useDownloadStore.getState().clearAllDownloads();
    usePlayerStore.getState().reset();
    usePurchaseStore.setState({ status: null });
    useSettingsStore.getState().clearPushRegistration();
    void notificationManager.unregister();
    void getSyncQueue()?.clearAll();
  };

  return useAuthStore.subscribe((state) => {
    const currentToken = state.accessToken;
    const currentUserId = state.userId;

    // Detect session clear or identity switch without a logout round-trip.
    if (
      (previousAccessToken !== null && currentToken === null) ||
      (previousUserId !== null && currentUserId !== null && previousUserId !== currentUserId)
    ) {
      resetUserScopedStores();
    }

    previousAccessToken = currentToken;
    previousUserId = currentUserId;
  });
}
