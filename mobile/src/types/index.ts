// Domain types matching backend Go structs

export type POIType =
  | 'building'
  | 'street'
  | 'park'
  | 'monument'
  | 'church'
  | 'bridge'
  | 'square'
  | 'museum'
  | 'district'
  | 'other';

export type POIStatus = 'active' | 'disabled' | 'pending_review';

export type StoryLayerType =
  | 'atmosphere'
  | 'human_story'
  | 'hidden_detail'
  | 'time_shift'
  | 'general';

export type StoryStatus = 'active' | 'disabled' | 'reported' | 'pending_review';

export type AuthProvider = 'email' | 'google' | 'apple';

export type ReportType = 'wrong_location' | 'wrong_fact' | 'inappropriate_content';

export type ReportStatus = 'new' | 'reviewed' | 'resolved' | 'dismissed';

export type PurchaseType = 'city_pack' | 'subscription' | 'lifetime';

export interface City {
  id: number;
  name: string;
  name_ru: string | null;
  country: string;
  center_lat: number;
  center_lng: number;
  radius_km: number;
  is_active: boolean;
  download_size_mb: number;
  created_at: string;
  updated_at: string;
}

export interface POI {
  id: number;
  city_id: number;
  name: string;
  name_ru: string | null;
  lat: number;
  lng: number;
  type: POIType;
  tags: unknown;
  address: string | null;
  interest_score: number;
  status: POIStatus;
  created_at: string;
  updated_at: string;
}

export interface Story {
  id: number;
  poi_id: number;
  language: string;
  text: string;
  audio_url: string | null;
  duration_sec: number | null;
  layer_type: StoryLayerType;
  order_index: number;
  is_inflation: boolean;
  confidence: number;
  sources: unknown;
  status: StoryStatus;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: string;
  email: string | null;
  name: string | null;
  auth_provider: AuthProvider;
  language_pref: string;
  is_anonymous: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserListening {
  id: number;
  user_id: string;
  story_id: number;
  listened_at: string;
  completed: boolean;
  lat: number | null;
  lng: number | null;
}

export interface Report {
  id: number;
  story_id: number;
  user_id: string;
  type: ReportType;
  comment: string | null;
  user_lat: number | null;
  user_lng: number | null;
  status: ReportStatus;
  resolved_at: string | null;
  created_at: string;
}

export interface Purchase {
  id: number;
  user_id: string;
  type: PurchaseType;
  city_id: number | null;
  platform: string;
  transaction_id: string | null;
  price: number;
  is_ltd: boolean;
  expires_at: string | null;
  created_at: string;
}

// API response types

export interface NearbyStoryCandidate {
  poi_id: number;
  poi_name: string;
  story_id: number;
  story_text: string;
  audio_url: string | null;
  duration_sec: number | null;
  distance_m: number;
  score: number;
}

export interface NearbyStoriesResponse {
  data: NearbyStoryCandidate[];
}

export interface TrackListeningRequest {
  user_id: string;
  story_id: number;
  completed: boolean;
  lat?: number;
  lng?: number;
}

export interface ReportStoryRequest {
  story_id: number;
  user_id: string;
  type: ReportType;
  comment?: string;
  lat?: number;
  lng?: number;
}

export interface CityPOI extends POI {
  story_count: number;
  distance_m?: number;
}

export interface CityPOIsResponse {
  data: CityPOI[];
  total_stories: number;
}

export interface DownloadManifestItem {
  story_id: number;
  poi_id: number;
  poi_name: string;
  audio_url: string | null;
  duration_sec: number | null;
  file_size_bytes: number;
}

export interface CityDownloadManifest {
  data: DownloadManifestItem[];
  total_size_bytes: number;
  total_stories: number;
  city_name: string;
}

export interface ApiError {
  error: string;
}

// Purchase API types

export interface VerifyPurchaseRequest {
  platform: 'ios' | 'android';
  transaction_id: string;
  receipt: string;
  type: PurchaseType;
  city_id?: number;
  price: number;
}

export interface PurchaseStatusResponse {
  has_full_access: boolean;
  is_lifetime: boolean;
  active_subscription: Purchase | null;
  city_packs: Purchase[];
  free_stories_used: number;
  free_stories_limit: number;
  free_stories_left: number;
}
