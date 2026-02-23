import React from 'react';
import { View, Text, StyleSheet, Modal, Pressable, ActivityIndicator } from 'react-native';
import type { DownloadStatus } from '@/store/useDownloadStore';

interface DownloadCitySheetProps {
  visible: boolean;
  cityName: string;
  status: DownloadStatus;
  totalSizeMB: number;
  totalStories: number;
  completedFiles: number;
  totalFiles: number;
  completedMB: number;
  onStart: () => void;
  onCancel: () => void;
  onClose: () => void;
  error: string | null;
}

function formatMB(bytes: number): string {
  const mb = bytes / (1024 * 1024);
  return mb < 1 ? `${(mb * 1024).toFixed(0)} KB` : `${mb.toFixed(1)} MB`;
}

export function DownloadCitySheet({
  visible,
  cityName,
  status,
  totalSizeMB,
  totalStories,
  completedFiles,
  totalFiles,
  completedMB,
  onStart,
  onCancel,
  onClose,
  error,
}: DownloadCitySheetProps) {
  const progressFraction = totalFiles > 0 ? completedFiles / totalFiles : 0;

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={styles.overlay} onPress={onClose}>
        <Pressable style={styles.sheet} onPress={() => {}}>
          <View style={styles.handle} />
          <Text style={styles.title}>Download {cityName}</Text>

          {status === 'idle' || status === 'fetching_manifest' ? (
            <>
              <Text style={styles.subtitle}>Download all stories for offline listening</Text>
              <View style={styles.infoRow}>
                <View style={styles.infoItem}>
                  <Text style={styles.infoValue}>{totalStories}</Text>
                  <Text style={styles.infoLabel}>Stories</Text>
                </View>
                <View style={styles.infoDivider} />
                <View style={styles.infoItem}>
                  <Text style={styles.infoValue}>{totalSizeMB.toFixed(1)} MB</Text>
                  <Text style={styles.infoLabel}>Total Size</Text>
                </View>
              </View>
              <Pressable
                onPress={onStart}
                disabled={status === 'fetching_manifest'}
                style={[
                  styles.primaryButton,
                  status === 'fetching_manifest' && styles.buttonDisabled,
                ]}
                accessibilityRole="button"
                accessibilityLabel="Start download"
              >
                {status === 'fetching_manifest' ? (
                  <ActivityIndicator color="#0D0D0D" />
                ) : (
                  <Text style={styles.primaryButtonText}>Download</Text>
                )}
              </Pressable>
            </>
          ) : status === 'downloading' ? (
            <>
              <Text style={styles.subtitle}>
                Downloading {completedFiles} of {totalFiles} stories...
              </Text>
              <View style={styles.progressContainer}>
                <View style={styles.progressBar}>
                  <View style={[styles.progressFill, { width: `${progressFraction * 100}%` }]} />
                </View>
                <Text style={styles.progressText}>
                  {formatMB(completedMB * 1024 * 1024)} / {formatMB(totalSizeMB * 1024 * 1024)}
                </Text>
              </View>
              <Pressable
                onPress={onCancel}
                style={styles.cancelButton}
                accessibilityRole="button"
                accessibilityLabel="Cancel download"
              >
                <Text style={styles.cancelText}>Cancel</Text>
              </Pressable>
            </>
          ) : status === 'completed' ? (
            <>
              <Text style={styles.successText}>Download complete!</Text>
              <Text style={styles.subtitle}>{totalFiles} stories are now available offline.</Text>
              <Pressable
                onPress={onClose}
                style={styles.primaryButton}
                accessibilityRole="button"
                accessibilityLabel="Done"
              >
                <Text style={styles.primaryButtonText}>Done</Text>
              </Pressable>
            </>
          ) : status === 'error' ? (
            <>
              <Text style={styles.errorText}>{error ?? 'Download failed'}</Text>
              <Pressable
                onPress={onStart}
                style={styles.primaryButton}
                accessibilityRole="button"
                accessibilityLabel="Retry download"
              >
                <Text style={styles.primaryButtonText}>Retry</Text>
              </Pressable>
              <Pressable
                onPress={onClose}
                style={styles.cancelButton}
                accessibilityRole="button"
                accessibilityLabel="Close"
              >
                <Text style={styles.cancelText}>Close</Text>
              </Pressable>
            </>
          ) : null}
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.6)',
    justifyContent: 'flex-end',
  },
  sheet: {
    backgroundColor: '#1A1A1A',
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    paddingHorizontal: 24,
    paddingBottom: 40,
    paddingTop: 12,
  },
  handle: {
    width: 40,
    height: 4,
    borderRadius: 2,
    backgroundColor: '#444444',
    alignSelf: 'center',
    marginBottom: 16,
  },
  title: {
    fontSize: 20,
    fontWeight: '700',
    color: '#FFFFFF',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 14,
    color: '#888888',
    textAlign: 'center',
    marginTop: 4,
    marginBottom: 20,
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 24,
  },
  infoItem: {
    alignItems: 'center',
    paddingHorizontal: 24,
  },
  infoValue: {
    fontSize: 22,
    fontWeight: '700',
    color: '#4ADE80',
  },
  infoLabel: {
    fontSize: 12,
    color: '#888888',
    marginTop: 2,
  },
  infoDivider: {
    width: 1,
    height: 32,
    backgroundColor: '#333333',
  },
  progressContainer: {
    marginBottom: 24,
  },
  progressBar: {
    height: 8,
    borderRadius: 4,
    backgroundColor: '#2A2A2A',
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    borderRadius: 4,
    backgroundColor: '#4ADE80',
  },
  progressText: {
    fontSize: 13,
    color: '#888888',
    textAlign: 'center',
    marginTop: 8,
  },
  primaryButton: {
    backgroundColor: '#4ADE80',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    minHeight: 48,
    justifyContent: 'center',
  },
  buttonDisabled: {
    opacity: 0.4,
  },
  primaryButtonText: {
    fontSize: 16,
    fontWeight: '700',
    color: '#0D0D0D',
  },
  cancelButton: {
    paddingVertical: 14,
    alignItems: 'center',
    marginTop: 4,
    minHeight: 48,
    justifyContent: 'center',
  },
  cancelText: {
    fontSize: 14,
    color: '#888888',
  },
  successText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#4ADE80',
    textAlign: 'center',
    marginTop: 8,
    marginBottom: 4,
  },
  errorText: {
    fontSize: 14,
    color: '#EF4444',
    textAlign: 'center',
    marginTop: 8,
    marginBottom: 20,
  },
});
