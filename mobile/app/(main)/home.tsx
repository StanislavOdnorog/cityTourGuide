import { View, Text, StyleSheet } from 'react-native';
import { API_BASE_URL } from '@/constants';

export default function HomeScreen() {
  return (
    <View style={styles.container}>
      <Text style={styles.title}>City Stories Guide</Text>
      <Text style={styles.subtitle}>Start walking to hear stories</Text>
      <Text style={styles.subtitle}>{API_BASE_URL}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#666',
  },
});
