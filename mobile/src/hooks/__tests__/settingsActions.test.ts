import { Alert } from 'react-native';
import { ensurePushRegistered, handleNotificationToggle, clearCache } from '../settingsActions';

// ---------- Mocks ----------

jest.mock('react-native', () => ({
  Alert: { alert: jest.fn() },
}));

function createMockNotificationManager() {
  return {
    registerForPushNotifications: jest.fn() as jest.Mock<Promise<string | null>>,
  } as { registerForPushNotifications: jest.Mock<Promise<string | null>> };
}

function createMockCacheManager() {
  return {
    clearAll: jest.fn() as jest.Mock<Promise<void>>,
    getStats: jest.fn() as jest.Mock<
      Promise<{ totalSizeBytes: number; cachedFileCount: number; maxSizeBytes: number }>
    >,
  };
}

// ---------- ensurePushRegistered ----------

describe('ensurePushRegistered', () => {
  beforeEach(() => jest.clearAllMocks());

  it('returns true immediately when a token already exists', async () => {
    const setPushToken = jest.fn();
    const manager = createMockNotificationManager();

    const result = await ensurePushRegistered(
      'existing-token',
      setPushToken,
      manager as unknown as Parameters<typeof ensurePushRegistered>[2],
    );

    expect(result).toBe(true);
    expect(manager.registerForPushNotifications).not.toHaveBeenCalled();
    expect(setPushToken).not.toHaveBeenCalled();
  });

  it('registers, persists the token, and returns true on success', async () => {
    const setPushToken = jest.fn();
    const manager = createMockNotificationManager();
    manager.registerForPushNotifications.mockResolvedValue('new-token-abc');

    const result = await ensurePushRegistered(
      null,
      setPushToken,
      manager as unknown as Parameters<typeof ensurePushRegistered>[2],
    );

    expect(result).toBe(true);
    expect(manager.registerForPushNotifications).toHaveBeenCalledTimes(1);
    expect(setPushToken).toHaveBeenCalledWith('new-token-abc');
  });

  it('shows alert and returns false when registration returns null', async () => {
    const setPushToken = jest.fn();
    const manager = createMockNotificationManager();
    manager.registerForPushNotifications.mockResolvedValue(null);

    const result = await ensurePushRegistered(
      null,
      setPushToken,
      manager as unknown as Parameters<typeof ensurePushRegistered>[2],
    );

    expect(result).toBe(false);
    expect(setPushToken).not.toHaveBeenCalled();
    expect(Alert.alert).toHaveBeenCalledWith(
      'Notifications Disabled',
      'Please enable notifications in your device settings to receive alerts.',
    );
  });
});

// ---------- handleNotificationToggle — geo ----------

