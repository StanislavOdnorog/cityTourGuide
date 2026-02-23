import { Stack } from 'expo-router';
import { View, StyleSheet } from 'react-native';
import { MiniPlayer } from '@/components';

export default function MainLayout() {
  return (
    <View style={styles.container}>
      <Stack screenOptions={{ headerShown: false }} />
      <MiniPlayer />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
