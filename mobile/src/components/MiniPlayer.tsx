import React, { useState } from 'react';
import { View, Text, StyleSheet, Pressable } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import TrackPlayer, { useProgress, useIsPlaying } from 'react-native-track-player';
import { ReportSheet } from '@/components/ReportSheet';
import { usePlayerStore } from '@/store/usePlayerStore';
import { formatTime } from '@/utils/formatTime';

export function MiniPlayer() {
  const currentStory = usePlayerStore((s) => s.currentStory);
  const { playing } = useIsPlaying();
  const { position, duration } = useProgress(500);
  const insets = useSafeAreaInsets();
  const [reportVisible, setReportVisible] = useState(false);

  if (!currentStory) return null;

  const progress = duration > 0 ? Math.min(position / duration, 1) : 0;

  const handlePlayPause = async () => {
    if (playing) {
      await TrackPlayer.pause();
    } else {
      await TrackPlayer.play();
    }
  };

  return (
    <View style={[styles.container, { paddingBottom: Math.max(insets.bottom, 8) }]}>
      <View style={styles.progressTrack}>
        <View style={[styles.progressFill, { flex: progress }]} />
        <View style={{ flex: 1 - progress }} />
      </View>

      <View style={styles.content}>
        <Pressable
          onPress={() => void handlePlayPause()}
          style={styles.playPauseButton}
          accessibilityRole="button"
          accessibilityLabel={playing ? 'Pause' : 'Resume'}
          hitSlop={8}
        >
          <Text style={styles.playPauseIcon}>{playing ? '\u275A\u275A' : '\u25B6'}</Text>
        </Pressable>

        <View style={styles.info}>
          <Text style={styles.storyName} numberOfLines={1}>
            {currentStory.poi_name}
          </Text>
          <Text style={styles.time}>
            {formatTime(position)} / {formatTime(duration)}
          </Text>
        </View>

        <Pressable
          onPress={() => setReportVisible(true)}
          style={styles.reportButton}
          accessibilityRole="button"
          accessibilityLabel="Report story"
          hitSlop={8}
        >
          <Text style={styles.reportIcon}>{'\u2691'}</Text>
        </Pressable>
      </View>

      <ReportSheet
        visible={reportVisible}
        storyId={currentStory.story_id}
        onClose={() => setReportVisible(false)}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: '#1A1A1A',
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: '#333333',
  },
  progressTrack: {
    height: 3,
    flexDirection: 'row',
    backgroundColor: '#333333',
  },
  progressFill: {
    backgroundColor: '#4ADE80',
  },
  content: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 12,
    paddingTop: 10,
    paddingBottom: 2,
    minHeight: 48,
  },
  playPauseButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#4ADE80',
    justifyContent: 'center',
    alignItems: 'center',
  },
  playPauseIcon: {
    fontSize: 16,
    color: '#0D0D0D',
    fontWeight: '700',
  },
  info: {
    flex: 1,
    marginHorizontal: 12,
  },
  storyName: {
    fontSize: 14,
    fontWeight: '600',
    color: '#FFFFFF',
  },
  time: {
    fontSize: 12,
    color: '#888888',
    marginTop: 2,
  },
  reportButton: {
    width: 44,
    height: 44,
    justifyContent: 'center',
    alignItems: 'center',
  },
  reportIcon: {
    fontSize: 22,
    color: '#888888',
  },
});
