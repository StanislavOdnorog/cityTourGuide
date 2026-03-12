import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { registerDeviceToken, unregisterDeviceToken } from '@/api/endpoints';
import { getAuthenticatedUserId } from '@/store/authBootstrap';
import { useAuthStore } from '@/store/useAuthStore';
import { useSettingsStore } from '@/store/useSettingsStore';
import { useWalkStore } from '@/store/useWalkStore';

export type NotificationData = {
  type: 'geo' | 'content';
  [key: string]: string;
};

export type NotificationReceivedListener = (notification: Notifications.Notification) => void;

export type NotificationResponseListener = (response: Notifications.NotificationResponse) => void;

/**
 * NotificationManager handles push notification registration,
 * permission requests, and notification event listeners.
 */
export class NotificationManager {
  private notificationListener: Notifications.EventSubscription | null = null;
  private responseListener: Notifications.EventSubscription | null = null;
  private currentToken: string | null = null;

  constructor() {
    this.configureForegroundNotificationHandler();
  }

  private configureForegroundNotificationHandler(): void {
    Notifications.setNotificationHandler({
      handleNotification: async () => this.getForegroundNotificationBehavior(),
    });
  }

  private getForegroundNotificationBehavior() {
    if (useWalkStore.getState().isWalking) {
      return {
        shouldShowAlert: false,
        shouldShowBanner: false,
        shouldShowList: false,
        shouldPlaySound: false,
        shouldSetBadge: false,
      };
    }

    return {
      shouldShowAlert: true,
      shouldShowBanner: true,
      shouldShowList: true,
      shouldPlaySound: true,
      shouldSetBadge: false,
    };
  }

  private async setupAndroidNotificationChannel(): Promise<void> {
    if (Platform.OS !== 'android') {
      return;
    }

    await Notifications.setNotificationChannelAsync('city-stories', {
      name: 'City Stories',
      importance: Notifications.AndroidImportance.HIGH,
      sound: 'default',
      vibrationPattern: [0, 250, 250, 250],
    });
  }

  /**
   * Request notification permissions from the user.
   * Returns true if permission was granted.
   */
  async requestPermissions(): Promise<boolean> {
    const { status: existingStatus } = await Notifications.getPermissionsAsync();
    if (existingStatus === 'granted') {
      return true;
    }

    const { status } = await Notifications.requestPermissionsAsync();
    return status === 'granted';
  }

  /**
   * Get the current permission status.
   */
  async getPermissionStatus(): Promise<Notifications.PermissionStatus> {
    const { status } = await Notifications.getPermissionsAsync();
    return status;
  }

  /**
   * Register the device for push notifications with the backend.
   * Gets the Expo Push Token and sends it to the API.
   */
  async registerForPushNotifications(): Promise<string | null> {
    const userId = await getAuthenticatedUserId();
    if (!userId) {
      return null;
    }

    const granted = await this.requestPermissions();
    if (!granted) {
      return null;
    }

    try {
      const tokenData = await Notifications.getExpoPushTokenAsync();
      const token = tokenData.data;
      const platform = Platform.OS === 'ios' ? 'ios' : 'android';

      await registerDeviceToken({ user_id: userId, token, platform });
      this.currentToken = token;
      useSettingsStore.getState().setPushRegistration(token, userId);
      await this.setupAndroidNotificationChannel();

      return token;
    } catch {
      return null;
    }
  }

  /**
   * Unregister the device from push notifications.
   */
  async unregister(): Promise<void> {
    const token = this.currentToken ?? useSettingsStore.getState().pushToken;
    if (token) {
      try {
        await unregisterDeviceToken(token);
      } catch {
        // Best effort — token may already be invalid
      }
      this.currentToken = null;
      useSettingsStore.getState().clearPushRegistration();
    }
  }

  /**
   * Reconcile push notification registration state with the current
   * authenticated user and notification preferences.
   *
   * Call after auth bootstrap completes and on notification setting changes.
   * - Skips registration when the same token is already stored for the active user.
   * - Unregisters on auth loss or when notifications are fully disabled.
   * - Re-registers when a different user is present.
   */
  async reconcile(): Promise<void> {
    const authUserId = useAuthStore.getState().userId;
    const settings = useSettingsStore.getState();
    const { pushToken: storedToken, registeredPushUserId } = settings;
    const notificationsEnabled = settings.geoNotifications || settings.contentNotifications;

    // No authenticated user → clear any stale registration
    if (!authUserId) {
      if (storedToken) {
        await this.unregister();
      }
      return;
    }

    // Notifications fully disabled → unregister if registered
    if (!notificationsEnabled) {
      if (storedToken) {
        await this.unregister();
      }
      return;
    }

    // Different user than what's stored → unregister old, register new
    if (storedToken && registeredPushUserId && registeredPushUserId !== authUserId) {
      await this.unregister();
      await this.registerForPushNotifications();
      return;
    }

    // Same user + same token already stored → skip duplicate registration
    if (storedToken && registeredPushUserId === authUserId) {
      this.currentToken = storedToken;
      return;
    }

    // No stored registration → register
    await this.registerForPushNotifications();
  }

  /**
   * Add a listener for incoming notifications (while app is in foreground).
   */
  addNotificationReceivedListener(listener: NotificationReceivedListener): void {
    this.removeNotificationReceivedListener();
    this.notificationListener = Notifications.addNotificationReceivedListener(listener);
  }

  /**
   * Add a listener for notification responses (user tapped notification).
   */
  addNotificationResponseListener(listener: NotificationResponseListener): void {
    this.removeNotificationResponseListener();
    this.responseListener = Notifications.addNotificationResponseReceivedListener(listener);
  }

  /**
   * Remove the notification received listener.
   */
  removeNotificationReceivedListener(): void {
    if (this.notificationListener) {
      this.notificationListener.remove();
      this.notificationListener = null;
    }
  }

  /**
   * Remove the notification response listener.
   */
  removeNotificationResponseListener(): void {
    if (this.responseListener) {
      this.responseListener.remove();
      this.responseListener = null;
    }
  }

  /**
   * Clean up all listeners.
   */
  dispose(): void {
    this.removeNotificationReceivedListener();
    this.removeNotificationResponseListener();
  }

  /**
   * Check whether notification preferences allow sending a specific type.
   */
  isTypeEnabled(type: 'geo' | 'content'): boolean {
    const settings = useSettingsStore.getState();
    if (type === 'geo') return settings.geoNotifications;
    if (type === 'content') return settings.contentNotifications;
    return false;
  }

  getRegisteredUserId(): string | null {
    return useAuthStore.getState().userId;
  }
}

// Singleton instance
export const notificationManager = new NotificationManager();
