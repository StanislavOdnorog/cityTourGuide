import { AxiosError } from 'axios';

/**
 * Stable error categories for mobile API failures.
 * Consumers switch on `category` instead of inspecting raw errors.
 */
export type AppApiErrorCategory = 'network' | 'unauthorized' | 'validation' | 'server' | 'unknown';

export class AppApiError extends Error {
  readonly category: AppApiErrorCategory;
  readonly status: number | undefined;
  readonly traceId: string | undefined;
  /** Backend error message (may differ from `message` which is user-safe). */
  readonly serverMessage: string | undefined;

  constructor(opts: {
    category: AppApiErrorCategory;
    message: string;
    status?: number;
    traceId?: string;
    serverMessage?: string;
    cause?: unknown;
  }) {
    super(opts.message, { cause: opts.cause });
    this.name = 'AppApiError';
    this.category = opts.category;
    this.status = opts.status;
    this.traceId = opts.traceId;
    this.serverMessage = opts.serverMessage;
  }
}

/**
 * Returns true if the value is an AppApiError.
 */
export function isAppApiError(err: unknown): err is AppApiError {
  return err instanceof AppApiError;
}

/**
 * Extract a backend error body shape from an Axios response, if present.
 */
function extractBackendBody(error: AxiosError): { error?: string; trace_id?: string } | undefined {
  const data = error.response?.data;
  if (data && typeof data === 'object') {
    return data as { error?: string; trace_id?: string };
  }
  return undefined;
}

/**
 * Normalize any error thrown by Axios or the generated client into an AppApiError.
 *
 * The generated openapi-fetch client returns `{ data, error }` — callers in
 * endpoints.ts handle that shape themselves and call `normalizeGeneratedError`
 * instead.  This function handles raw Axios / network failures that bubble up
 * from the interceptor layer.
 */
export function normalizeError(err: unknown): AppApiError {
  if (err instanceof AppApiError) {
    return err;
  }

  if (
    err instanceof AxiosError ||
    (err != null &&
      typeof err === 'object' &&
      'isAxiosError' in err &&
      (err as AxiosError).isAxiosError)
  ) {
    return normalizeAxiosError(err as AxiosError);
  }

  const message = err instanceof Error ? err.message : 'An unexpected error occurred';
  return new AppApiError({
    category: 'unknown',
    message,
    cause: err,
  });
}

function normalizeAxiosError(err: AxiosError): AppApiError {
  const body = extractBackendBody(err);
  const traceId = typeof body?.trace_id === 'string' ? body.trace_id : undefined;
  const serverMessage = typeof body?.error === 'string' ? body.error : undefined;
  const status = err.response?.status;

  // Network-level failure (no response received)
  if (
    !err.response &&
    err.code !== 'ERR_CANCELED' &&
    (err.code === 'ERR_NETWORK' || err.code === 'ECONNABORTED' || err.message === 'Network Error')
  ) {
    return new AppApiError({
      category: 'network',
      message: 'Unable to connect. Please check your internet connection and try again.',
      traceId,
      serverMessage,
      cause: err,
    });
  }

  if (status === 401) {
    return new AppApiError({
      category: 'unauthorized',
      message: 'Your session has expired. Please sign in again.',
      status,
      traceId,
      serverMessage,
      cause: err,
    });
  }

  if (status === 400 || status === 422) {
    return new AppApiError({
      category: 'validation',
      message: serverMessage ?? 'The request was invalid. Please check your input.',
      status,
      traceId,
      serverMessage,
      cause: err,
    });
  }

  if (status !== undefined && status >= 500) {
    return new AppApiError({
      category: 'server',
      message: 'Something went wrong on our end. Please try again later.',
      status,
      traceId,
      serverMessage,
      cause: err,
    });
  }

  return new AppApiError({
    category: 'unknown',
    message: serverMessage ?? err.message ?? 'An unexpected error occurred',
    status,
    traceId,
    serverMessage,
    cause: err,
  });
}

/**
 * Normalize an error body returned by the generated openapi-fetch client.
 * The generated client does not throw — it returns `{ error }` with a body shape.
 */
export function normalizeGeneratedError(
  errorBody: { error?: string; trace_id?: string } | undefined,
  fallbackMessage: string,
  status?: number,
): AppApiError {
  const serverMessage = typeof errorBody?.error === 'string' ? errorBody.error : undefined;
  const traceId = typeof errorBody?.trace_id === 'string' ? errorBody.trace_id : undefined;

  let category: AppApiErrorCategory = 'unknown';
  if (status === 401) category = 'unauthorized';
  else if (status === 400 || status === 422) category = 'validation';
  else if (status !== undefined && status >= 500) category = 'server';

  const message =
    category === 'validation' && serverMessage ? serverMessage : (serverMessage ?? fallbackMessage);

  return new AppApiError({
    category,
    message,
    status,
    traceId,
    serverMessage,
  });
}

/**
 * Returns a user-friendly message for display, based on error category.
 */
export function userMessageForError(err: unknown): string {
  if (isAppApiError(err)) {
    switch (err.category) {
      case 'network':
        return 'Unable to connect. Please check your internet connection and try again.';
      case 'unauthorized':
        return 'Your session has expired. Please sign in again.';
      case 'validation':
        return err.serverMessage ?? 'The request was invalid. Please check your input.';
      case 'server':
        return 'Something went wrong on our end. Please try again later.';
      default:
        return err.message;
    }
  }
  return 'An unexpected error occurred. Please try again.';
}
