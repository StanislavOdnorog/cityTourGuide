import { useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Pressable,
  Animated,
  Easing,
} from 'react-native';
import { useHomeScreen } from '@/hooks/useHomeScreen';

const CITY_NAME = 'Tbilisi';

function PulseIndicator() {
  const scale = useRef(new Animated.Value(1)).current;
  const opacity = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    const pulse = Animated.loop(
      Animated.sequence([
        Animated.parallel([
          Animated.timing(scale, {
            toValue: 1.4,
            duration: 1000,
            easing: Easing.out(Easing.ease),
            useNativeDriver: true,
          }),
          Animated.timing(opacity, {
            toValue: 0.3,
            duration: 1000,
            easing: Easing.out(Easing.ease),
            useNativeDriver: true,
          }),
        ]),
        Animated.parallel([
          Animated.timing(scale, {
            toValue: 1,
            duration: 1000,
            easing: Easing.inOut(Easing.ease),
            useNativeDriver: true,
          }),
          Animated.timing(opacity, {
            toValue: 1,
            duration: 1000,
            easing: Easing.inOut(Easing.ease),
            useNativeDriver: true,
          }),
        ]),
      ]),
    );
    pulse.start();
    return () => pulse.stop();
  }, [scale, opacity]);

  return (
    <View style={styles.pulseContainer}>
      <Animated.View
        style={[
          styles.pulseRing,
          { transform: [{ scale }], opacity },
        ]}
      />
      <View style={styles.pulseDot} />
    </View>
  );
}

export default function HomeScreen() {
  const {
    isWalking,
    isPlaying,
    currentStoryName,
    listenedCount,
    toggleWalking,
  } = useHomeScreen();

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.cityName}>{CITY_NAME}</Text>
        <Text style={styles.storyStat}>
          {listenedCount} {listenedCount === 1 ? 'story' : 'stories'} listened
        </Text>
      </View>

      <View style={styles.center}>
        {isWalking && (
          <View style={styles.listeningSection}>
            <PulseIndicator />
            <Text style={styles.listeningText}>
              {isPlaying && currentStoryName
                ? currentStoryName
                : 'Listening...'}
            </Text>
          </View>
        )}

        <Pressable
          onPress={() => void toggleWalking()}
          style={({ pressed }) => [
            styles.mainButton,
            isWalking ? styles.mainButtonStop : styles.mainButtonStart,
            pressed && styles.mainButtonPressed,
          ]}
          accessibilityRole="button"
          accessibilityLabel={isWalking ? 'Stop Walking' : 'Start Walking'}
        >
          <Text style={styles.mainButtonText}>
            {isWalking ? 'Stop' : 'Start Walking'}
          </Text>
        </Pressable>
      </View>

      <View style={styles.footer}>
        <Text style={styles.footerText}>Put on your headphones and explore</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0D0D0D',
    paddingHorizontal: 24,
    paddingTop: 60,
    paddingBottom: 40,
  },
  header: {
    alignItems: 'center',
  },
  cityName: {
    fontSize: 18,
    fontWeight: '600',
    color: '#FFFFFF',
    letterSpacing: 1,
    textTransform: 'uppercase',
  },
  storyStat: {
    fontSize: 14,
    color: '#888888',
    marginTop: 4,
  },
  center: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  listeningSection: {
    alignItems: 'center',
    marginBottom: 48,
  },
  listeningText: {
    fontSize: 16,
    color: '#4ADE80',
    marginTop: 16,
    fontWeight: '500',
  },
  pulseContainer: {
    width: 48,
    height: 48,
    justifyContent: 'center',
    alignItems: 'center',
  },
  pulseRing: {
    position: 'absolute',
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#4ADE80',
  },
  pulseDot: {
    width: 16,
    height: 16,
    borderRadius: 8,
    backgroundColor: '#4ADE80',
  },
  mainButton: {
    width: 200,
    height: 200,
    borderRadius: 100,
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: 48,
    minWidth: 48,
  },
  mainButtonStart: {
    backgroundColor: '#4ADE80',
  },
  mainButtonStop: {
    backgroundColor: '#EF4444',
  },
  mainButtonPressed: {
    opacity: 0.8,
  },
  mainButtonText: {
    fontSize: 22,
    fontWeight: '700',
    color: '#0D0D0D',
    textAlign: 'center',
  },
  footer: {
    alignItems: 'center',
  },
  footerText: {
    fontSize: 14,
    color: '#555555',
  },
});
