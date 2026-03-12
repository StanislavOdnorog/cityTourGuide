jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

jest.mock('expo-notifications', () => ({
  setNotificationHandler: jest.fn(),
  getPermissionsAsync: jest.fn(),
  requestPermissionsAsync: jest.fn(),
  getExpoPushTokenAsync: jest.fn(),
  setNotificationChannelAsync: jest.fn(),
  addNotificationReceivedListener: jest.fn(),
  addNotificationResponseReceivedListener: jest.fn(),
  AndroidImportance: { HIGH: 'high' },
}));

jest.mock('react-native', () => ({
  Platform: { OS: 'ios' },
}));

jest.mock('@/api/endpoints', () => ({
  registerDeviceToken: jest.fn(),
  unregisterDeviceToken: jest.fn(),
}));

jest.mock('@/store/authBootstrap', () => ({
  getAuthenticatedUserId: jest.fn(),
}));

jest.mock('@/store/useSettingsStore', () => ({
  useSettingsStore: {
    getState: jest.fn(),
  },
}));

jest.mock('@/store/useWalkStore', () => ({
  useWalkStore: {
    getState: jest.fn(),
  },
}));

jest.mock('@/store/useAuthStore', () => ({
  useAuthStore: {
    getState: jest.fn(),
  },
}));

import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { registerDeviceToken, unregisterDeviceToken } from '@/api/endpoints';
import { getAuthenticatedUserId } from '@/store/authBootstrap';
import { useAuthStore } from '@/store/useAuthStore';
import { useSettingsStore } from '@/store/useSettingsStore';
import { useWalkStore } from '@/store/useWalkStore';
import { NotificationManager } from '../NotificationManager';

const mockNotifications = Notifications as jest.Mocked<typeof Notifications>;
const mockPlatform = Platform as { OS: 'ios' | 'android' };
const mockRegisterDeviceToken = registerDeviceToken as jest.MockedFunction<
  typeof registerDeviceToken
>;
const mockUnregisterDeviceToken = unregisterDeviceToken as jest.MockedFunction<
  typeof unregisterDeviceToken
>;
const mockGetAuthenticatedUserId = getAuthenticatedUserId as jest.MockedFunction<
  typeof getAuthenticatedUserId
>;
const mockSettingsGetState = useSettingsStore.getState as jest.Mock;
const mockWalkGetState = useWalkStore.getState as jest.Mock;
const mockAuthGetState = useAuthStore.getState as jest.Mock;

function createSubscription() {
  return { remove: jest.fn() };
}

function getLatestNotificationHandler(): {
  handleNotification: () => Promise<{
    shouldShowAlert: boolean;
    shouldShowBanner: boolean;
    shouldShowList: boolean;
    shouldPlaySound: boolean;
    shouldSetBadge: boolean;
  }>;
} {
  const call = mockNotifications.setNotificationHandler.mock.calls.at(-1);
  if (!call) {
    throw new Error('Notification handler was not configured');
  }
  return call[0] as {
    handleNotification: () => Promise<{
      shouldShowAlert: boolean;
      shouldShowBanner: boolean;
      shouldShowList: boolean;
      shouldPlaySound: boolean;
      shouldSetBadge: boolean;
    }>;
  };
}

function defaultSettingsState(overrides: Record<string, unknown> = {}) {
  return {
    geoNotifications: true,
    contentNotifications: true,
    pushToken: null,
    registeredPushUserId: null,
    setPushRegistration: jest.fn(),
    clearPushRegistration: jest.fn(),
    ...overrides,
  };
}

