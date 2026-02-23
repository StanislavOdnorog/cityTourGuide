import { useRouter } from 'expo-router';
import { useRef, useEffect } from 'react';
import { View, Text, StyleSheet, Pressable, Animated, Easing } from 'react-native';

export default function WelcomeScreen() {
  const router = useRouter();
  const fadeAnim = useRef(new Animated.Value(0)).current;
  const slideAnim = useRef(new Animated.Value(30)).current;

  useEffect(() => {
    Animated.parallel([
      Animated.timing(fadeAnim, {
        toValue: 1,
        duration: 800,
        easing: Easing.out(Easing.ease),
        useNativeDriver: true,
      }),
      Animated.timing(slideAnim, {
        toValue: 0,
        duration: 800,
        easing: Easing.out(Easing.ease),
        useNativeDriver: true,
      }),
    ]).start();
  }, [fadeAnim, slideAnim]);

  return (
    <View style={styles.container}>
      <View style={styles.content}>
        <Animated.View
          style={[
            styles.textContainer,
            { opacity: fadeAnim, transform: [{ translateY: slideAnim }] },
          ]}
        >
          <Text style={styles.title}>The city has stories</Text>
          <Text style={styles.subtitle}>You just need headphones.</Text>
        </Animated.View>
      </View>

      <View style={styles.footer}>
        <View style={styles.dots}>
          <View style={[styles.dot, styles.dotActive]} />
          <View style={styles.dot} />
          <View style={styles.dot} />
        </View>

        <Pressable
          onPress={() => router.push('/onboarding/language')}
          style={({ pressed }) => [styles.button, pressed && styles.buttonPressed]}
          accessibilityRole="button"
          accessibilityLabel="Next"
        >
          <Text style={styles.buttonText}>Next</Text>
        </Pressable>
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
  textContainer: {
    alignItems: 'center',
  },
  title: {
    fontSize: 36,
    fontWeight: '700',
    color: '#FFFFFF',
    textAlign: 'center',
    lineHeight: 44,
  },
  subtitle: {
    fontSize: 20,
    fontWeight: '400',
    color: '#888888',
    textAlign: 'center',
    marginTop: 16,
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
  buttonPressed: {
    opacity: 0.8,
  },
  buttonText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#0D0D0D',
  },
});
