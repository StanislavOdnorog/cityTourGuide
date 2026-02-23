import { useRouter } from 'expo-router';
import { View, Text, StyleSheet, Pressable } from 'react-native';
import { useSettingsStore } from '@/store/useSettingsStore';
import type { AppLanguage } from '@/store/useSettingsStore';

const LANGUAGES: { code: AppLanguage; label: string; native: string }[] = [
  { code: 'en', label: 'English', native: 'English' },
  { code: 'ru', label: 'Russian', native: 'Русский' },
];

export default function LanguageScreen() {
  const router = useRouter();
  const language = useSettingsStore((s) => s.language);
  const setLanguage = useSettingsStore((s) => s.setLanguage);

  return (
    <View style={styles.container}>
      <View style={styles.content}>
        <Text style={styles.title}>Choose your language</Text>
        <Text style={styles.subtitle}>Stories will be played in this language</Text>

        <View style={styles.options}>
          {LANGUAGES.map((lang) => {
            const isSelected = language === lang.code;
            return (
              <Pressable
                key={lang.code}
                onPress={() => setLanguage(lang.code)}
                style={({ pressed }) => [
                  styles.option,
                  isSelected && styles.optionSelected,
                  pressed && styles.optionPressed,
                ]}
                accessibilityRole="radio"
                accessibilityState={{ selected: isSelected }}
                accessibilityLabel={lang.label}
              >
                <Text style={[styles.optionNative, isSelected && styles.optionTextSelected]}>
                  {lang.native}
                </Text>
                {lang.native !== lang.label && (
                  <Text style={[styles.optionLabel, isSelected && styles.optionLabelSelected]}>
                    {lang.label}
                  </Text>
                )}
              </Pressable>
            );
          })}
        </View>
      </View>

      <View style={styles.footer}>
        <View style={styles.dots}>
          <View style={styles.dot} />
          <View style={[styles.dot, styles.dotActive]} />
          <View style={styles.dot} />
        </View>

        <Pressable
          onPress={() => router.push('/onboarding/permissions')}
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
  options: {
    width: '100%',
    gap: 16,
  },
  option: {
    borderWidth: 2,
    borderColor: '#333333',
    borderRadius: 16,
    paddingVertical: 20,
    paddingHorizontal: 24,
    alignItems: 'center',
  },
  optionSelected: {
    borderColor: '#4ADE80',
    backgroundColor: 'rgba(74, 222, 128, 0.08)',
  },
  optionPressed: {
    opacity: 0.8,
  },
  optionNative: {
    fontSize: 22,
    fontWeight: '600',
    color: '#FFFFFF',
  },
  optionTextSelected: {
    color: '#4ADE80',
  },
  optionLabel: {
    fontSize: 14,
    color: '#666666',
    marginTop: 4,
  },
  optionLabelSelected: {
    color: '#4ADE80',
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
