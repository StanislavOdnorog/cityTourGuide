import { Alert } from 'react-native';
import type { StoryCacheManager, CacheStats } from '@/services/cache';
import type { NotificationManager } from '@/services/notifications/NotificationManager';

/**
 * Ensures push notifications are registered.
 * Returns true if a token is already present or registration succeeds.
 * Shows an alert and returns false if registration fails.
 */
export async function ensurePushRegistered(
  pushToken: string | null,
  setPushToken: (token: string | null) => void,
  manager: NotificationManager,
): Promise<boolean> {
  if (pushToken) return true;
  const token = await manager.registerForPushNotifications();
  if (token) {
    setPushToken(token);
    return true;
  }
  Alert.alert(
    'Notifications Disabled',
    'Please enable notifications in your device settings to receive alerts.',
  );
  return false;
}

/**
 * Handles toggling a notification type (geo or content).
 * When enabling, ensures push registration first; if registration fails the toggle stays off.
 */
export async function handleNotificationToggle(
  enabled: boolean,
  pushToken: string | null,
  setPushToken: (token: string | null) => void,
  setNotificationEnabled: (enabled: boolean) => void,
  manager: NotificationManager,
): Promise<void> {
  if (enabled) {
    const ok = await ensurePushRegistered(pushToken, setPushToken, manager);
    if (!ok) return;
  }
  setNotificationEnabled(enabled);
}

/**
 * Clears all cached audio files and refreshes stats.
 * Always resets the clearing flag, even on error.
 */
export async function clearCache(
  cacheManager: StoryCacheManager,
  setIsClearing: (clearing: boolean) => void,
  setCacheStats: (stats: CacheStats) => void,
  clearAllDownloads: () => void,
): Promise<void> {
  setIsClearing(true);
  try {
    await cacheManager.clearAll();
    clearAllDownloads();
    const stats = await cacheManager.getStats();
    setCacheStats(stats);
  } catch {
    // Silently handle cache clear errors
  } finally {
    setIsClearing(false);
  }
}
