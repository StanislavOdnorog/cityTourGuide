import { router } from 'expo-router';
import { useCallback, useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Pressable,
  ScrollView,
  Switch,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { deleteAccount } from '@/api/endpoints';
import { isAppApiError, userMessageForError } from '@/api/errors';
import { handleNotificationToggle } from '@/hooks/settingsActions';
import { useCacheManager } from '@/hooks/useCacheManager';
import { StoryCacheManager } from '@/services/cache';
import { notificationManager } from '@/services/notifications';
import { useAuthStore } from '@/store/useAuthStore';
import { usePurchaseStore } from '@/store/usePurchaseStore';
import { useSettingsStore, type AppLanguage } from '@/store/useSettingsStore';

const cacheManager = new StoryCacheManager();

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function SectionHeader({ title }: { title: string }) {
  return <Text style={styles.sectionTitle}>{title}</Text>;
}

function SettingRow({
  label,
  value,
  onPress,
}: {
  label: string;
  value?: string;
  onPress?: () => void;
}) {
  return (
    <Pressable
      onPress={onPress}
      disabled={!onPress}
      style={({ pressed }) => [styles.settingRow, pressed && onPress && styles.settingRowPressed]}
      accessibilityRole={onPress ? 'button' : 'text'}
    >
      <Text style={styles.settingLabel}>{label}</Text>
      {value != null && <Text style={styles.settingValue}>{value}</Text>}
    </Pressable>
  );
}

function ToggleRow({
  label,
  description,
  value,
  onValueChange,
}: {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (val: boolean) => void;
}) {
  return (
    <View style={styles.settingRow}>
      <View style={styles.toggleTextContainer}>
        <Text style={styles.settingLabel}>{label}</Text>
        {description && <Text style={styles.settingDescription}>{description}</Text>}
      </View>
      <Switch
        value={value}
        onValueChange={onValueChange}
        trackColor={{ false: '#333333', true: '#4ADE80' }}
        thumbColor="#FFFFFF"
      />
    </View>
  );
}

export default function SettingsScreen() {
  const language = useSettingsStore((s) => s.language);
  const setLanguage = useSettingsStore((s) => s.setLanguage);
  const geoNotifications = useSettingsStore((s) => s.geoNotifications);
  const setGeoNotifications = useSettingsStore((s) => s.setGeoNotifications);
  const contentNotifications = useSettingsStore((s) => s.contentNotifications);
  const setContentNotifications = useSettingsStore((s) => s.setContentNotifications);
  const pushToken = useSettingsStore((s) => s.pushToken);
  const setPushToken = useSettingsStore((s) => s.setPushToken);

  const {
    stats: cacheStats,
    isClearing,
    initialized: cacheInitialized,
    clearCache,
    removeCityCache,
    downloadedCityCount,
    downloadedCityList,
  } = useCacheManager(cacheManager);

  const purchaseStatus = usePurchaseStore((s) => s.status);

  const handleGeoToggle = useCallback(
    (enabled: boolean) =>
      handleNotificationToggle(
        enabled,
        pushToken,
        setPushToken,
        setGeoNotifications,
        notificationManager,
      ),
    [pushToken, setPushToken, setGeoNotifications],
  );

  const handleContentToggle = useCallback(
    (enabled: boolean) =>
      handleNotificationToggle(
        enabled,
        pushToken,
        setPushToken,
        setContentNotifications,
        notificationManager,
      ),
    [pushToken, setPushToken, setContentNotifications],
  );

  const handleLanguageToggle = useCallback(() => {
    const next: AppLanguage = language === 'en' ? 'ru' : 'en';
    setLanguage(next);
  }, [language, setLanguage]);

  const handleClearCache = useCallback(() => {
    Alert.alert('Clear Cache', 'This will delete all cached audio files. Continue?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Clear',
        style: 'destructive',
        onPress: () => void clearCache(),
      },
    ]);
  }, [clearCache]);

  const handleRemoveCity = useCallback(
    (cityId: number, cityName: string) => {
      Alert.alert('Remove City', `Remove all cached audio for ${cityName}?`, [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Remove',
          style: 'destructive',
          onPress: () => void removeCityCache(cityId),
        },
      ]);
    },
    [removeCityCache],
  );

  const [isDeleting, setIsDeleting] = useState(false);

  const handleDeleteAccount = useCallback(() => {
    Alert.alert(
      'Delete Account',
      'Your account will be scheduled for deletion. You have 30 days to restore it. After that, all your data will be permanently removed.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete Account',
          style: 'destructive',
          onPress: async () => {
            setIsDeleting(true);
            try {
              await deleteAccount();
              Alert.alert(
                'Account Scheduled for Deletion',
                'Your account will be permanently deleted in 30 days. You can restore it from the settings before then.',
              );
            } catch (err) {
              if (isAppApiError(err) && err.category === 'unauthorized') {
                useAuthStore.getState().clearSession();
                Alert.alert('Session Expired', 'Please sign in again.');
              } else {
                Alert.alert('Error', userMessageForError(err));
              }
            } finally {
              setIsDeleting(false);
            }
          },
        },
      ],
    );
  }, []);

  const purchaseLabel = getPurchaseLabel(purchaseStatus);

  return (
    <View style={styles.container}>
      <View style={styles.headerBar}>
        <Pressable
          onPress={() => router.back()}
          style={styles.backButton}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          <Text style={styles.backText}>{'< Back'}</Text>
        </Pressable>
        <Text style={styles.headerTitle}>Settings</Text>
        <View style={styles.backButton} />
      </View>

      <ScrollView style={styles.scrollView} contentContainerStyle={styles.scrollContent}>
        {/* Language */}
        <SectionHeader title="Language" />
        <View style={styles.section}>
          <Pressable
            onPress={handleLanguageToggle}
            style={styles.languageToggle}
            accessibilityRole="button"
            accessibilityLabel={`Switch language to ${language === 'en' ? 'Russian' : 'English'}`}
          >
            <View style={[styles.languageOption, language === 'en' && styles.languageOptionActive]}>
              <Text
                style={[
                  styles.languageOptionText,
                  language === 'en' && styles.languageOptionTextActive,
                ]}
              >
                English
              </Text>
            </View>
            <View style={[styles.languageOption, language === 'ru' && styles.languageOptionActive]}>
              <Text
                style={[
                  styles.languageOptionText,
                  language === 'ru' && styles.languageOptionTextActive,
                ]}
              >
                Русский
              </Text>
            </View>
          </Pressable>
        </View>

        {/* Notifications */}
        <SectionHeader title="Notifications" />
        <View style={styles.section}>
          <ToggleRow
            label="Nearby stories"
            description="Alert when you're near interesting places (max 2/day)"
            value={geoNotifications}
            onValueChange={handleGeoToggle}
          />
          <View style={styles.divider} />
          <ToggleRow
            label="New content"
            description="Notify about new stories in your city (max 1/week)"
            value={contentNotifications}
            onValueChange={handleContentToggle}
          />
        </View>

        {/* Cache */}
        <SectionHeader title="Cache" />
        <View style={styles.section}>
          <SettingRow
            label="Cached audio"
            value={
              cacheInitialized
                ? `${formatBytes(cacheStats.totalSizeBytes)} / ${formatBytes(cacheStats.maxSizeBytes)}`
                : '...'
            }
          />
          <View style={styles.divider} />
          <SettingRow
            label="Cached files"
            value={cacheInitialized ? String(cacheStats.cachedFileCount) : '...'}
          />
          <View style={styles.divider} />
          <SettingRow
            label="Downloaded cities"
            value={cacheInitialized ? String(downloadedCityCount) : '...'}
          />
          {downloadedCityList.map((city) => (
            <View key={city.cityId}>
              <View style={styles.divider} />
              <Pressable
                onPress={() => handleRemoveCity(city.cityId, `City #${city.cityId}`)}
                disabled={isClearing}
                style={({ pressed }) => [
                  styles.settingRow,
                  pressed && styles.settingRowPressed,
                  isClearing && styles.settingRowDisabled,
                ]}
                accessibilityRole="button"
                accessibilityLabel={`Remove city ${city.cityId}`}
              >
                <Text style={styles.settingLabel}>City #{city.cityId}</Text>
                <Text style={styles.settingValue}>
                  {formatBytes(city.totalSizeBytes)} · {city.totalFiles} files
                </Text>
              </Pressable>
            </View>
          ))}
          <View style={styles.divider} />
          <Pressable
            onPress={handleClearCache}
            disabled={isClearing || !cacheInitialized}
            style={({ pressed }) => [
              styles.settingRow,
              pressed && styles.settingRowPressed,
              (isClearing || !cacheInitialized) && styles.settingRowDisabled,
            ]}
            accessibilityRole="button"
            accessibilityLabel="Clear cache"
          >
            <Text style={[styles.settingLabel, styles.destructiveText]}>Clear Cache</Text>
            {isClearing && <ActivityIndicator size="small" color="#EF4444" />}
          </Pressable>
        </View>

        {/* Subscription */}
        <SectionHeader title="Subscription" />
        <View style={styles.section}>
          <SettingRow label="Status" value={purchaseLabel} />
          {purchaseStatus?.free_stories_limit != null && !purchaseStatus.has_full_access && (
            <>
              <View style={styles.divider} />
              <SettingRow
                label="Free stories today"
                value={`${purchaseStatus.free_stories_left} / ${purchaseStatus.free_stories_limit}`}
              />
            </>
          )}
        </View>

        {/* Account */}
        <SectionHeader title="Account" />
        <View style={styles.section}>
          <SettingRow label="Account type" value="Anonymous" />
          <View style={styles.divider} />
          <Pressable
            onPress={handleDeleteAccount}
            disabled={isDeleting}
            style={({ pressed }) => [
              styles.settingRow,
              pressed && styles.settingRowPressed,
              isDeleting && styles.settingRowDisabled,
            ]}
            accessibilityRole="button"
            accessibilityLabel="Delete account"
          >
            <Text style={[styles.settingLabel, styles.destructiveText]}>Delete Account</Text>
            {isDeleting && <ActivityIndicator size="small" color="#EF4444" />}
          </Pressable>
        </View>

        <View style={styles.bottomPadding} />
      </ScrollView>
    </View>
  );
}

