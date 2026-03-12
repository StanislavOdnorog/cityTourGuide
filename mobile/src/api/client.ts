import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { API_BASE_URL } from '@/constants';
import { createGeneratedApiClient } from './generated/runtime';
import {
  canRetryTooManyRequests,
  cloneRequestConfigForRetry,
  computeRetryDelayMs,
  RetryableRequestConfig,
} from './retry';

let accessToken: string | null = null;
let refreshToken: string | null = null;
let onRefreshToken: ((token: string) => Promise<string | null>) | null = null;
let offlineEnqueue:
  | ((request: {
      endpoint: string;
      method: string;
      body: unknown;
      headers: Record<string, string>;
    }) => Promise<boolean>)
  | null = null;

const SAFE_OFFLINE_ENDPOINTS = ['/listenings', '/reports', '/device-tokens'];
const MUTATION_METHODS = ['post', 'put', 'delete'];

export function setOfflineEnqueue(handler: typeof offlineEnqueue): void {
  offlineEnqueue = handler;
}

function isNetworkError(error: AxiosError): boolean {
  return (
    !error.response &&
    error.code !== 'ERR_CANCELED' &&
    (error.code === 'ERR_NETWORK' ||
      error.code === 'ECONNABORTED' ||
      error.message === 'Network Error')
  );
}

function isSafeOfflineEndpoint(url: string | undefined): boolean {
  if (!url) return false;
  return SAFE_OFFLINE_ENDPOINTS.some((ep) => url.includes(ep));
}

function isMutationMethod(method: string | undefined): boolean {
  return MUTATION_METHODS.includes((method ?? '').toLowerCase());
}

export function setTokens(access: string | null, refresh: string | null): void {
  accessToken = access;
  refreshToken = refresh;
}

export function setRefreshHandler(
  handler: ((token: string) => Promise<string | null>) | null,
): void {
  onRefreshToken = handler;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function hasRefreshToken(): boolean {
  return Boolean(refreshToken && onRefreshToken);
}

const apiClient = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
});

apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string | null) => void;
  reject: (error: unknown) => void;
}> = [];

function processQueue(error: unknown, token: string | null): void {
  for (const promise of failedQueue) {
    if (error) {
      promise.reject(error);
    } else {
      promise.resolve(token);
    }
  }
  failedQueue = [];
}

export async function refreshAccessToken(): Promise<string | null> {
  if (!refreshToken || !onRefreshToken) {
    return accessToken;
  }

  if (isRefreshing) {
    return new Promise((resolve, reject) => {
      failedQueue.push({ resolve, reject });
    });
  }

  isRefreshing = true;

  try {
    const newToken = await onRefreshToken(refreshToken);
    if (newToken) {
      accessToken = newToken;
    }
    processQueue(null, newToken);
    return newToken;
  } catch (refreshError) {
    processQueue(refreshError, null);
    throw refreshError;
  } finally {
    isRefreshing = false;
  }
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as RetryableRequestConfig | undefined;

    if (!originalRequest || error.code === 'ERR_CANCELED' || originalRequest.signal?.aborted) {
      return Promise.reject(error);
    }

    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      refreshToken &&
      onRefreshToken
    ) {
      originalRequest._retry = true;
      originalRequest._skip429Retry = true;

      try {
        const newToken = await refreshAccessToken();
        if (newToken) {
          const retryRequest = cloneRequestConfigForRetry(originalRequest);
          retryRequest.headers.set('Authorization', `Bearer ${newToken}`);
          return apiClient(retryRequest);
        }
      } catch (refreshError) {
        return Promise.reject(refreshError);
      }
    }

    if (error.response?.status === 429 && canRetryTooManyRequests(originalRequest)) {
      const retryCount = originalRequest._retryCount ?? 0;
      const retryAfterHeader = error.response.headers['retry-after'];
      const delayMs = computeRetryDelayMs(
        retryCount,
        Array.isArray(retryAfterHeader) ? retryAfterHeader[0] : retryAfterHeader,
      );

      const retryRequest = cloneRequestConfigForRetry(originalRequest, {
        _retryCount: retryCount + 1,
      });

      await new Promise((resolve) => setTimeout(resolve, delayMs));

      if (retryRequest.signal?.aborted) {
        return Promise.reject(error);
      }

      return apiClient(retryRequest);
    }

    // Queue safe mutations that fail due to network errors
    if (
      offlineEnqueue &&
      !originalRequest._skipOfflineQueue &&
      isNetworkError(error) &&
      isMutationMethod(originalRequest.method) &&
      isSafeOfflineEndpoint(originalRequest.url)
    ) {
      const headers: Record<string, string> = {};
      if (originalRequest.headers) {
        const authHeader =
          originalRequest.headers.Authorization ?? originalRequest.headers.authorization;
        if (typeof authHeader === 'string') {
          headers['Authorization'] = authHeader;
        }
      }
      await offlineEnqueue({
        endpoint: originalRequest.url ?? '',
        method: originalRequest.method ?? 'POST',
        body: originalRequest.data,
        headers,
      });
      // Resolve silently — the mutation is queued for later
      return { data: null, status: 0, statusText: 'queued', headers: {}, config: originalRequest };
    }

    return Promise.reject(error);
  },
);

export const generatedApiClient = createGeneratedApiClient(apiClient);

export default apiClient;
