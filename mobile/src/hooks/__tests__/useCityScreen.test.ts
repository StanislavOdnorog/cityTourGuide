// Mock AsyncStorage (required by useSettingsStore imported transitively)
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
}));

import type { CityPOI } from '@/types';
import { getMarkerColor } from '../useCityScreen';

const makePOI = (overrides?: Partial<CityPOI>): CityPOI => ({
  id: 1,
  city_id: 1,
  name: 'Narikala Fortress',
  name_ru: 'Крепость Нарикала',
  lat: 41.6875,
  lng: 44.8078,
  type: 'monument',
  tags: null,
  address: null,
  interest_score: 90,
  status: 'active',
  story_count: 3,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
});

describe('getMarkerColor', () => {
  it('returns grey for unpurchased POI with stories', () => {
    const poi = makePOI({ story_count: 2 });
    const listenedPoiIds = new Set<number>();
    expect(getMarkerColor(poi, listenedPoiIds, false)).toBe('grey');
  });

  it('returns blue for purchased POI that has not been listened to', () => {
    const poi = makePOI({ id: 5 });
    const listenedPoiIds = new Set<number>();
    expect(getMarkerColor(poi, listenedPoiIds, true)).toBe('blue');
  });

  it('returns green for listened POI', () => {
    const poi = makePOI({ id: 5 });
    const listenedPoiIds = new Set<number>([5]);
    expect(getMarkerColor(poi, listenedPoiIds, true)).toBe('green');
  });

  it('returns green for listened POI even when not purchased (listened before lock)', () => {
    const poi = makePOI({ id: 5, story_count: 0 });
    const listenedPoiIds = new Set<number>([5]);
    // story_count is 0, so it won't be grey even when not purchased
    expect(getMarkerColor(poi, listenedPoiIds, false)).toBe('green');
  });

  it('returns blue for unpurchased POI with 0 stories', () => {
    const poi = makePOI({ story_count: 0 });
    const listenedPoiIds = new Set<number>();
    expect(getMarkerColor(poi, listenedPoiIds, false)).toBe('blue');
  });

  it('returns grey before green for unpurchased POI with stories', () => {
    const poi = makePOI({ id: 5, story_count: 2 });
    const listenedPoiIds = new Set<number>([5]);
    // When not purchased and has stories, grey takes priority
    expect(getMarkerColor(poi, listenedPoiIds, false)).toBe('grey');
  });

  it('handles multiple listened POIs correctly', () => {
    const listenedPoiIds = new Set<number>([1, 3, 5]);

    expect(getMarkerColor(makePOI({ id: 1 }), listenedPoiIds, true)).toBe('green');
    expect(getMarkerColor(makePOI({ id: 2 }), listenedPoiIds, true)).toBe('blue');
    expect(getMarkerColor(makePOI({ id: 3 }), listenedPoiIds, true)).toBe('green');
    expect(getMarkerColor(makePOI({ id: 4 }), listenedPoiIds, true)).toBe('blue');
    expect(getMarkerColor(makePOI({ id: 5 }), listenedPoiIds, true)).toBe('green');
  });
});
