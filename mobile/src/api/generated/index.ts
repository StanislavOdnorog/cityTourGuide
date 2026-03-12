/**
 * Re-exports from generated OpenAPI types for convenient usage.
 * Do not edit — regenerate with `make generate-api` from backend/.
 */
export type { components, operations, paths } from './schema';

import type { components, operations } from './schema';

// Schema types
export type City = components['schemas']['City'];
export type POI = components['schemas']['POI'];
export type Story = components['schemas']['Story'];
export type StoryCandidate = components['schemas']['StoryCandidate'];
export type User = components['schemas']['User'];
export type TokenPair = components['schemas']['TokenPair'];
export type UserListening = components['schemas']['UserListening'];
export type Report = components['schemas']['Report'];
export type DeviceToken = components['schemas']['DeviceToken'];
export type Purchase = components['schemas']['Purchase'];
export type PurchaseStatus = components['schemas']['PurchaseStatus'];
export type InflationJob = components['schemas']['InflationJob'];
export type DownloadManifestItem = components['schemas']['DownloadManifestItem'];
export type ApiError = components['schemas']['Error'];
export type ValidationError = components['schemas']['ValidationError'];
export type ValidationDetail = components['schemas']['ValidationDetail'];

// Enum-like types extracted from schemas
export type POIType = POI['type'];
export type POIStatus = POI['status'];
export type StoryLayerType = Story['layer_type'];
export type StoryStatus = Story['status'];
export type AuthProvider = User['auth_provider'];
export type ReportType = Report['type'];
export type ReportStatus = Report['status'];
export type PurchaseType = Purchase['type'];
export type InflationJobStatus = InflationJob['status'];
export type InflationTriggerType = InflationJob['trigger_type'];
export type DevicePlatform = DeviceToken['platform'];

// Operation-derived request body types
export type TrackListeningRequest =
  operations['trackListening']['requestBody']['content']['application/json'];
export type CreateReportRequest =
  operations['createReport']['requestBody']['content']['application/json'];
export type VerifyPurchaseRequest =
  operations['verifyPurchase']['requestBody']['content']['application/json'];
export type RegisterDeviceTokenRequest =
  operations['registerDeviceToken']['requestBody']['content']['application/json'];

// Operation-derived query parameter types
export type NearbyStoriesParams = operations['getNearbyStories']['parameters']['query'];

// Operation-derived response types
export type NearbyStoriesResponse =
  operations['getNearbyStories']['responses']['200']['content']['application/json'];
export type DownloadManifestResponse =
  operations['getCityDownloadManifest']['responses']['200']['content']['application/json'];
