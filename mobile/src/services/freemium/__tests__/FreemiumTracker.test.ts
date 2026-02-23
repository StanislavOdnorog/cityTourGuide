import AsyncStorage from '@react-native-async-storage/async-storage';
import { FreemiumTracker } from '../FreemiumTracker';

jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

const mockGetItem = AsyncStorage.getItem as jest.Mock;
const mockSetItem = AsyncStorage.setItem as jest.Mock;

function getTodayDate(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

describe('FreemiumTracker', () => {
  let tracker: FreemiumTracker;

  beforeEach(() => {
    jest.clearAllMocks();
    tracker = new FreemiumTracker(5);
  });

  it('allows listening when no data stored', async () => {
    mockGetItem.mockResolvedValue(null);
    const canListen = await tracker.canListenFree();
    expect(canListen).toBe(true);
  });

  it('returns correct remaining listens on fresh day', async () => {
    mockGetItem.mockResolvedValue(null);
    const remaining = await tracker.getRemainingListens();
    expect(remaining).toBe(5);
  });

  it('decrements remaining after recordListen', async () => {
    const today = getTodayDate();
    mockGetItem.mockResolvedValue(JSON.stringify({ date: today, count: 2 }));

    const remaining = await tracker.recordListen();
    expect(remaining).toBe(2); // 5 - 3 = 2

    expect(mockSetItem).toHaveBeenCalledWith(
      'city-stories-freemium',
      JSON.stringify({ date: today, count: 3 }),
    );
  });

  it('returns 0 remaining when limit reached', async () => {
    const today = getTodayDate();
    mockGetItem.mockResolvedValue(JSON.stringify({ date: today, count: 5 }));

    const remaining = await tracker.getRemainingListens();
    expect(remaining).toBe(0);
  });

  it('canListenFree returns false when limit reached', async () => {
    const today = getTodayDate();
    mockGetItem.mockResolvedValue(JSON.stringify({ date: today, count: 5 }));

    const canListen = await tracker.canListenFree();
    expect(canListen).toBe(false);
  });

  it('resets counter for new day', async () => {
    mockGetItem.mockResolvedValue(JSON.stringify({ date: '2020-01-01', count: 5 }));

    const canListen = await tracker.canListenFree();
    expect(canListen).toBe(true);

    const remaining = await tracker.getRemainingListens();
    expect(remaining).toBe(5);
  });

  it('getUsedListens returns correct count', async () => {
    const today = getTodayDate();
    mockGetItem.mockResolvedValue(JSON.stringify({ date: today, count: 3 }));

    const used = await tracker.getUsedListens();
    expect(used).toBe(3);
  });

  it('reset clears storage', async () => {
    await tracker.reset();
    expect(AsyncStorage.removeItem).toHaveBeenCalledWith('city-stories-freemium');
  });

  it('getDailyLimit returns configured limit', () => {
    expect(tracker.getDailyLimit()).toBe(5);

    const customTracker = new FreemiumTracker(10);
    expect(customTracker.getDailyLimit()).toBe(10);
  });

  it('recordListen does not go below 0 remaining', async () => {
    const today = getTodayDate();
    mockGetItem.mockResolvedValue(JSON.stringify({ date: today, count: 6 }));

    const remaining = await tracker.recordListen();
    expect(remaining).toBe(0);
  });
});
