import { AxiosHeaders } from 'axios';
import type { InternalAxiosRequestConfig } from 'axios';

export const SAFE_RETRY_METHODS = new Set(['GET', 'HEAD']);
export const DEFAULT_RETRY_BASE_DELAY_MS = 1000;
export const DEFAULT_RETRY_MAX_DELAY_MS = 8000;
export const DEFAULT_RETRY_JITTER_MS = 250;
export const DEFAULT_RETRY_ATTEMPTS = 2;

export type RetryableRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean;
  _retryCount?: number;
  _skip429Retry?: boolean;
  _skipOfflineQueue?: boolean;
  retryOn429?: boolean;
};

export function isSafeRetryMethod(method?: string): boolean {
  return SAFE_RETRY_METHODS.has((method ?? 'GET').toUpperCase());
}

export function canRetryTooManyRequests(config?: RetryableRequestConfig): boolean {
  if (!config || config._skip429Retry || config.retryOn429 === false) {
    return false;
  }

  if ((config._retryCount ?? 0) >= DEFAULT_RETRY_ATTEMPTS) {
    return false;
  }

  return config.retryOn429 === true || isSafeRetryMethod(config.method);
}

export function parseRetryAfterMs(
  retryAfter: string | null | undefined,
  now = Date.now(),
): number | null {
  if (!retryAfter) {
    return null;
  }

  const seconds = Number(retryAfter);
  if (Number.isFinite(seconds) && seconds >= 0) {
    return seconds * 1000;
  }

  const retryAt = Date.parse(retryAfter);
  if (Number.isNaN(retryAt)) {
    return null;
  }

  return Math.max(0, retryAt - now);
}

export function computeRetryDelayMs(
  retryCount: number,
  retryAfter: string | null | undefined,
  options?: {
    now?: number;
    baseDelayMs?: number;
    maxDelayMs?: number;
    jitterMs?: number;
    random?: () => number;
  },
): number {
  const baseDelayMs = options?.baseDelayMs ?? DEFAULT_RETRY_BASE_DELAY_MS;
  const maxDelayMs = options?.maxDelayMs ?? DEFAULT_RETRY_MAX_DELAY_MS;
  const jitterMs = options?.jitterMs ?? DEFAULT_RETRY_JITTER_MS;

  const retryAfterMs = parseRetryAfterMs(retryAfter, options?.now);
  if (retryAfterMs !== null) {
    return Math.min(retryAfterMs, maxDelayMs);
  }

  const exponentialDelay = Math.min(baseDelayMs * 2 ** retryCount, maxDelayMs);
  const jitter = Math.floor((options?.random ?? Math.random)() * jitterMs);
  return Math.min(exponentialDelay + jitter, maxDelayMs);
}

export function cloneRequestConfigForRetry(
  config: RetryableRequestConfig,
  overrides: Partial<RetryableRequestConfig> = {},
): RetryableRequestConfig {
  const headers = AxiosHeaders.from(config.headers ?? {});
  const overrideHeaders = AxiosHeaders.from(overrides.headers ?? {});

  return {
    ...config,
    ...overrides,
    headers: AxiosHeaders.from({
      ...headers.toJSON(),
      ...overrideHeaders.toJSON(),
    }),
  };
}
