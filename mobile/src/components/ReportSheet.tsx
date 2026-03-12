import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Modal,
  Pressable,
  TextInput,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { reportStory } from '@/api';
import { getAuthenticatedUserId } from '@/store/authBootstrap';
import { useAuthStore } from '@/store/useAuthStore';
import { useWalkStore } from '@/store/useWalkStore';
import type { ReportType } from '@/types';

interface ReportSheetProps {
  visible: boolean;
  storyId: number;
  onClose: () => void;
}

const REPORT_TYPES: { type: ReportType; label: string; icon: string }[] = [
  { type: 'wrong_location', label: 'Wrong Location', icon: '\uD83D\uDCCD' },
  { type: 'wrong_fact', label: 'Wrong Fact', icon: '\u2757' },
  { type: 'inappropriate_content', label: 'Inappropriate', icon: '\u26A0\uFE0F' },
];

export function ReportSheet({ visible, storyId, onClose }: ReportSheetProps) {
  const [selectedType, setSelectedType] = useState<ReportType | null>(null);
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const hasHydrated = useAuthStore((s) => s._hasHydrated);
  const bootstrapStatus = useAuthStore((s) => s.bootstrapStatus);
  const authReady = hasHydrated && bootstrapStatus !== 'loading' && bootstrapStatus !== 'idle';
  const currentLocation = useWalkStore((s) => s.currentLocation);

  const handleSubmit = async () => {
    if (!selectedType) return;

    setSubmitting(true);
    try {
      const userId = await getAuthenticatedUserId();
      if (!userId) {
        Alert.alert('Unavailable', 'Please wait for account setup to finish and try again.');
        return;
      }

      await reportStory({
        story_id: storyId,
        user_id: userId,
        type: selectedType,
        comment: comment.trim() || undefined,
        lat: currentLocation?.lat,
        lng: currentLocation?.lng,
      });
      handleClose();
      Alert.alert('Report Sent', 'Thank you for your feedback.');
    } catch {
      Alert.alert('Error', 'Failed to send report. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleClose = () => {
    setSelectedType(null);
    setComment('');
    onClose();
  };

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={handleClose}>
      <Pressable style={styles.overlay} onPress={handleClose}>
        <Pressable style={styles.sheet} onPress={() => {}}>
          <View style={styles.handle} />
          <Text style={styles.title}>Report Story</Text>
          <Text style={styles.subtitle}>What's wrong with this story?</Text>

          <View style={styles.typeButtons}>
            {REPORT_TYPES.map(({ type, label, icon }) => (
              <Pressable
                key={type}
                onPress={() => setSelectedType(type)}
                style={[styles.typeButton, selectedType === type && styles.typeButtonSelected]}
                accessibilityRole="button"
                accessibilityLabel={label}
              >
                <Text style={styles.typeIcon}>{icon}</Text>
                <Text style={[styles.typeLabel, selectedType === type && styles.typeLabelSelected]}>
                  {label}
                </Text>
              </Pressable>
            ))}
          </View>

          <TextInput
            style={styles.commentInput}
            placeholder="Add a comment (optional)"
            placeholderTextColor="#666666"
            value={comment}
            onChangeText={setComment}
            multiline
            maxLength={500}
          />

          <Pressable
            onPress={() => void handleSubmit()}
            disabled={!selectedType || submitting || !authReady}
            style={[
              styles.submitButton,
              (!selectedType || submitting || !authReady) && styles.submitDisabled,
            ]}
            accessibilityRole="button"
            accessibilityLabel="Submit report"
          >
            {submitting ? (
              <ActivityIndicator color="#0D0D0D" />
            ) : (
              <Text style={styles.submitText}>Submit Report</Text>
            )}
          </Pressable>

          <Pressable onPress={handleClose} style={styles.cancelButton} accessibilityRole="button">
            <Text style={styles.cancelText}>Cancel</Text>
          </Pressable>
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
  typeButtons: {
    flexDirection: 'row',
    gap: 10,
  },
  typeButton: {
    flex: 1,
    backgroundColor: '#2A2A2A',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    borderWidth: 2,
    borderColor: 'transparent',
  },
  typeButtonSelected: {
    borderColor: '#4ADE80',
    backgroundColor: '#1A2E1A',
  },
  typeIcon: {
    fontSize: 24,
    marginBottom: 6,
  },
  typeLabel: {
    fontSize: 12,
    color: '#CCCCCC',
    textAlign: 'center',
    fontWeight: '500',
  },
  typeLabelSelected: {
    color: '#4ADE80',
  },
  commentInput: {
    backgroundColor: '#2A2A2A',
    borderRadius: 12,
    padding: 14,
    marginTop: 16,
    color: '#FFFFFF',
    fontSize: 14,
    minHeight: 80,
    textAlignVertical: 'top',
  },
  submitButton: {
    backgroundColor: '#4ADE80',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 16,
    minHeight: 48,
    justifyContent: 'center',
  },
  submitDisabled: {
    opacity: 0.4,
  },
  submitText: {
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
});