describe('NotificationManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPlatform.OS = 'ios';
    mockNotifications.getPermissionsAsync.mockResolvedValue({ status: 'granted' } as never);
    mockNotifications.requestPermissionsAsync.mockResolvedValue({ status: 'granted' } as never);
    mockNotifications.getExpoPushTokenAsync.mockResolvedValue({ data: 'expo-token-123' } as never);
    mockNotifications.setNotificationChannelAsync.mockResolvedValue(undefined as never);
    mockNotifications.addNotificationReceivedListener.mockImplementation(
      () => createSubscription() as never,
    );
    mockNotifications.addNotificationResponseReceivedListener.mockImplementation(
      () => createSubscription() as never,
    );
    mockRegisterDeviceToken.mockResolvedValue(undefined);
    mockUnregisterDeviceToken.mockResolvedValue(undefined);
    mockGetAuthenticatedUserId.mockResolvedValue('backend-user-123');
    mockSettingsGetState.mockReturnValue(defaultSettingsState());
    mockWalkGetState.mockReturnValue({ isWalking: false });
    mockAuthGetState.mockReturnValue({ userId: 'auth-user-123' });
  });

  it('returns granted immediately when permissions are already available', async () => {
    const manager = new NotificationManager();

    await expect(manager.requestPermissions()).resolves.toBe(true);

    expect(mockNotifications.getPermissionsAsync).toHaveBeenCalledTimes(1);
    expect(mockNotifications.requestPermissionsAsync).not.toHaveBeenCalled();
  });

  it('requests permissions when needed and returns false when the user denies access', async () => {
    mockNotifications.getPermissionsAsync.mockResolvedValue({ status: 'undetermined' } as never);
    mockNotifications.requestPermissionsAsync.mockResolvedValue({ status: 'denied' } as never);
    const manager = new NotificationManager();

    await expect(manager.requestPermissions()).resolves.toBe(false);

    expect(mockNotifications.requestPermissionsAsync).toHaveBeenCalledTimes(1);
  });

  it('registers the Expo push token with the backend for iOS', async () => {
    const manager = new NotificationManager();

    await expect(manager.registerForPushNotifications()).resolves.toBe('expo-token-123');

    expect(mockRegisterDeviceToken).toHaveBeenCalledWith({
      user_id: 'backend-user-123',
      token: 'expo-token-123',
      platform: 'ios',
    });
    expect(mockNotifications.setNotificationChannelAsync).not.toHaveBeenCalled();
  });

  it('persists token and userId in settings store after registration', async () => {
    const settingsState = defaultSettingsState();
    mockSettingsGetState.mockReturnValue(settingsState);
    const manager = new NotificationManager();

    await manager.registerForPushNotifications();

    expect(settingsState.setPushRegistration).toHaveBeenCalledWith(
      'expo-token-123',
      'backend-user-123',
    );
  });

  it('returns null without registering when permissions are denied', async () => {
    mockNotifications.getPermissionsAsync.mockResolvedValue({ status: 'denied' } as never);
    mockNotifications.requestPermissionsAsync.mockResolvedValue({ status: 'denied' } as never);
    const manager = new NotificationManager();

    await expect(manager.registerForPushNotifications()).resolves.toBeNull();

    expect(mockNotifications.getExpoPushTokenAsync).not.toHaveBeenCalled();
    expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
  });

  it('returns null when the Expo push token fetch fails', async () => {
    mockNotifications.getExpoPushTokenAsync.mockRejectedValue(new Error('expo failed'));
    const manager = new NotificationManager();

    await expect(manager.registerForPushNotifications()).resolves.toBeNull();

    expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
  });

  it('configures the Android notification channel after successful registration', async () => {
    mockPlatform.OS = 'android';
    const manager = new NotificationManager();

    await expect(manager.registerForPushNotifications()).resolves.toBe('expo-token-123');

    expect(mockRegisterDeviceToken).toHaveBeenCalledWith({
      user_id: 'backend-user-123',
      token: 'expo-token-123',
      platform: 'android',
    });
    expect(mockNotifications.setNotificationChannelAsync).toHaveBeenCalledWith('city-stories', {
      name: 'City Stories',
      importance: mockNotifications.AndroidImportance.HIGH,
      sound: 'default',
      vibrationPattern: [0, 250, 250, 250],
    });
  });

  it('returns null when there is no authenticated user id', async () => {
    mockGetAuthenticatedUserId.mockResolvedValue(null);
    const manager = new NotificationManager();

    await expect(manager.registerForPushNotifications()).resolves.toBeNull();

    expect(mockNotifications.getPermissionsAsync).not.toHaveBeenCalled();
    expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
  });

  it('unregisters the current token and resets local state', async () => {
    const manager = new NotificationManager();
    await manager.registerForPushNotifications();

    await expect(manager.unregister()).resolves.toBeUndefined();
    await expect(manager.unregister()).resolves.toBeUndefined();

    expect(mockUnregisterDeviceToken).toHaveBeenCalledTimes(1);
    expect(mockUnregisterDeviceToken).toHaveBeenCalledWith('expo-token-123');
  });

  it('unregisters using persisted token when in-memory token is absent', async () => {
    const settingsState = defaultSettingsState({ pushToken: 'persisted-token' });
    mockSettingsGetState.mockReturnValue(settingsState);
    const manager = new NotificationManager();

    await manager.unregister();

    expect(mockUnregisterDeviceToken).toHaveBeenCalledWith('persisted-token');
    expect(settingsState.clearPushRegistration).toHaveBeenCalled();
  });

  it('treats unregister failures as best effort and still clears the token', async () => {
    const manager = new NotificationManager();
    await manager.registerForPushNotifications();
    mockUnregisterDeviceToken.mockRejectedValueOnce(new Error('backend failed'));

    await expect(manager.unregister()).resolves.toBeUndefined();
    await expect(manager.unregister()).resolves.toBeUndefined();

    expect(mockUnregisterDeviceToken).toHaveBeenCalledTimes(1);
  });

  it('replaces duplicate notification received listeners and removes them deterministically', () => {
    const firstSubscription = createSubscription();
    const secondSubscription = createSubscription();
    mockNotifications.addNotificationReceivedListener
      .mockReturnValueOnce(firstSubscription as never)
      .mockReturnValueOnce(secondSubscription as never);
    const manager = new NotificationManager();

    manager.addNotificationReceivedListener(jest.fn());
    manager.addNotificationReceivedListener(jest.fn());
    manager.removeNotificationReceivedListener();
    manager.removeNotificationReceivedListener();

    expect(firstSubscription.remove).toHaveBeenCalledTimes(1);
    expect(secondSubscription.remove).toHaveBeenCalledTimes(1);
  });

  it('replaces duplicate response listeners and dispose removes the active subscriptions', () => {
    const receivedSubscription = createSubscription();
    const responseSubscriptionOne = createSubscription();
    const responseSubscriptionTwo = createSubscription();
    mockNotifications.addNotificationReceivedListener.mockReturnValue(
      receivedSubscription as never,
    );
    mockNotifications.addNotificationResponseReceivedListener
      .mockReturnValueOnce(responseSubscriptionOne as never)
      .mockReturnValueOnce(responseSubscriptionTwo as never);
    const manager = new NotificationManager();

    manager.addNotificationReceivedListener(jest.fn());
    manager.addNotificationResponseListener(jest.fn());
    manager.addNotificationResponseListener(jest.fn());
    manager.dispose();
    manager.dispose();

    expect(responseSubscriptionOne.remove).toHaveBeenCalledTimes(1);
    expect(responseSubscriptionTwo.remove).toHaveBeenCalledTimes(1);
    expect(receivedSubscription.remove).toHaveBeenCalledTimes(1);
  });

  it('suppresses foreground notifications while a walk is in progress', async () => {
    mockWalkGetState.mockReturnValue({ isWalking: true });
    new NotificationManager();

    await expect(getLatestNotificationHandler().handleNotification()).resolves.toEqual({
      shouldShowAlert: false,
      shouldShowBanner: false,
      shouldShowList: false,
      shouldPlaySound: false,
      shouldSetBadge: false,
    });
  });

  it('shows foreground notifications when no walk is active', async () => {
    new NotificationManager();

    await expect(getLatestNotificationHandler().handleNotification()).resolves.toEqual({
      shouldShowAlert: true,
      shouldShowBanner: true,
      shouldShowList: true,
      shouldPlaySound: true,
      shouldSetBadge: false,
    });
  });

  it('respects settings toggles for geo and content notification types', () => {
    mockSettingsGetState.mockReturnValue(
      defaultSettingsState({ geoNotifications: false, contentNotifications: true }),
    );
    const manager = new NotificationManager();

    expect(manager.isTypeEnabled('geo')).toBe(false);
    expect(manager.isTypeEnabled('content')).toBe(true);
  });

  it('reads the registered user id from auth store state', () => {
    const manager = new NotificationManager();

    expect(manager.getRegisteredUserId()).toBe('auth-user-123');
  });

  describe('reconcile', () => {
    it('skips registration when the same token is stored for the same user', async () => {
      mockAuthGetState.mockReturnValue({ userId: 'user-1' });
      mockSettingsGetState.mockReturnValue(
        defaultSettingsState({
          pushToken: 'expo-token-123',
          registeredPushUserId: 'user-1',
        }),
      );
      const manager = new NotificationManager();

      await manager.reconcile();

      expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
      expect(mockUnregisterDeviceToken).not.toHaveBeenCalled();
    });

    it('registers when no stored registration exists for an authenticated user', async () => {
      mockAuthGetState.mockReturnValue({ userId: 'user-1' });
      mockSettingsGetState.mockReturnValue(defaultSettingsState());
      const manager = new NotificationManager();

      await manager.reconcile();

      expect(mockRegisterDeviceToken).toHaveBeenCalledTimes(1);
    });

    it('unregisters and re-registers when a different user is active', async () => {
      const settingsState = defaultSettingsState({
        pushToken: 'old-token',
        registeredPushUserId: 'user-old',
      });
      mockAuthGetState.mockReturnValue({ userId: 'user-new' });
      mockSettingsGetState.mockReturnValue(settingsState);
      const manager = new NotificationManager();

      await manager.reconcile();

      expect(mockUnregisterDeviceToken).toHaveBeenCalledWith('old-token');
      expect(mockRegisterDeviceToken).toHaveBeenCalledTimes(1);
    });

    it('unregisters when there is no authenticated user but a stored token exists', async () => {
      const settingsState = defaultSettingsState({
        pushToken: 'stale-token',
        registeredPushUserId: 'user-gone',
      });
      mockAuthGetState.mockReturnValue({ userId: null });
      mockSettingsGetState.mockReturnValue(settingsState);
      const manager = new NotificationManager();

      await manager.reconcile();

      expect(mockUnregisterDeviceToken).toHaveBeenCalledWith('stale-token');
      expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
    });

    it('unregisters when notifications are fully disabled', async () => {
      const settingsState = defaultSettingsState({
        pushToken: 'existing-token',
        registeredPushUserId: 'user-1',
        geoNotifications: false,
        contentNotifications: false,
      });
      mockAuthGetState.mockReturnValue({ userId: 'user-1' });
      mockSettingsGetState.mockReturnValue(settingsState);
      const manager = new NotificationManager();

      await manager.reconcile();

      expect(mockUnregisterDeviceToken).toHaveBeenCalledWith('existing-token');
      expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
    });

    it('does nothing when no auth user and no stored token', async () => {
      mockAuthGetState.mockReturnValue({ userId: null });
      mockSettingsGetState.mockReturnValue(defaultSettingsState());
      const manager = new NotificationManager();

      await manager.reconcile();

      expect(mockRegisterDeviceToken).not.toHaveBeenCalled();
      expect(mockUnregisterDeviceToken).not.toHaveBeenCalled();
    });

    it('does not crash when registration fails during reconcile', async () => {
      mockAuthGetState.mockReturnValue({ userId: 'user-1' });
      mockSettingsGetState.mockReturnValue(defaultSettingsState());
      mockNotifications.getExpoPushTokenAsync.mockRejectedValue(new Error('expo error'));
      const manager = new NotificationManager();

      await expect(manager.reconcile()).resolves.toBeUndefined();
    });

    it('does not crash when unregister fails during reconcile', async () => {
      const settingsState = defaultSettingsState({
        pushToken: 'token',
        registeredPushUserId: 'old-user',
      });
      mockAuthGetState.mockReturnValue({ userId: null });
      mockSettingsGetState.mockReturnValue(settingsState);
      mockUnregisterDeviceToken.mockRejectedValueOnce(new Error('backend error'));
      const manager = new NotificationManager();

      await expect(manager.reconcile()).resolves.toBeUndefined();
    });
  });
});
