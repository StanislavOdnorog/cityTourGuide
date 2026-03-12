import axios, { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
// eslint-disable-next-line import/order
import { DEFAULT_RETRY_ATTEMPTS } from '../retry';
jest.mock('axios', () => {
  const actualAxios = jest.requireActual('axios');
  const instance = {
    interceptors: {
      request: { use: jest.fn(), eject: jest.fn(), clear: jest.fn() },
      response: { use: jest.fn(), eject: jest.fn(), clear: jest.fn() },
    },
    defaults: { headers: { common: {} } },
    request: jest.fn(),
    get: jest.fn(),
    post: jest.fn(),
  } as unknown;

  // Make the instance callable (for retries that call apiClient(config))
  const callable = jest.fn((...args: unknown[]) =>
    (instance as Record<string, jest.Mock>).request(...args),
  );
  Object.assign(callable, instance);

  const create = jest.fn(() => callable);

  return {
    ...actualAxios,
    default: { ...actualAxios.default, create },
    create,
    __instance: callable,
  };
});

jest.mock('../generated/runtime', () => ({
  createGeneratedApiClient: jest.fn(() => ({})),
}));

// eslint-disable-next-line import/order
import apiClient, { setTokens, setRefreshHandler } from '../client';

const mockAxios = axios as jest.Mocked<typeof axios> & { __instance: unknown };

// Grab the interceptor callbacks registered during import
const requestInterceptor = (mockAxios.create as jest.Mock).mock.results[0]?.value?.interceptors
  ?.request?.use?.mock?.calls?.[0]?.[0];
const responseInterceptorCalls = (mockAxios.create as jest.Mock).mock.results[0]?.value
  ?.interceptors?.response?.use?.mock?.calls?.[0];
const responseSuccessHandler = responseInterceptorCalls?.[0];
const responseErrorHandler = responseInterceptorCalls?.[1];

function makeConfig(
  overrides: Partial<InternalAxiosRequestConfig> = {},
): InternalAxiosRequestConfig {
  return {
    headers: new AxiosHeaders(),
    method: 'get',
    ...overrides,
  } as InternalAxiosRequestConfig;
}

function makeRetryConfig(
  overrides: Partial<InternalAxiosRequestConfig> & Record<string, unknown>,
): InternalAxiosRequestConfig {
  return makeConfig(overrides as Partial<InternalAxiosRequestConfig>);
}

function make401Error(config?: InternalAxiosRequestConfig): AxiosError {
  const cfg = config ?? makeConfig();
  return {
    isAxiosError: true,
    config: cfg,
    response: {
      status: 401,
      statusText: 'Unauthorized',
      headers: {},
      config: cfg,
      data: {},
    },
    message: 'Unauthorized',
    name: 'AxiosError',
    toJSON: () => ({}),
  } as AxiosError;
}

function make429Error(
  retryAfter: string | undefined,
  config?: InternalAxiosRequestConfig,
): AxiosError {
  const cfg = config ?? makeConfig();
  return {
    isAxiosError: true,
    config: cfg,
    response: {
      status: 429,
      statusText: 'Too Many Requests',
      headers: retryAfter !== undefined ? { 'retry-after': retryAfter } : {},
      config: cfg,
      data: {},
    },
    message: 'Too Many Requests',
    name: 'AxiosError',
    toJSON: () => ({}),
  } as AxiosError;
}

describe('API client', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.spyOn(Math, 'random').mockReturnValue(0);
    // Reset module-level state
    setTokens(null, null);
    setRefreshHandler((() => Promise.resolve(null)) as unknown as Parameters<
      typeof setRefreshHandler
    >[0]);
    // Clear mock call history
    (apiClient as unknown as jest.Mock).mockReset();
    (mockAxios.__instance.request as jest.Mock).mockReset();
  });

  afterEach(() => {
    jest.restoreAllMocks();
    jest.useRealTimers();
  });

  describe('request interceptor – token attachment', () => {
    it('attaches Bearer token when accessToken is set', () => {
      setTokens('my-access-token', 'my-refresh-token');
      const config = makeConfig();
      const result = requestInterceptor(config);
      expect(result.headers.Authorization).toBe('Bearer my-access-token');
    });

    it('does not attach Authorization header when no token is set', () => {
      setTokens(null, null);
      const config = makeConfig();
      const result = requestInterceptor(config);
      expect(result.headers.Authorization).toBeUndefined();
    });
  });

  describe('response interceptor – 401 refresh', () => {
    it('refreshes token and retries on 401', async () => {
      setTokens('expired-token', 'valid-refresh');
      const refreshHandler = jest.fn().mockResolvedValue('new-access-token');
      setRefreshHandler(refreshHandler);

      const fakeResponse = { data: 'success' };
      (apiClient as unknown as jest.Mock).mockResolvedValueOnce(fakeResponse);

      const error = make401Error();
      const result = await responseErrorHandler(error);

      expect(refreshHandler).toHaveBeenCalledWith('valid-refresh');
      expect(result).toEqual(fakeResponse);
      // Verify the retried request has the new token
      const retriedConfig = (apiClient as unknown as jest.Mock).mock.calls[0][0];
      expect(retriedConfig.headers.Authorization).toBe('Bearer new-access-token');
      expect(retriedConfig._skip429Retry).toBe(true);
    });

    it('queues concurrent 401s and resolves them all after one refresh', async () => {
      setTokens('expired', 'refresh-tok');

      let resolveRefresh!: (value: string | null) => void;
      const refreshHandler = jest.fn(() => new Promise<string | null>((r) => (resolveRefresh = r)));
      setRefreshHandler(refreshHandler);

      const fakeResponse1 = { data: 'resp1' };
      const fakeResponse2 = { data: 'resp2' };
      const fakeResponse3 = { data: 'resp3' };

      // First call triggers refresh; second and third queue up
      (apiClient as unknown as jest.Mock)
        .mockResolvedValueOnce(fakeResponse1) // retry of first request
        .mockResolvedValueOnce(fakeResponse2) // replay of second queued request
        .mockResolvedValueOnce(fakeResponse3); // replay of third queued request

      const error1 = make401Error(makeConfig());
      const error2 = make401Error(makeConfig());
      const error3 = make401Error(makeConfig());

      // Kick off three 401 errors concurrently
      const p1 = responseErrorHandler(error1);
      const p2 = responseErrorHandler(error2);
      const p3 = responseErrorHandler(error3);

      // Only one refresh call should have been made
      expect(refreshHandler).toHaveBeenCalledTimes(1);

      // Resolve the refresh
      resolveRefresh('fresh-token');

      const results = await Promise.all([p1, p2, p3]);
      // All three should resolve successfully (order depends on internal scheduling)
      const dataValues = results.map((r: { data: string }) => r.data).sort();
      expect(dataValues).toEqual(['resp1', 'resp2', 'resp3']);
      // Only one refresh call should have been made
      expect(refreshHandler).toHaveBeenCalledTimes(1);
    });

    it('rejects all queued requests when refresh fails', async () => {
      setTokens('expired', 'refresh-tok');

      const refreshError = new Error('refresh failed');
      const refreshHandler = jest.fn().mockRejectedValue(refreshError);
      setRefreshHandler(refreshHandler);

      const error1 = make401Error(makeConfig());
      const error2 = make401Error(makeConfig());

      const p1 = responseErrorHandler(error1);
      const p2 = responseErrorHandler(error2);

      await expect(p1).rejects.toThrow('refresh failed');
      await expect(p2).rejects.toThrow('refresh failed');
    });

    it('does not retry when refresh returns null', async () => {
      setTokens('expired', 'refresh-tok');
      const refreshHandler = jest.fn().mockResolvedValue(null);
      setRefreshHandler(refreshHandler);

      const error = make401Error();
      // When newToken is null, the code falls through without retrying,
      // then hits the final Promise.reject(error)
      await expect(responseErrorHandler(error)).rejects.toBe(error);
      expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();
    });

    it('does not attempt refresh without refreshToken', async () => {
      setTokens('expired', null);
      const refreshHandler = jest.fn();
      setRefreshHandler(refreshHandler);

      const error = make401Error();
      await expect(responseErrorHandler(error)).rejects.toBeDefined();
      expect(refreshHandler).not.toHaveBeenCalled();
    });

    it('does not attempt refresh without refresh handler', async () => {
      setTokens('expired', 'refresh-tok');
      // setRefreshHandler was reset in beforeEach but let's be explicit
      setRefreshHandler(null as unknown as Parameters<typeof setRefreshHandler>[0]);

      const error = make401Error();
      await expect(responseErrorHandler(error)).rejects.toBeDefined();
    });

    it('does not replay queued requests with Bearer null', async () => {
      setTokens('expired', 'refresh-tok');

      let resolveRefresh!: (value: string | null) => void;
      const refreshHandler = jest.fn(() => new Promise<string | null>((r) => (resolveRefresh = r)));
      setRefreshHandler(refreshHandler);

      const fakeResponse = { data: 'ok' };
      (apiClient as unknown as jest.Mock).mockResolvedValue(fakeResponse);

      const error1 = make401Error(makeConfig());
      const error2 = make401Error(makeConfig());

      const p1 = responseErrorHandler(error1);
      const p2 = responseErrorHandler(error2);

      // Resolve with a valid token
      resolveRefresh('valid-token');

      await Promise.all([p1, p2]);

      // Check the queued request's replayed config
      const calls = (apiClient as unknown as jest.Mock).mock.calls;
      for (const call of calls) {
        const config = call[0];
        expect(config.headers.Authorization).not.toBe('Bearer null');
        expect(config.headers.Authorization).toBe('Bearer valid-token');
      }
    });
  });

  describe('response interceptor – 429 retry', () => {
    it.each([
      ['get', undefined, 1000],
      ['head', undefined, 1000],
      ['get', '3', 3000],
      ['get', 'not-a-number', 1000],
    ])(
      'retries safe %s requests with a bounded delay when Retry-After is %s',
      async (method, retryAfter, expectedDelayMs) => {
        const fakeResponse = { data: 'ok' };
        (apiClient as unknown as jest.Mock).mockResolvedValueOnce(fakeResponse);

        const error = make429Error(retryAfter, makeConfig({ method }));

        const promise = responseErrorHandler(error);

        expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();

        jest.advanceTimersByTime(expectedDelayMs - 1);
        await Promise.resolve();
        expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();

        jest.advanceTimersByTime(1);

        const result = await promise;
        expect(result).toEqual(fakeResponse);
        expect(apiClient as unknown as jest.Mock).toHaveBeenCalledTimes(1);
        expect((apiClient as unknown as jest.Mock).mock.calls[0][0]._retryCount).toBe(1);
      },
    );

    it('does not retry non-idempotent requests without explicit opt-in', async () => {
      const error = make429Error(undefined, makeConfig({ method: 'post' }));

      await expect(responseErrorHandler(error)).rejects.toBe(error);
      expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();
    });

    it('retries mutation requests only when explicitly opted in', async () => {
      const fakeResponse = { data: 'ok' };
      (apiClient as unknown as jest.Mock).mockResolvedValueOnce(fakeResponse);

      const error = make429Error(undefined, makeRetryConfig({ method: 'post', retryOn429: true }));

      const promise = responseErrorHandler(error);
      jest.advanceTimersByTime(1000);

      await expect(promise).resolves.toEqual(fakeResponse);
      expect(apiClient as unknown as jest.Mock).toHaveBeenCalledTimes(1);
    });

    it('stops retrying after the configured maximum attempts', async () => {
      const error = make429Error(
        undefined,
        makeRetryConfig({
          method: 'get',
          _retryCount: DEFAULT_RETRY_ATTEMPTS,
        }),
      );

      await expect(responseErrorHandler(error)).rejects.toBe(error);
      expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();
    });

    it('does not retry a request that already replayed after refresh', async () => {
      const error = make429Error(
        undefined,
        makeRetryConfig({ method: 'get', _retry: true, _skip429Retry: true }),
      );

      await expect(responseErrorHandler(error)).rejects.toBe(error);
      expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();
    });

    it('does not retry cancelled requests', async () => {
      const error = make429Error(undefined, makeConfig({ method: 'get' }));
      error.code = 'ERR_CANCELED';

      await expect(responseErrorHandler(error)).rejects.toBe(error);
      expect(apiClient as unknown as jest.Mock).not.toHaveBeenCalled();
    });
  });

  describe('response interceptor – other errors', () => {
    it('rejects non-401/429 errors', async () => {
      const error: AxiosError = {
        isAxiosError: true,
        config: makeConfig(),
        response: {
          status: 500,
          statusText: 'Internal Server Error',
          headers: {},
          config: makeConfig(),
          data: {},
        },
        message: 'Server Error',
        name: 'AxiosError',
        toJSON: () => ({}),
      } as AxiosError;

      await expect(responseErrorHandler(error)).rejects.toBe(error);
    });

    it('passes through successful responses', () => {
      const response = { data: 'ok', status: 200 };
      expect(responseSuccessHandler(response)).toBe(response);
    });
  });
});
