export interface User {
  id: string;
  email: string | null;
  name: string | null;
  auth_provider: 'email' | 'google' | 'apple';
  language_pref: string;
  is_anonymous: boolean;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface LoginResponse {
  data: User;
  tokens: TokenPair;
}

export interface ApiError {
  error: string;
}

// Pagination
export interface PaginationMeta {
  total: number;
  page: number;
  per_page: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

// City
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

// POI
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

export interface POI {
  id: number;
  city_id: number;
  name: string;
  name_ru: string | null;
  lat: number;
  lng: number;
  type: POIType;
  tags: Record<string, unknown> | null;
  address: string | null;
  interest_score: number;
  status: POIStatus;
  created_at: string;
  updated_at: string;
}

// Story
export type StoryLayerType =
  | 'atmosphere'
  | 'human_story'
  | 'hidden_detail'
  | 'time_shift'
  | 'general';
export type StoryStatus = 'active' | 'disabled' | 'reported' | 'pending_review';

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

// Report
export type ReportType = 'wrong_location' | 'wrong_fact' | 'inappropriate_content';
export type ReportStatus = 'new' | 'reviewed' | 'resolved' | 'dismissed';

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
