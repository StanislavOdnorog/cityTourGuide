import type { PurchaseStatus } from '@/types';
import { usePurchaseStore } from '../usePurchaseStore';

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

const makePurchaseStatus = (overrides: Partial<PurchaseStatus> = {}): PurchaseStatus => ({
  has_full_access: false,
  is_lifetime: false,
  active_subscription: null,
  city_packs: [],
  free_stories_used: 0,
  free_stories_limit: 5,
  free_stories_left: 5,
  ...overrides,
});

describe('usePurchaseStore', () => {
  beforeEach(() => {
    usePurchaseStore.setState({
      status: null,
      isLoading: false,
      paywallVisible: false,
      _hasHydrated: false,
    });
  });

  it('has correct initial state', () => {
    const state = usePurchaseStore.getState();
    expect(state.status).toBeNull();
    expect(state.isLoading).toBe(false);
    expect(state.paywallVisible).toBe(false);
  });

  it('setStatus updates purchase status', () => {
    const status = makePurchaseStatus({ free_stories_left: 3 });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().status?.free_stories_left).toBe(3);
  });

  it('showPaywall / hidePaywall toggles paywall visibility', () => {
    usePurchaseStore.getState().showPaywall();
    expect(usePurchaseStore.getState().paywallVisible).toBe(true);

    usePurchaseStore.getState().hidePaywall();
    expect(usePurchaseStore.getState().paywallVisible).toBe(false);
  });

  it('hasFullAccess returns true for lifetime purchase', () => {
    const status = makePurchaseStatus({ has_full_access: true, is_lifetime: true });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().hasFullAccess()).toBe(true);
  });

  it('hasFullAccess returns false without purchases', () => {
    const status = makePurchaseStatus();
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().hasFullAccess()).toBe(false);
  });

  it('canListenFree returns true when free stories left', () => {
    const status = makePurchaseStatus({ free_stories_left: 3 });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().canListenFree()).toBe(true);
  });

  it('canListenFree returns false when no free stories left', () => {
    const status = makePurchaseStatus({ free_stories_left: 0, free_stories_used: 5 });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().canListenFree()).toBe(false);
  });

  it('canListenFree returns true for full access regardless of free count', () => {
    const status = makePurchaseStatus({ has_full_access: true, free_stories_left: 0 });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().canListenFree()).toBe(true);
  });

  it('decrementFreeStories reduces free_stories_left', () => {
    const status = makePurchaseStatus({ free_stories_used: 2, free_stories_left: 3 });
    usePurchaseStore.getState().setStatus(status);
    usePurchaseStore.getState().decrementFreeStories();

    const updated = usePurchaseStore.getState().status;
    expect(updated?.free_stories_used).toBe(3);
    expect(updated?.free_stories_left).toBe(2);
  });

  it('decrementFreeStories does not go below 0', () => {
    const status = makePurchaseStatus({ free_stories_used: 5, free_stories_left: 0 });
    usePurchaseStore.getState().setStatus(status);
    usePurchaseStore.getState().decrementFreeStories();

    const updated = usePurchaseStore.getState().status;
    expect(updated?.free_stories_left).toBe(0);
    expect(updated?.free_stories_used).toBe(6);
  });

  it('hasCityAccess returns true for full access', () => {
    const status = makePurchaseStatus({ has_full_access: true });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().hasCityAccess(1)).toBe(true);
  });

  it('hasCityAccess returns true for matching city pack', () => {
    const status = makePurchaseStatus({
      city_packs: [
        {
          id: 1,
          user_id: 'u1',
          type: 'city_pack',
          city_id: 42,
          platform: 'ios',
          transaction_id: 't1',
          price: 4.99,
          is_ltd: false,
          expires_at: null,
          created_at: '2026-01-01',
        },
      ],
    });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().hasCityAccess(42)).toBe(true);
  });

  it('hasCityAccess returns false for non-matching city pack', () => {
    const status = makePurchaseStatus({
      city_packs: [
        {
          id: 1,
          user_id: 'u1',
          type: 'city_pack',
          city_id: 42,
          platform: 'ios',
          transaction_id: 't1',
          price: 4.99,
          is_ltd: false,
          expires_at: null,
          created_at: '2026-01-01',
        },
      ],
      free_stories_left: 0,
    });
    usePurchaseStore.getState().setStatus(status);
    expect(usePurchaseStore.getState().hasCityAccess(99)).toBe(false);
  });
});
