import type { NearbyStoryCandidate } from '@/types';

const MAX_PROXIMITY_BONUS = 30;
const DIRECTION_BONUS_FACTOR = 0.2;
const DIRECTION_ANGLE_LIMIT = 45;

export interface ScoredCandidate extends NearbyStoryCandidate {
  localScore: number;
}

/**
 * Computes initial bearing from point A to point B in degrees [0, 360).
 */
export function bearing(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const toRad = Math.PI / 180;
  const phi1 = lat1 * toRad;
  const phi2 = lat2 * toRad;
  const deltaLambda = (lng2 - lng1) * toRad;

  const y = Math.sin(deltaLambda) * Math.cos(phi2);
  const x =
    Math.cos(phi1) * Math.sin(phi2) - Math.sin(phi1) * Math.cos(phi2) * Math.cos(deltaLambda);

  const theta = Math.atan2(y, x);
  return ((theta * 180) / Math.PI + 360) % 360;
}

/**
 * Returns the smallest angular difference between two bearings in degrees [0, 180].
 */
export function angleDiff(a: number, b: number): number {
  const diff = Math.abs(a - b);
  return diff > 180 ? 360 - diff : diff;
}

/**
 * Proximity bonus: linearly increases from 0 (at radius) to MAX_PROXIMITY_BONUS (at distance=0).
 */
export function proximityBonus(distanceM: number, radiusM: number): number {
  if (radiusM <= 0) return 0;
  const ratio = distanceM / radiusM;
  if (ratio >= 1.0) return 0;
  return MAX_PROXIMITY_BONUS * (1.0 - ratio);
}

/**
 * Direction bonus: +20% of base score if POI is within ±45° of user's heading.
 * Returns 0 if heading is negative (unavailable).
 */
export function directionBonus(
  baseScore: number,
  heading: number,
  userLat: number,
  userLng: number,
  poiLat: number,
  poiLng: number,
): number {
  if (heading < 0) return 0;
  const brng = bearing(userLat, userLng, poiLat, poiLng);
  const diff = angleDiff(heading, brng);
  return diff <= DIRECTION_ANGLE_LIMIT ? DIRECTION_BONUS_FACTOR * baseScore : 0;
}

/**
 * Computes composite score for a story candidate.
 * Matches the backend algorithm: score = base + proximity_bonus + direction_bonus
 */
export function calculateScore(
  baseInterestScore: number,
  distanceM: number,
  radiusM: number,
  heading: number,
  userLat: number,
  userLng: number,
  poiLat: number,
  poiLng: number,
): number {
  let score = baseInterestScore;
  score += proximityBonus(distanceM, radiusM);
  score += directionBonus(baseInterestScore, heading, userLat, userLng, poiLat, poiLng);
  return score;
}

/**
 * Filters out listened stories and sorts candidates by server-computed score.
 * The backend already computes proximity_bonus and direction_bonus;
 * the client applies recently_played_penalty (exclusion) and sorts.
 */
export function scoreAndRankCandidates(
  candidates: NearbyStoryCandidate[],
  listenedStoryIds: Set<number>,
): ScoredCandidate[] {
  const scored: ScoredCandidate[] = [];

  for (const candidate of candidates) {
    if (listenedStoryIds.has(candidate.story_id)) {
      continue;
    }

    scored.push({
      ...candidate,
      localScore: candidate.score,
    });
  }

  scored.sort((a, b) => b.localScore - a.localScore);
  return scored;
}