function getPurchaseLabel(status: ReturnType<typeof usePurchaseStore.getState>['status']): string {
  if (!status) return 'Free';
  if (status.is_lifetime) return 'Lifetime Access';
  if (status.active_subscription) return 'Monthly Pass';
  if ((status.city_packs ?? []).length > 0) return `${status.city_packs?.length ?? 0} City Pack(s)`;
  return 'Free';
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0D0D0D',
  },
  headerBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingTop: 56,
    paddingBottom: 12,
  },
  backButton: {
    width: 60,
    minHeight: 48,
    justifyContent: 'center',
  },
  backText: {
    fontSize: 16,
    color: '#4ADE80',
    fontWeight: '500',
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#FFFFFF',
    letterSpacing: 0.5,
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    paddingHorizontal: 16,
  },
  sectionTitle: {
    fontSize: 13,
    fontWeight: '600',
    color: '#888888',
    textTransform: 'uppercase',
    letterSpacing: 1,
    marginTop: 24,
    marginBottom: 8,
    paddingHorizontal: 4,
  },
  section: {
    backgroundColor: '#1A1A1A',
    borderRadius: 12,
    overflow: 'hidden',
  },
  settingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 14,
    minHeight: 48,
  },
  settingRowPressed: {
    backgroundColor: '#2A2A2A',
  },
  settingRowDisabled: {
    opacity: 0.4,
  },
  settingLabel: {
    fontSize: 16,
    color: '#FFFFFF',
  },
  settingValue: {
    fontSize: 16,
    color: '#888888',
  },
  settingDescription: {
    fontSize: 13,
    color: '#666666',
    marginTop: 2,
  },
  toggleTextContainer: {
    flex: 1,
    marginRight: 12,
  },
  divider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: '#333333',
    marginLeft: 16,
  },
  destructiveText: {
    color: '#EF4444',
  },
  languageToggle: {
    flexDirection: 'row',
    margin: 4,
  },
  languageOption: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
    borderRadius: 10,
  },
  languageOptionActive: {
    backgroundColor: '#4ADE80',
  },
  languageOptionText: {
    fontSize: 16,
    fontWeight: '500',
    color: '#888888',
  },
  languageOptionTextActive: {
    color: '#0D0D0D',
    fontWeight: '600',
  },
  bottomPadding: {
    height: 40,
  },
});
