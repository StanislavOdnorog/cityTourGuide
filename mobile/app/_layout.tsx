import { Stack, router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';
import { notificationManager } from '@/services/notifications';

export default function RootLayout() {
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

  return (
    <>
      <StatusBar style="auto" />
      <Stack screenOptions={{ headerShown: false }} />
    </>
  );
}