describe('handleNotificationToggle (geo)', () => {
  beforeEach(() => jest.clearAllMocks());

  it('enables geo notifications after successful registration', async () => {
    const setPushToken = jest.fn();
    const setGeo = jest.fn();
    const manager = createMockNotificationManager();
    manager.registerForPushNotifications.mockResolvedValue('token-geo');

    await handleNotificationToggle(
      true,
      null,
      setPushToken,
      setGeo,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(manager.registerForPushNotifications).toHaveBeenCalledTimes(1);
    expect(setPushToken).toHaveBeenCalledWith('token-geo');
    expect(setGeo).toHaveBeenCalledWith(true);
  });

  it('enables geo notifications without registering when token exists', async () => {
    const setPushToken = jest.fn();
    const setGeo = jest.fn();
    const manager = createMockNotificationManager();

    await handleNotificationToggle(
      true,
      'existing',
      setPushToken,
      setGeo,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(manager.registerForPushNotifications).not.toHaveBeenCalled();
    expect(setGeo).toHaveBeenCalledWith(true);
  });

  it('does not enable geo when registration fails', async () => {
    const setPushToken = jest.fn();
    const setGeo = jest.fn();
    const manager = createMockNotificationManager();
    manager.registerForPushNotifications.mockResolvedValue(null);

    await handleNotificationToggle(
      true,
      null,
      setPushToken,
      setGeo,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(setGeo).not.toHaveBeenCalled();
    expect(Alert.alert).toHaveBeenCalledWith('Notifications Disabled', expect.any(String));
  });

  it('disables geo notifications without registration', async () => {
    const setPushToken = jest.fn();
    const setGeo = jest.fn();
    const manager = createMockNotificationManager();

    await handleNotificationToggle(
      false,
      null,
      setPushToken,
      setGeo,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(manager.registerForPushNotifications).not.toHaveBeenCalled();
    expect(setGeo).toHaveBeenCalledWith(false);
  });
});

// ---------- handleNotificationToggle — content ----------

describe('handleNotificationToggle (content)', () => {
  beforeEach(() => jest.clearAllMocks());

  it('enables content notifications after successful registration', async () => {
    const setPushToken = jest.fn();
    const setContent = jest.fn();
    const manager = createMockNotificationManager();
    manager.registerForPushNotifications.mockResolvedValue('token-content');

    await handleNotificationToggle(
      true,
      null,
      setPushToken,
      setContent,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(manager.registerForPushNotifications).toHaveBeenCalledTimes(1);
    expect(setPushToken).toHaveBeenCalledWith('token-content');
    expect(setContent).toHaveBeenCalledWith(true);
  });

  it('enables content notifications without registering when token exists', async () => {
    const setPushToken = jest.fn();
    const setContent = jest.fn();
    const manager = createMockNotificationManager();

    await handleNotificationToggle(
      true,
      'existing',
      setPushToken,
      setContent,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(manager.registerForPushNotifications).not.toHaveBeenCalled();
    expect(setContent).toHaveBeenCalledWith(true);
  });

  it('does not enable content when registration fails', async () => {
    const setPushToken = jest.fn();
    const setContent = jest.fn();
    const manager = createMockNotificationManager();
    manager.registerForPushNotifications.mockResolvedValue(null);

    await handleNotificationToggle(
      true,
      null,
      setPushToken,
      setContent,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(setContent).not.toHaveBeenCalled();
    expect(Alert.alert).toHaveBeenCalledWith('Notifications Disabled', expect.any(String));
  });

  it('disables content notifications without registration', async () => {
    const setPushToken = jest.fn();
    const setContent = jest.fn();
    const manager = createMockNotificationManager();

    await handleNotificationToggle(
      false,
      'existing',
      setPushToken,
      setContent,
      manager as unknown as Parameters<typeof handleNotificationToggle>[4],
    );

    expect(manager.registerForPushNotifications).not.toHaveBeenCalled();
    expect(setContent).toHaveBeenCalledWith(false);
  });
});

// ---------- clearCache ----------

describe('clearCache', () => {
  beforeEach(() => jest.clearAllMocks());

  it('clears cache, refreshes stats, and resets clearing flag on success', async () => {
    const cm = createMockCacheManager();
    const setIsClearing = jest.fn();
    const setCacheStats = jest.fn();
    const clearAllDownloads = jest.fn();

    const freshStats = { totalSizeBytes: 0, cachedFileCount: 0, maxSizeBytes: 104857600 };
    cm.clearAll.mockResolvedValue(undefined);
    cm.getStats.mockResolvedValue(freshStats);

    await clearCache(
      cm as unknown as Parameters<typeof clearCache>[0],
      setIsClearing,
      setCacheStats,
      clearAllDownloads,
    );

    // setIsClearing(true) called first
    expect(setIsClearing).toHaveBeenNthCalledWith(1, true);
    // cache cleared
    expect(cm.clearAll).toHaveBeenCalledTimes(1);
    // downloads store cleared
    expect(clearAllDownloads).toHaveBeenCalledTimes(1);
    // stats refreshed
    expect(cm.getStats).toHaveBeenCalledTimes(1);
    expect(setCacheStats).toHaveBeenCalledWith(freshStats);
    // setIsClearing(false) called last
    expect(setIsClearing).toHaveBeenNthCalledWith(2, false);
  });

  it('resets clearing flag even when clearAll throws', async () => {
    const cm = createMockCacheManager();
    const setIsClearing = jest.fn();
    const setCacheStats = jest.fn();
    const clearAllDownloads = jest.fn();

    cm.clearAll.mockRejectedValue(new Error('disk full'));

    await clearCache(
      cm as unknown as Parameters<typeof clearCache>[0],
      setIsClearing,
      setCacheStats,
      clearAllDownloads,
    );

    expect(setIsClearing).toHaveBeenNthCalledWith(1, true);
    // stats should NOT be refreshed since clearAll threw
    expect(setCacheStats).not.toHaveBeenCalled();
    // clearing flag must still be reset
    expect(setIsClearing).toHaveBeenNthCalledWith(2, false);
  });

  it('resets clearing flag even when getStats throws after successful clear', async () => {
    const cm = createMockCacheManager();
    const setIsClearing = jest.fn();
    const setCacheStats = jest.fn();
    const clearAllDownloads = jest.fn();

    cm.clearAll.mockResolvedValue(undefined);
    cm.getStats.mockRejectedValue(new Error('db error'));

    await clearCache(
      cm as unknown as Parameters<typeof clearCache>[0],
      setIsClearing,
      setCacheStats,
      clearAllDownloads,
    );

    expect(cm.clearAll).toHaveBeenCalledTimes(1);
    expect(clearAllDownloads).toHaveBeenCalledTimes(1);
    // setCacheStats not called because getStats threw
    expect(setCacheStats).not.toHaveBeenCalled();
    // clearing flag still reset
    expect(setIsClearing).toHaveBeenNthCalledWith(2, false);
  });

  it('calls operations in the correct order', async () => {
    const order: string[] = [];
    const cm = createMockCacheManager();
    const setIsClearing = jest.fn((v) => order.push(`clearing:${v}`));
    const setCacheStats = jest.fn(() => order.push('setStats'));
    const clearAllDownloads = jest.fn(() => order.push('clearDownloads'));

    cm.clearAll.mockImplementation(async () => {
      order.push('clearAll');
    });
    cm.getStats.mockImplementation(async () => {
      order.push('getStats');
      return { totalSizeBytes: 0, cachedFileCount: 0, maxSizeBytes: 104857600 };
    });

    await clearCache(
      cm as unknown as Parameters<typeof clearCache>[0],
      setIsClearing,
      setCacheStats,
      clearAllDownloads,
    );

    expect(order).toEqual([
      'clearing:true',
      'clearAll',
      'clearDownloads',
      'getStats',
      'setStats',
      'clearing:false',
    ]);
  });
});
