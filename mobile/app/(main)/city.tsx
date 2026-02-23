import { useLocalSearchParams, router } from 'expo-router';
import React, { useCallback, useRef } from 'react';
import { View, Text, StyleSheet, Pressable, ActivityIndicator } from 'react-native';
import MapView, { Marker, Callout, type Region } from 'react-native-maps';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useCityScreen, type CityMarker } from '@/hooks/useCityScreen';

const MARKER_COLORS: Record<string, string> = {
  green: '#4ADE80',
  blue: '#60A5FA',
  grey: '#6B7280',
};

const LATITUDE_DELTA = 0.04;
const LONGITUDE_DELTA = 0.04;

function POIMarker({ marker }: { marker: CityMarker }) {
  return (
    <Marker
      coordinate={{ latitude: marker.poi.lat, longitude: marker.poi.lng }}
      pinColor={MARKER_COLORS[marker.color]}
      tracksViewChanges={false}
    >
      <Callout>
        <View style={styles.callout}>
          <Text style={styles.calloutTitle}>{marker.poi.name}</Text>
          <Text style={styles.calloutSubtitle}>
            {marker.poi.story_count} {marker.poi.story_count === 1 ? 'story' : 'stories'}
          </Text>
        </View>
      </Callout>
    </Marker>
  );
}

export default function CityScreen() {
  const params = useLocalSearchParams<{ cityId: string }>();
  const cityId = Number(params.cityId) || 1;
  const insets = useSafeAreaInsets();
  const mapRef = useRef<MapView>(null);

  const {
    cityName,
    markers,
    totalStories,
    listenedCount,
    centerLat,
    centerLng,
    isLoading,
    error,
    refresh,
  } = useCityScreen(cityId);

  const initialRegion: Region = {
    latitude: centerLat || 41.7151,
    longitude: centerLng || 44.8271,
    latitudeDelta: LATITUDE_DELTA,
    longitudeDelta: LONGITUDE_DELTA,
  };

  const handleBack = useCallback(() => {
    router.back();
  }, []);

  if (isLoading && markers.length === 0) {
    return (
      <View style={[styles.container, styles.centerContent]}>
        <ActivityIndicator size="large" color="#4ADE80" />
        <Text style={styles.loadingText}>Loading city...</Text>
      </View>
    );
  }

  if (error && markers.length === 0) {
    return (
      <View style={[styles.container, styles.centerContent]}>
        <Text style={styles.errorText}>{error}</Text>
        <Pressable
          onPress={() => void refresh()}
          style={styles.retryButton}
          accessibilityRole="button"
          accessibilityLabel="Retry"
        >
          <Text style={styles.retryButtonText}>Retry</Text>
        </Pressable>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <MapView
        ref={mapRef}
        style={styles.map}
        initialRegion={initialRegion}
        showsUserLocation
        showsMyLocationButton={false}
      >
        {markers.map((marker) => (
          <POIMarker key={marker.poi.id} marker={marker} />
        ))}
      </MapView>

      <View style={[styles.header, { paddingTop: Math.max(insets.top, 16) }]}>
        <Pressable
          onPress={handleBack}
          style={styles.backButton}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
        >
          <Text style={styles.backIcon}>{'\u2190'}</Text>
        </Pressable>
        <View style={styles.headerCenter}>
          <Text style={styles.cityTitle} numberOfLines={1}>
            {cityName ?? 'City'}
          </Text>
          <Text style={styles.statsText}>
            {totalStories} {totalStories === 1 ? 'story' : 'stories'} / {listenedCount} listened
          </Text>
        </View>
        <View style={styles.headerSpacer} />
      </View>

      <View style={[styles.footer, { paddingBottom: Math.max(insets.bottom, 16) }]}>
        <Pressable
          style={({ pressed }) => [
            styles.actionButton,
            styles.downloadButton,
            pressed && styles.buttonPressed,
          ]}
          accessibilityRole="button"
          accessibilityLabel="Download for Offline"
        >
          <Text style={styles.actionButtonText}>Download for Offline</Text>
        </Pressable>

        <Pressable
          style={({ pressed }) => [
            styles.actionButton,
            styles.buyButton,
            pressed && styles.buttonPressed,
          ]}
          accessibilityRole="button"
          accessibilityLabel="Buy City"
        >
          <Text style={styles.buyButtonText}>Buy City</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0D0D0D',
  },
  centerContent: {
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 24,
  },
  map: {
    flex: 1,
  },
  header: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingBottom: 12,
    backgroundColor: 'rgba(13, 13, 13, 0.85)',
  },
  backButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255, 255, 255, 0.15)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  backIcon: {
    fontSize: 20,
    color: '#FFFFFF',
  },
  headerCenter: {
    flex: 1,
    alignItems: 'center',
    marginHorizontal: 8,
  },
  headerSpacer: {
    width: 40,
  },
  cityTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#FFFFFF',
  },
  statsText: {
    fontSize: 13,
    color: '#AAAAAA',
    marginTop: 2,
  },
  footer: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    paddingHorizontal: 16,
    paddingTop: 12,
    gap: 12,
    backgroundColor: 'rgba(13, 13, 13, 0.85)',
  },
  actionButton: {
    flex: 1,
    height: 48,
    borderRadius: 24,
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: 48,
  },
  downloadButton: {
    backgroundColor: '#4ADE80',
  },
  buyButton: {
    backgroundColor: 'transparent',
    borderWidth: 1.5,
    borderColor: '#4ADE80',
  },
  buttonPressed: {
    opacity: 0.7,
  },
  actionButtonText: {
    fontSize: 15,
    fontWeight: '600',
    color: '#0D0D0D',
  },
  buyButtonText: {
    fontSize: 15,
    fontWeight: '600',
    color: '#4ADE80',
  },
  loadingText: {
    fontSize: 16,
    color: '#888888',
    marginTop: 16,
  },
  errorText: {
    fontSize: 16,
    color: '#EF4444',
    textAlign: 'center',
    marginBottom: 16,
  },
  retryButton: {
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 20,
    backgroundColor: '#4ADE80',
    minHeight: 48,
    justifyContent: 'center',
  },
  retryButtonText: {
    fontSize: 15,
    fontWeight: '600',
    color: '#0D0D0D',
  },
  callout: {
    padding: 4,
    minWidth: 120,
  },
  calloutTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#000000',
  },
  calloutSubtitle: {
    fontSize: 12,
    color: '#666666',
    marginTop: 2,
  },
});
