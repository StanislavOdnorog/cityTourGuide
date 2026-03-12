import { Redirect } from 'expo-router';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import { useAuthStore } from '@/store/useAuthStore';
import { useSettingsStore } from '@/store/useSettingsStore';

export default function Index() {
  const settingsHydrated = useSettingsStore((s) => s._hasHydrated);
  const onboardingCompleted = useSettingsStore((s) => s.onboardingCompleted);
  const authHydrated = useAuthStore((s) => s._hasHydrated);
  const authBootstrapStatus = useAuthStore((s) => s.bootstrapStatus);

  if (
    !settingsHydrated ||
    !authHydrated ||
    authBootstrapStatus === 'idle' ||
    authBootstrapStatus === 'loading'
  ) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator size="large" color="#4ADE80" />
      </View>
    );
  }

  if (onboardingCompleted) {
    return <Redirect href="/(main)/home" />;
  }

  return <Redirect href="/onboarding" />;
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#0D0D0D',
  },
});
