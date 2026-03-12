import axios, { type AxiosRequestConfig } from 'axios';
import { API_BASE_URL } from '../constants';
import { useAuthStore } from '../store/authStore';
import type { LoginResponse } from '../types';
import type { operations } from './generated';
import { createGeneratedApiClient } from './generated/runtime';

type RefreshTokenRequest =
  operations['refreshToken']['requestBody']['content']['application/json'];
type RefreshTokenResponse =
  operations['refreshToken']['responses']['200']['content']['application/json'];

const apiClient = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const generatedApiClient = createGeneratedApiClient(apiClient);

// Request interceptor: attach JWT token
apiClient.interceptors.request.use((config) => {
  const { token } = useAuthStore.getState();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Single-flight token refresh with pending-request queue
let isRefreshing = false;
let pendingQueue: {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}[] = [];

function processQueue(error: unknown, token: string | null) {
  for (const pending of pendingQueue) {
    if (token) {
      pending.resolve(token);
    } else {
      pending.reject(error);
    }
  }
  pendingQueue = [];
}

// Response interceptor: handle 401 with token refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean };

    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      originalRequest.url !== '/api/v1/auth/login' &&
      originalRequest.url !== '/api/v1/auth/refresh' &&
      originalRequest.url !== '/auth/login' &&
      originalRequest.url !== '/auth/refresh'
    ) {
      originalRequest._retry = true;

      // If a refresh is already in flight, queue this request
      if (isRefreshing) {
        return new Promise<string>((resolve, reject) => {
          pendingQueue.push({ resolve, reject });
        }).then((newToken) => {
          originalRequest.headers = {
            ...originalRequest.headers,
            Authorization: `Bearer ${newToken}`,
          };
          return apiClient(originalRequest);
        });
      }

      const { refreshToken, logout } = useAuthStore.getState();
      if (!refreshToken) {
        logout();
        return Promise.reject(error);
      }

      isRefreshing = true;

      try {
        const { data } = await axios.post<RefreshTokenResponse, { data: RefreshTokenResponse }, RefreshTokenRequest>(
          `${API_BASE_URL}/api/v1/auth/refresh`,
          { refresh_token: refreshToken },
        );
        if (!data.tokens) {
          throw new Error('Refresh response is missing tokens');
        }

        const newAccessToken = data.tokens.access_token;
        const newRefreshToken = data.tokens.refresh_token;

        // Only update if the user hasn't logged out during the refresh
        const { token: currentToken } = useAuthStore.getState();
        if (currentToken) {
          useAuthStore.getState().setTokens(newAccessToken, newRefreshToken);
        }

        processQueue(null, newAccessToken);

        originalRequest.headers = {
          ...originalRequest.headers,
          Authorization: `Bearer ${newAccessToken}`,
        };
        return apiClient(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError, null);
        logout();
        return Promise.reject(error);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  },
);

export async function login(email: string, password: string): Promise<LoginResponse> {
  const { data, error } = await generatedApiClient.POST('/auth/login', {
    body: { email, password },
  });
  if (error) {
    throw new Error(error.error);
  }
  if (!data.tokens) {
    throw new Error('Login response is missing tokens');
  }
  return data;
}

export default apiClient;
