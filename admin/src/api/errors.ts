import axios from 'axios';
import { notification } from 'antd';

export type NormalizedApiErrorDetail = {
  field?: string;
  message: string;
};

export type NormalizedApiErrorShape = {
  message: string;
  details: NormalizedApiErrorDetail[];
  requestId?: string;
  status?: number;
};

type ErrorPayload = {
  error?: unknown;
  details?: unknown;
  trace_id?: unknown;
  request_id?: unknown;
  message?: unknown;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function normalizeDetails(details: unknown): NormalizedApiErrorDetail[] {
  if (!Array.isArray(details)) {
    return [];
  }

  return details.flatMap((detail) => {
    if (!isRecord(detail)) {
      return [];
    }

    const field = typeof detail.field === 'string' ? detail.field : undefined;
    const message = typeof detail.message === 'string' ? detail.message : undefined;
    if (!message) {
      return [];
    }

    return [{ field, message }];
  });
}

function formatValidationMessage(details: NormalizedApiErrorDetail[]): string {
  return details
    .map((detail) => (detail.field ? `${detail.field}: ${detail.message}` : detail.message))
    .join('; ');
}

function getPayload(value: unknown): ErrorPayload | undefined {
  if (!isRecord(value)) {
    return undefined;
  }

  const response = isRecord(value.response) ? value.response : undefined;
  if (response && isRecord(response.data)) {
    return response.data as ErrorPayload;
  }

  if (isRecord(value.data)) {
    return value.data as ErrorPayload;
  }

  if (
    'error' in value ||
    'details' in value ||
    'trace_id' in value ||
    'request_id' in value ||
    'message' in value
  ) {
    return value as ErrorPayload;
  }

  return undefined;
}

function getStatus(value: unknown): number | undefined {
  if (!isRecord(value)) {
    return undefined;
  }

  if (typeof value.status === 'number') {
    return value.status;
  }

  if (isRecord(value.response) && typeof value.response.status === 'number') {
    return value.response.status;
  }

  return undefined;
}

function getFallbackMessage(error: unknown, fallbackMessage: string): string {
  if (axios.isAxiosError(error) && !error.response) {
    return 'Network error. Please check your connection and try again.';
  }

  return fallbackMessage;
}

export function normalizeApiError(
  error: unknown,
  fallbackMessage = 'Request failed',
): NormalizedApiErrorShape {
  if (error instanceof ApiRequestError) {
    return {
      message: error.message,
      details: error.details,
      requestId: error.requestId,
      status: error.status,
    };
  }

  const payload = getPayload(error);
  const details = normalizeDetails(payload?.details);
  const status = getStatus(error);
  const requestId =
    typeof payload?.request_id === 'string'
      ? payload.request_id
      : typeof payload?.trace_id === 'string'
        ? payload.trace_id
        : undefined;

  const fallback = getFallbackMessage(error, fallbackMessage);

  let message =
    typeof payload?.error === 'string'
      ? payload.error
      : typeof payload?.message === 'string'
        ? payload.message
        : undefined;

  if (message === 'validation_error' && details.length > 0) {
    message = 'Validation failed';
  }

  return {
    message: fallback !== fallbackMessage ? fallback : message ?? fallback,
    details,
    requestId,
    status,
  };
}

export function formatApiErrorMessage(error: NormalizedApiErrorShape): string {
  const detailsText = formatValidationMessage(error.details);
  if (!detailsText) {
    return error.message;
  }

  if (error.message === 'Validation failed') {
    return `${error.message}: ${detailsText}`;
  }

  return `${error.message} (${detailsText})`;
}

export class ApiRequestError extends Error implements NormalizedApiErrorShape {
  details: NormalizedApiErrorDetail[];
  requestId?: string;
  status?: number;

  constructor(normalized: NormalizedApiErrorShape) {
    super(normalized.message);
    this.name = 'ApiRequestError';
    this.details = normalized.details;
    this.requestId = normalized.requestId;
    this.status = normalized.status;
  }

  get traceId(): string | undefined {
    return this.requestId;
  }
}

export function createApiRequestError(
  error: unknown,
  fallbackMessage: string,
  status?: number,
): ApiRequestError {
  const normalized = normalizeApiError(error, fallbackMessage);
  return new ApiRequestError({
    ...normalized,
    status: normalized.status ?? status,
  });
}

const MAX_DISPLAYED_DETAILS = 5;

function buildValidationDescription(details: NormalizedApiErrorDetail[]): string | undefined {
  if (details.length === 0) return undefined;

  const shown = details.slice(0, MAX_DISPLAYED_DETAILS);
  const lines = shown.map((d) => (d.field ? `${d.field}: ${d.message}` : d.message));
  const remaining = details.length - shown.length;
  if (remaining > 0) {
    lines.push(`...and ${remaining} more`);
  }
  return lines.join('\n');
}

export function handleMutationError(error: unknown): void {
  const normalized = normalizeApiError(error);

  // 401s are handled by the auth refresh flow — don't show a toast
  if (normalized.status === 401) return;

  notification.error({
    message: normalized.message,
    description: buildValidationDescription(normalized.details),
    duration: 4.5,
  });
}
