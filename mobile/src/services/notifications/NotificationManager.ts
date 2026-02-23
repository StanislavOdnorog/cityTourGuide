import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { registerDeviceToken, unregisterDeviceToken } from '@/api/endpoints';
import { useSettingsStore } from '@/store/useSettingsStore';
import { useWalkStore } from '@/store/useWalkStore';

// Configure notification handler for foreground notifications
Notifications.setNotificationHandler({
  handleNotification: async () => {
    const isWalking = getWalkingState();
    // Suppress notifications during active walking session
    if (isWalking) {
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
  },
});

function getWalkingState(): boolean {
  return useWalkStore.getState().isWalking;
}

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
  async registerForPushNotifications(userId: string): Promise<string | null> {
    const granted = await this.requestPermissions();
    if (!granted) {
      return null;
    }

    try {
      const tokenData = await Notifications.getExpoPushTokenAsync();
      const token = tokenData.data;
      const platform = Platform.OS === 'ios' ? 'ios' : 'android';

      await registerDeviceToken(userId, token, platform);
      this.currentToken = token;

      // Set up Android notification channel
      if (Platform.OS === 'android') {
        await Notifications.setNotificationChannelAsync('city-stories', {
          name: 'City Stories',
          importance: Notifications.AndroidImportance.HIGH,
          sound: 'default',
          vibrationPattern: [0, 250, 250, 250],
        });
      }

      return token;
    } catch {
      return null;
    }
  }

  /**
   * Unregister the device from push notifications.
   */
  async unregister(): Promise<void> {
    if (this.currentToken) {
      try {
        await unregisterDeviceToken(this.currentToken);
      } catch {
        // Best effort — token may already be invalid
      }
      this.currentToken = null;
    }
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
}

// Singleton instance
export const notificationManager = new NotificationManager();
