import * as Location from 'expo-location';
import * as Notifications from 'expo-notifications';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { View, Text, StyleSheet, Pressable, Alert } from 'react-native';
import { useSettingsStore } from '@/store/useSettingsStore';

export default function PermissionsScreen() {
  const router = useRouter();
  const completeOnboarding = useSettingsStore((s) => s.completeOnboarding);
  const [locationGranted, setLocationGranted] = useState(false);
  const [notificationsGranted, setNotificationsGranted] = useState(false);

  const requestLocationPermission = async () => {
    const { status: fgStatus } = await Location.requestForegroundPermissionsAsync();
    if (fgStatus !== Location.PermissionStatus.GRANTED) {
      Alert.alert(
        'Location Required',
        'City Stories needs your location to find stories about nearby places. Please enable it in Settings.',
      );
      return;
    }

    const { status: bgStatus } = await Location.requestBackgroundPermissionsAsync();
    if (bgStatus === Location.PermissionStatus.GRANTED) {
      setLocationGranted(true);
    } else {
      // Foreground-only is acceptable, background is optional
      setLocationGranted(true);
    }
  };

  const requestNotificationPermission = async () => {
    const { status } = await Notifications.requestPermissionsAsync();
    setNotificationsGranted(status === 'granted');
  };

  const handleGetStarted = () => {
    completeOnboarding();
    router.replace('/(main)/home');
  };

  const allPermissionsHandled = locationGranted;

  return (
    <View style={styles.container}>
      <View style={styles.content}>
        <Text style={styles.title}>Almost ready</Text>
        <Text style={styles.subtitle}>
          We need a couple of permissions to find stories around you
        </Text>

        <View style={styles.permissions}>
          <Pressable
            onPress={() => void requestLocationPermission()}
            style={({ pressed }) => [
              styles.permissionCard,
              locationGranted && styles.permissionGranted,
              pressed && styles.permissionPressed,
            ]}
            disabled={locationGranted}
            accessibilityRole="button"
            accessibilityLabel="Allow location access"
          >
            <Text style={styles.permissionIcon}>{locationGranted ? '\u2713' : '\u{1F4CD}'}</Text>
            <View style={styles.permissionText}>
              <Text style={styles.permissionTitle}>Location</Text>
              <Text style={styles.permissionDesc}>
                To find stories about nearby places while you walk
              </Text>
            </View>
          </Pressable>

          <Pressable
            onPress={() => void requestNotificationPermission()}
            style={({ pressed }) => [
              styles.permissionCard,
              notificationsGranted && styles.permissionGranted,
              pressed && styles.permissionPressed,
            ]}
            disabled={notificationsGranted}
            accessibilityRole="button"
            accessibilityLabel="Allow notifications"
          >
            <Text style={styles.permissionIcon}>
              {notificationsGranted ? '\u2713' : '\u{1F514}'}
            </Text>
            <View style={styles.permissionText}>
              <Text style={styles.permissionTitle}>Notifications</Text>
              <Text style={styles.permissionDesc}>To tell you when new stories appear nearby</Text>
            </View>
          </Pressable>
        </View>
      </View>

      <View style={styles.footer}>
        <View style={styles.dots}>
          <View style={styles.dot} />
          <View style={styles.dot} />
          <View style={[styles.dot, styles.dotActive]} />
        </View>

        <Pressable
          onPress={handleGetStarted}
          style={({ pressed }) => [
            styles.button,
            !allPermissionsHandled && styles.buttonDisabled,
            pressed && allPermissionsHandled && styles.buttonPressed,
          ]}
          disabled={!allPermissionsHandled}
          accessibilityRole="button"
          accessibilityLabel="Get Started"
        >
          <Text style={[styles.buttonText, !allPermissionsHandled && styles.buttonTextDisabled]}>
            Get Started
          </Text>
        </Pressable>

        {!allPermissionsHandled && (
          <Text style={styles.hint}>Please allow location access to continue</Text>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0D0D0D',
    paddingHorizontal: 32,
    paddingTop: 80,
    paddingBottom: 48,
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: '#FFFFFF',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 16,
    fontWeight: '400',
    color: '#888888',
    textAlign: 'center',
    marginTop: 8,
    marginBottom: 40,
  },
  permissions: {
    width: '100%',
    gap: 16,
  },
  permissionCard: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 2,
    borderColor: '#333333',
    borderRadius: 16,
    padding: 20,
    gap: 16,
  },
  permissionGranted: {
    borderColor: '#4ADE80',
    backgroundColor: 'rgba(74, 222, 128, 0.08)',
  },
  permissionPressed: {
    opacity: 0.8,
  },
  permissionIcon: {
    fontSize: 28,
  },
  permissionText: {
    flex: 1,
  },
  permissionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#FFFFFF',
  },
  permissionDesc: {
    fontSize: 14,
    color: '#888888',
    marginTop: 4,
  },
  footer: {
    alignItems: 'center',
    gap: 32,
  },
  dots: {
    flexDirection: 'row',
    gap: 8,
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#333333',
  },
  dotActive: {
    backgroundColor: '#4ADE80',
    width: 24,
  },
  button: {
    backgroundColor: '#4ADE80',
    paddingHorizontal: 48,
    paddingVertical: 16,
    borderRadius: 28,
    width: '100%',
    alignItems: 'center',
    minHeight: 48,
  },
  buttonDisabled: {
    backgroundColor: '#333333',
  },
  buttonPressed: {
    opacity: 0.8,
  },
  buttonText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#0D0D0D',
  },
  buttonTextDisabled: {
    color: '#666666',
  },
  hint: {
    fontSize: 13,
    color: '#666666',
    textAlign: 'center',
  },
});
