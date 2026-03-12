import { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
import {
  AppApiError,
  isAppApiError,
  normalizeError,
  normalizeGeneratedError,
  userMessageForError,
} from '../errors';

function makeAxiosError(opts: {
  status?: number;
  code?: string;
  message?: string;
  data?: unknown;
}): AxiosError {
  const config = { headers: new AxiosHeaders() } as InternalAxiosRequestConfig;
  const response = opts.status
    ? {
        status: opts.status,
        statusText: '',
        headers: {},
        config,
        data: opts.data ?? {},
      }
    : undefined;

  return {
    isAxiosError: true,
    config,
    response,
    code: opts.code,
    message: opts.message ?? 'error',
    name: 'AxiosError',
    toJSON: () => ({}),
  } as AxiosError;
}

describe('normalizeError', () => {
  it('returns the same AppApiError if already normalized', () => {
    const original = new AppApiError({ category: 'server', message: 'test' });
    expect(normalizeError(original)).toBe(original);
  });

  describe('network errors', () => {
    it('categorizes ERR_NETWORK as network', () => {
      const err = makeAxiosError({ code: 'ERR_NETWORK', message: 'Network Error' });
      const result = normalizeError(err);
      expect(result).toBeInstanceOf(AppApiError);
      expect(result.category).toBe('network');
      expect(result.status).toBeUndefined();
    });

    it('categorizes ECONNABORTED as network', () => {
      const err = makeAxiosError({ code: 'ECONNABORTED', message: 'timeout' });
      const result = normalizeError(err);
      expect(result.category).toBe('network');
    });

    it('categorizes "Network Error" message as network', () => {
      const err = makeAxiosError({ message: 'Network Error' });
      const result = normalizeError(err);
      expect(result.category).toBe('network');
    });

    it('does not categorize ERR_CANCELED as network', () => {
      const err = makeAxiosError({ code: 'ERR_CANCELED', message: 'canceled' });
      const result = normalizeError(err);
      expect(result.category).toBe('unknown');
    });
  });

  describe('401 unauthorized', () => {
    it('categorizes 401 as unauthorized', () => {
      const err = makeAxiosError({ status: 401 });
      const result = normalizeError(err);
      expect(result.category).toBe('unauthorized');
      expect(result.status).toBe(401);
    });

    it('extracts trace_id from 401 response', () => {
      const err = makeAxiosError({
        status: 401,
        data: { error: 'token expired', trace_id: 'tr-123' },
      });
      const result = normalizeError(err);
      expect(result.category).toBe('unauthorized');
      expect(result.traceId).toBe('tr-123');
      expect(result.serverMessage).toBe('token expired');
    });
  });

  describe('validation errors (400/422)', () => {
    it('categorizes 400 as validation', () => {
      const err = makeAxiosError({
        status: 400,
        data: { error: 'invalid lat value' },
      });
      const result = normalizeError(err);
      expect(result.category).toBe('validation');
      expect(result.status).toBe(400);
      expect(result.message).toBe('invalid lat value');
    });

    it('categorizes 422 as validation', () => {
      const err = makeAxiosError({
        status: 422,
        data: { error: 'missing required field' },
      });
      const result = normalizeError(err);
      expect(result.category).toBe('validation');
      expect(result.status).toBe(422);
      expect(result.message).toBe('missing required field');
    });

    it('uses fallback message when server message is missing', () => {
      const err = makeAxiosError({ status: 400 });
      const result = normalizeError(err);
      expect(result.category).toBe('validation');
      expect(result.message).toBe('The request was invalid. Please check your input.');
    });
  });

  describe('5xx server errors', () => {
    it('categorizes 500 as server', () => {
      const err = makeAxiosError({ status: 500 });
      const result = normalizeError(err);
      expect(result.category).toBe('server');
      expect(result.status).toBe(500);
    });

    it('categorizes 502 as server', () => {
      const err = makeAxiosError({ status: 502 });
      const result = normalizeError(err);
      expect(result.category).toBe('server');
    });

    it('categorizes 503 as server', () => {
      const err = makeAxiosError({ status: 503 });
      const result = normalizeError(err);
      expect(result.category).toBe('server');
    });

    it('extracts trace_id from server error', () => {
      const err = makeAxiosError({
        status: 500,
        data: { error: 'internal', trace_id: 'srv-456' },
      });
      const result = normalizeError(err);
      expect(result.traceId).toBe('srv-456');
      expect(result.serverMessage).toBe('internal');
    });
  });

  describe('other errors', () => {
    it('categorizes non-axios Error as unknown', () => {
      const err = new Error('something broke');
      const result = normalizeError(err);
      expect(result.category).toBe('unknown');
      expect(result.message).toBe('something broke');
    });

    it('handles non-Error values', () => {
      const result = normalizeError('string error');
      expect(result.category).toBe('unknown');
      expect(result.message).toBe('An unexpected error occurred');
    });

    it('handles null', () => {
      const result = normalizeError(null);
      expect(result.category).toBe('unknown');
    });

    it('categorizes 404 as unknown', () => {
      const err = makeAxiosError({ status: 404 });
      const result = normalizeError(err);
      expect(result.category).toBe('unknown');
      expect(result.status).toBe(404);
    });
  });
});

describe('normalizeGeneratedError', () => {
  it('creates AppApiError from backend error body', () => {
    const result = normalizeGeneratedError({ error: 'bad input', trace_id: 'gen-1' }, 'fallback');
    expect(result).toBeInstanceOf(AppApiError);
    expect(result.message).toBe('bad input');
    expect(result.traceId).toBe('gen-1');
  });

  it('uses fallback when error field is missing', () => {
    const result = normalizeGeneratedError({ trace_id: 'gen-2' }, 'Request failed');
    expect(result.message).toBe('Request failed');
    expect(result.traceId).toBe('gen-2');
  });

  it('maps status to correct category', () => {
    expect(normalizeGeneratedError({}, 'f', 401).category).toBe('unauthorized');
    expect(normalizeGeneratedError({}, 'f', 400).category).toBe('validation');
    expect(normalizeGeneratedError({}, 'f', 422).category).toBe('validation');
    expect(normalizeGeneratedError({}, 'f', 500).category).toBe('server');
    expect(normalizeGeneratedError({}, 'f', 404).category).toBe('unknown');
    expect(normalizeGeneratedError({}, 'f').category).toBe('unknown');
  });

  it('uses server message for validation errors', () => {
    const result = normalizeGeneratedError(
      { error: 'lat must be between -90 and 90' },
      'fallback',
      422,
    );
    expect(result.message).toBe('lat must be between -90 and 90');
  });
});

describe('isAppApiError', () => {
  it('returns true for AppApiError', () => {
    expect(isAppApiError(new AppApiError({ category: 'network', message: 'x' }))).toBe(true);
  });

  it('returns false for plain Error', () => {
    expect(isAppApiError(new Error('x'))).toBe(false);
  });

  it('returns false for non-errors', () => {
    expect(isAppApiError(null)).toBe(false);
    expect(isAppApiError('string')).toBe(false);
  });
});

describe('userMessageForError', () => {
  it('returns category-specific message for network errors', () => {
    const err = new AppApiError({ category: 'network', message: 'x' });
    expect(userMessageForError(err)).toContain('internet connection');
  });

  it('returns category-specific message for unauthorized errors', () => {
    const err = new AppApiError({ category: 'unauthorized', message: 'x' });
    expect(userMessageForError(err)).toContain('session');
  });

  it('returns server message for validation errors', () => {
    const err = new AppApiError({
      category: 'validation',
      message: 'bad lat',
      serverMessage: 'lat must be valid',
    });
    expect(userMessageForError(err)).toBe('lat must be valid');
  });

  it('returns fallback for validation errors without server message', () => {
    const err = new AppApiError({ category: 'validation', message: 'bad input' });
    expect(userMessageForError(err)).toContain('invalid');
  });

  it('returns generic message for server errors', () => {
    const err = new AppApiError({ category: 'server', message: 'x' });
    expect(userMessageForError(err)).toContain('our end');
  });

  it('returns generic message for non-AppApiError', () => {
    expect(userMessageForError(new Error('boom'))).toContain('unexpected');
  });
});
