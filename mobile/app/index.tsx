import { Redirect } from 'expo-router';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import { useSettingsStore } from '@/store/useSettingsStore';

export default function Index() {
  const hasHydrated = useSettingsStore((s) => s._hasHydrated);
  const onboardingCompleted = useSettingsStore((s) => s.onboardingCompleted);

  if (!hasHydrated) {
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
