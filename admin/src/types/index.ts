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
  InflationJob,
  InflationJobStatus,
  InflationTriggerType,
  operations,
} from '../api/generated';

import type { operations } from '../api/generated';

export type LoginResponse = operations['login']['responses']['200']['content']['application/json'];
