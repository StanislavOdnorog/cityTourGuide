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
