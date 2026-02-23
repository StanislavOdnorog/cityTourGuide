import AsyncStorage from '@react-native-async-storage/async-storage';

const STORAGE_KEY = 'city-stories-freemium';
const DEFAULT_DAILY_LIMIT = 5;

interface FreemiumData {
  date: string; // YYYY-MM-DD
  count: number;
}

function getTodayDate(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * FreemiumTracker tracks daily free story consumption.
 * The counter resets at midnight (local time).
 *
 * Usage:
 *   const tracker = new FreemiumTracker();
 *   const canListen = await tracker.canListenFree();
 *   if (canListen) {
 *     // play story...
 *     await tracker.recordListen();
 *   } else {
 *     // show paywall
 *   }
 */
export class FreemiumTracker {
  private dailyLimit: number;

  constructor(dailyLimit: number = DEFAULT_DAILY_LIMIT) {
    this.dailyLimit = dailyLimit;
  }

  /**
   * Load the current freemium data from storage.
   * Resets if the stored date is not today.
   */
  private async loadData(): Promise<FreemiumData> {
    const today = getTodayDate();
    const raw = await AsyncStorage.getItem(STORAGE_KEY);

    if (raw) {
      const data: FreemiumData = JSON.parse(raw) as FreemiumData;
      if (data.date === today) {
        return data;
      }
    }

    // New day or no data — reset counter
    const freshData: FreemiumData = { date: today, count: 0 };
    await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(freshData));
    return freshData;
  }

  /**
   * Check if the user can still listen for free today.
   */
  async canListenFree(): Promise<boolean> {
    const data = await this.loadData();
    return data.count < this.dailyLimit;
  }

  /**
   * Get the number of free listens remaining today.
   */
  async getRemainingListens(): Promise<number> {
    const data = await this.loadData();
    return Math.max(0, this.dailyLimit - data.count);
  }

  /**
   * Get the number of stories listened today.
   */
  async getUsedListens(): Promise<number> {
    const data = await this.loadData();
    return data.count;
  }

  /**
   * Record that the user listened to a free story.
   * Returns the number of free listens remaining.
   */
  async recordListen(): Promise<number> {
    const data = await this.loadData();
    data.count += 1;
    await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(data));
    return Math.max(0, this.dailyLimit - data.count);
  }

  /**
   * Reset the daily counter (e.g. after a purchase).
   */
  async reset(): Promise<void> {
    await AsyncStorage.removeItem(STORAGE_KEY);
  }

  /**
   * Get the daily limit.
   */
  getDailyLimit(): number {
    return this.dailyLimit;
  }
}
