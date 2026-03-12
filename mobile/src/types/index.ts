// Domain types — re-exported from generated OpenAPI types
export type {
  City,
  POI,
  Story,
  User,
  TokenPair,
  UserListening,
  Report,
  DeviceToken,
  Purchase,
  PurchaseStatus,
  DownloadManifestItem,
  PaginationMeta,
  ApiError,
  POIType,
  POIStatus,
  StoryLayerType,
  StoryStatus,
  AuthProvider,
  ReportType,
  ReportStatus,
  PurchaseType,
  StoryCandidate,
  // Operation-derived types
  TrackListeningRequest,
  CreateReportRequest,
  VerifyPurchaseRequest,
  RegisterDeviceTokenRequest,
  NearbyStoriesParams,
  NearbyStoriesResponse,
  DownloadManifestResponse,
} from '@/api/generated';

import type { POI } from '@/api/generated';

// Aliased for backward compatibility
export type { StoryCandidate as NearbyStoryCandidate } from '@/api/generated';
export type { CreateReportRequest as ReportStoryRequest } from '@/api/generated';
export type { PurchaseStatus as PurchaseStatusResponse } from '@/api/generated';
export type { DownloadManifestResponse as CityDownloadManifest } from '@/api/generated';

// Types not in OpenAPI spec (cities/{id}/pois endpoint is not spec'd)
export interface CityPOI extends POI {
  story_count: number;
  distance_m?: number;
}

export interface CityPOIsResponse {
  data: CityPOI[];
  total_stories: number;
}
