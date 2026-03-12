import { Stack, router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useEffect, useRef } from 'react';
import { checkAndMigrateCacheSchema } from '@/services/cache';
import { reconcileDownloadState } from '@/services/download';
import { notificationManager } from '@/services/notifications';
import { initSyncQueue, teardownSyncQueue } from '@/services/sync';
import { bootstrapAnonymousAuth } from '@/store/authBootstrap';
import { subscribeAuthBoundaryReset } from '@/store/authBoundaryReset';
import { useAuthStore } from '@/store/useAuthStore';
import { useDownloadStore } from '@/store/useDownloadStore';
import { useSettingsStore } from '@/store/useSettingsStore';

export default function RootLayout() {
  const settingsHydrated = useSettingsStore((s) => s._hasHydrated);
  const authHydrated = useAuthStore((s) => s._hasHydrated);
  const authBootstrapStatus = useAuthStore((s) => s.bootstrapStatus);
  const downloadHydrated = useDownloadStore((s) => s._hasHydrated);
  const reconciledRef = useRef(false);

  useEffect(() => {
    return subscribeAuthBoundaryReset();
  }, []);

  useEffect(() => {
    void initSyncQueue();

    return () => {
      void teardownSyncQueue();
    };
  }, []);

  useEffect(() => {
    // Listen for notification taps — navigate to home screen
    notificationManager.addNotificationResponseListener((response) => {
      const data = response.notification.request.content.data as Record<string, string> | undefined;
      if (data?.type === 'geo') {
        router.push('/(main)/home');
      }
    });

    return () => {
      notificationManager.dispose();
    };
  }, []);

  useEffect(() => {
    if (!settingsHydrated || !authHydrated || authBootstrapStatus !== 'idle') {
      return;
    }

    void bootstrapAnonymousAuth();
  }, [settingsHydrated, authHydrated, authBootstrapStatus]);

  // Reconcile push notification registration after auth is ready
  useEffect(() => {
    if (authBootstrapStatus !== 'ready') {
      return;
    }

    void notificationManager.reconcile();
  }, [authBootstrapStatus]);

  // Check cache schema version, then reconcile persisted download state
  useEffect(() => {
    if (!downloadHydrated || reconciledRef.current) return;
    reconciledRef.current = true;
    void (async () => {
      await checkAndMigrateCacheSchema();
      await reconcileDownloadState();
    })();
  }, [downloadHydrated]);

  return (
    <>
      <StatusBar style="auto" />
      <Stack screenOptions={{ headerShown: false }} />
    </>
  );
}
