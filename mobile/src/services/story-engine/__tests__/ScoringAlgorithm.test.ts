import type { NearbyStoryCandidate } from '@/types';
import {
  bearing,
  angleDiff,
  proximityBonus,
  directionBonus,
  calculateScore,
  scoreAndRankCandidates,
} from '../ScoringAlgorithm';

function makeCandidate(overrides: Partial<NearbyStoryCandidate> = {}): NearbyStoryCandidate {
  return {
    poi_id: 1,
    poi_name: 'Test POI',
    poi_lat: 41.6875,
    poi_lng: 44.8084,
    story_id: 1,
    story_text: 'A story',
    audio_url: 'https://example.com/audio.mp3',
    duration_sec: 30,
    distance_m: 100,
    score: 50,
    ...overrides,
  };
}

describe('bearing', () => {
  it('returns 0 for due north', () => {
    // Point B is directly north of Point A
    const b = bearing(41.0, 44.0, 42.0, 44.0);
    expect(b).toBeCloseTo(0, 0);
  });

  it('returns ~90 for due east', () => {
    const b = bearing(41.0, 44.0, 41.0, 45.0);
    expect(b).toBeCloseTo(90, 0);
  });

  it('returns ~180 for due south', () => {
    const b = bearing(42.0, 44.0, 41.0, 44.0);
    expect(b).toBeCloseTo(180, 0);
  });

  it('returns ~270 for due west', () => {
    const b = bearing(41.0, 44.0, 41.0, 43.0);
    expect(b).toBeCloseTo(270, 0);
  });
});

describe('angleDiff', () => {
  it('returns 0 for same bearing', () => {
    expect(angleDiff(90, 90)).toBe(0);
  });

  it('returns 180 for opposite bearings', () => {
    expect(angleDiff(0, 180)).toBe(180);
  });

  it('handles wraparound (350° vs 10°)', () => {
    expect(angleDiff(350, 10)).toBe(20);
  });

  it('is symmetric', () => {
    expect(angleDiff(30, 60)).toBe(angleDiff(60, 30));
  });

  it('returns correct diff for 45°', () => {
    expect(angleDiff(0, 45)).toBe(45);
  });
});

describe('proximityBonus', () => {
  it('returns max bonus at distance 0', () => {
    expect(proximityBonus(0, 150)).toBe(30);
  });

  it('returns 0 at distance equal to radius', () => {
    expect(proximityBonus(150, 150)).toBe(0);
  });

  it('returns 0 beyond radius', () => {
    expect(proximityBonus(200, 150)).toBe(0);
  });

  it('returns half bonus at half radius', () => {
    expect(proximityBonus(75, 150)).toBeCloseTo(15, 5);
  });

  it('returns 0 when radius is 0', () => {
    expect(proximityBonus(50, 0)).toBe(0);
  });

  it('increases linearly as distance decreases', () => {
    const b1 = proximityBonus(100, 150);
    const b2 = proximityBonus(50, 150);
    const b3 = proximityBonus(0, 150);
    expect(b2).toBeGreaterThan(b1);
    expect(b3).toBeGreaterThan(b2);
  });
});

describe('directionBonus', () => {
  // User at (41.0, 44.0), heading east (90°), POI at (41.0, 45.0) → due east
  it('returns bonus when POI is ahead', () => {
    const bonus = directionBonus(50, 90, 41.0, 44.0, 41.0, 45.0);
    expect(bonus).toBeCloseTo(10, 1); // 20% of 50
  });

  // User heading east (90°), POI is due west
  it('returns 0 when POI is behind', () => {
    const bonus = directionBonus(50, 90, 41.0, 44.0, 41.0, 43.0);
    expect(bonus).toBe(0);
  });

  // User heading north (0°), POI is exactly at ±45°
  it('returns bonus at angle limit (45°)', () => {
    const bonus = directionBonus(50, 0, 41.0, 44.0, 42.0, 45.0);
    // bearing ≈ 37° (northeast), within 45° limit
    expect(bonus).toBeGreaterThan(0);
  });

  it('returns 0 when heading is negative (unavailable)', () => {
    const bonus = directionBonus(50, -1, 41.0, 44.0, 41.0, 45.0);
    expect(bonus).toBe(0);
  });
});

describe('calculateScore', () => {
  it('combines all score components', () => {
    // base=50, distance=0 (max proximity=30), heading=90, POI east (direction=+10)
    const score = calculateScore(50, 0, 150, 90, 41.0, 44.0, 41.0, 45.0);
    expect(score).toBeCloseTo(90, 0); // 50 + 30 + 10
  });

  it('returns only base score with no bonuses', () => {
    // distance at radius (no proximity bonus), heading negative (no direction bonus)
    const score = calculateScore(50, 150, 150, -1, 41.0, 44.0, 41.0, 45.0);
    expect(score).toBe(50);
  });
});

describe('scoreAndRankCandidates', () => {
  it('sorts candidates by score descending', () => {
    const candidates = [
      makeCandidate({ story_id: 1, score: 30 }),
      makeCandidate({ story_id: 2, score: 80 }),
      makeCandidate({ story_id: 3, score: 50 }),
    ];
    const result = scoreAndRankCandidates(candidates, new Set());
    expect(result.map((c) => c.story_id)).toEqual([2, 3, 1]);
  });

  it('filters out listened stories', () => {
    const candidates = [
      makeCandidate({ story_id: 1, score: 80 }),
      makeCandidate({ story_id: 2, score: 60 }),
      makeCandidate({ story_id: 3, score: 40 }),
    ];
    const listened = new Set([1, 3]);
    const result = scoreAndRankCandidates(candidates, listened);
    expect(result).toHaveLength(1);
    expect(result[0].story_id).toBe(2);
  });

  it('returns empty array when all stories listened', () => {
    const candidates = [
      makeCandidate({ story_id: 1, score: 80 }),
      makeCandidate({ story_id: 2, score: 60 }),
    ];
    const listened = new Set([1, 2]);
    const result = scoreAndRankCandidates(candidates, listened);
    expect(result).toHaveLength(0);
  });

  it('returns empty array for empty input', () => {
    const result = scoreAndRankCandidates([], new Set());
    expect(result).toHaveLength(0);
  });

  it('preserves candidate fields in result', () => {
    const candidates = [
      makeCandidate({
        poi_id: 5,
        poi_name: 'Church',
        story_id: 10,
        story_text: 'A beautiful church',
        audio_url: 'https://example.com/church.mp3',
        duration_sec: 25,
        distance_m: 42,
        score: 70,
      }),
    ];
    const result = scoreAndRankCandidates(candidates, new Set());
    expect(result[0].poi_id).toBe(5);
    expect(result[0].poi_name).toBe('Church');
    expect(result[0].story_id).toBe(10);
    expect(result[0].audio_url).toBe('https://example.com/church.mp3');
    expect(result[0].duration_sec).toBe(25);
    expect(result[0].distance_m).toBe(42);
    expect(result[0].localScore).toBe(70);
  });

  it('closest POI gets higher score when server scores reflect proximity', () => {
    const candidates = [
      makeCandidate({ story_id: 1, distance_m: 200, score: 40 }),
      makeCandidate({ story_id: 2, distance_m: 10, score: 75 }),
    ];
    const result = scoreAndRankCandidates(candidates, new Set());
    expect(result[0].story_id).toBe(2);
    expect(result[0].localScore).toBeGreaterThan(result[1].localScore);
  });
});
