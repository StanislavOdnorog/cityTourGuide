import { notification } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  ApiRequestError,
  createApiRequestError,
  formatApiErrorMessage,
  handleMutationError,
  normalizeApiError,
} from './errors';

vi.mock('antd', () => ({
  notification: {
    error: vi.fn(),
  },
}));

describe('normalizeApiError', () => {
  it('extracts backend validation details and trace IDs from response data', () => {
    const normalized = normalizeApiError(
      {
        response: {
          status: 400,
          data: {
            error: 'validation_error',
            details: [
              { field: 'name', message: 'is required' },
              { field: 'country', message: 'must be 2 letters' },
            ],
            trace_id: 'trace-123',
          },
        },
      },
      'Failed to save city',
    );

    expect(normalized).toEqual({
      message: 'Validation failed',
      details: [
        { field: 'name', message: 'is required' },
        { field: 'country', message: 'must be 2 letters' },
      ],
      requestId: 'trace-123',
      status: 400,
    });
    expect(formatApiErrorMessage(normalized)).toBe(
      'Validation failed: name: is required; country: must be 2 letters',
    );
  });

  it('accepts request_id as well as trace_id', () => {
    const normalized = normalizeApiError(
      {
        error: 'request rejected',
        request_id: 'req-789',
      },
      'Fallback message',
    );

    expect(normalized.requestId).toBe('req-789');
    expect(normalized.message).toBe('request rejected');
  });

  it('uses a stable network message for transport failures', () => {
    const normalized = normalizeApiError(
      {
        isAxiosError: true,
        message: 'Network Error',
        toJSON: () => ({}),
      },
      'Fallback message',
    );

    expect(normalized.message).toBe('Network error. Please check your connection and try again.');
    expect(normalized.status).toBeUndefined();
  });

  it('preserves normalized ApiRequestError instances', () => {
    const error = createApiRequestError(
      {
        response: {
          status: 422,
          data: {
            error: 'validation_error',
            details: [{ field: 'radius_km', message: 'must be greater than 0' }],
            trace_id: 'trace-456',
          },
        },
      },
      'Failed to update city',
    );

    expect(error).toBeInstanceOf(ApiRequestError);
    expect(normalizeApiError(error)).toEqual({
      message: 'Validation failed',
      details: [{ field: 'radius_km', message: 'must be greater than 0' }],
      requestId: 'trace-456',
      status: 422,
    });
  });
});

describe('handleMutationError', () => {
  const notificationError = vi.mocked(notification.error);

  beforeEach(() => {
    notificationError.mockClear();
  });

  it('shows a notification for a generic backend error', () => {
    const error = new ApiRequestError({
      message: 'Failed to create city',
      details: [],
      status: 500,
    });

    handleMutationError(error);

    expect(notificationError).toHaveBeenCalledWith({
      message: 'Failed to create city',
      description: undefined,
      duration: 4.5,
    });
  });

  it('shows field-level validation details in the notification', () => {
    const error = new ApiRequestError({
      message: 'Validation failed',
      details: [
        { field: 'name', message: 'is required' },
        { field: 'country', message: 'must be 2 letters' },
      ],
      status: 400,
    });

    handleMutationError(error);

    expect(notificationError).toHaveBeenCalledWith({
      message: 'Validation failed',
      description: 'name: is required\ncountry: must be 2 letters',
      duration: 4.5,
    });
  });

  it('truncates validation details beyond 5 items', () => {
    const details = Array.from({ length: 8 }, (_, i) => ({
      field: `field_${i}`,
      message: `error ${i}`,
    }));
    const error = new ApiRequestError({
      message: 'Validation failed',
      details,
      status: 400,
    });

    handleMutationError(error);

    const call = notificationError.mock.calls[0][0];
    expect(call.description).toContain('field_4: error 4');
    expect(call.description).toContain('...and 3 more');
    expect(call.description).not.toContain('field_5');
  });

  it('does not show a notification for 401 errors', () => {
    const error = new ApiRequestError({
      message: 'Unauthorized',
      details: [],
      status: 401,
    });

    handleMutationError(error);

    expect(notificationError).not.toHaveBeenCalled();
  });

  it('shows a friendly message for network errors', () => {
    handleMutationError({
      isAxiosError: true,
      message: 'Network Error',
      toJSON: () => ({}),
    });

    expect(notificationError).toHaveBeenCalledWith(
      expect.objectContaining({
        message: 'Network error. Please check your connection and try again.',
      }),
    );
  });
});
