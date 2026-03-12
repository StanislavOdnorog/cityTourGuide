// eslint-disable-next-line import/order
import axios, { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
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

jest.mock('@/api/generated/runtime', () => ({
  createGeneratedApiClient: jest.fn(() => ({})),
}));

import { setOfflineEnqueue, setTokens, setRefreshHandler } from '@/api/client';

const mockAxios = axios as jest.Mocked<typeof axios> & { __instance: unknown };

const responseInterceptorCalls = (mockAxios.create as jest.Mock).mock.results[0]?.value
  ?.interceptors?.response?.use?.mock?.calls?.[0];
const responseErrorHandler = responseInterceptorCalls?.[1] as (
  error: AxiosError,
) => Promise<unknown>;

function makeConfig(
  overrides: Partial<InternalAxiosRequestConfig> = {},
): InternalAxiosRequestConfig {
  const headers = new AxiosHeaders();
  headers.set('Authorization', 'Bearer test-token');
  return {
    headers,
    url: '/api/v1/reports',
    method: 'post',
    ...overrides,
  } as InternalAxiosRequestConfig;
}

function makeNetworkError(config: InternalAxiosRequestConfig): AxiosError {
  const error = new AxiosError('Network Error', 'ERR_NETWORK', config);
  // No response (network error)
  error.response = undefined;
  return error;
}

describe('offline interceptor', () => {
  let mockEnqueue: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    mockEnqueue = jest.fn().mockResolvedValue(true);
    setTokens(null, null);
    setRefreshHandler(null);
    setOfflineEnqueue(mockEnqueue);
  });

  afterEach(() => {
    setOfflineEnqueue(null);
  });

  it('queues network errors for safe POST endpoints', async () => {
    const config = makeConfig({ url: '/api/v1/reports', method: 'post' });
    const error = makeNetworkError(config);

    const result = await responseErrorHandler(error);

    expect(mockEnqueue).toHaveBeenCalledWith({
      endpoint: '/api/v1/reports',
      method: 'post',
      body: undefined,
      headers: { Authorization: 'Bearer test-token' },
    });
    expect(result.statusText).toBe('queued');
  });

  it('queues network errors for /listenings endpoint', async () => {
    const config = makeConfig({ url: '/api/v1/listenings', method: 'post' });
    const error = makeNetworkError(config);

    await responseErrorHandler(error);

    expect(mockEnqueue).toHaveBeenCalledWith(
      expect.objectContaining({ endpoint: '/api/v1/listenings' }),
    );
  });

  it('queues network errors for /device-tokens endpoint', async () => {
    const config = makeConfig({ url: '/api/v1/device-tokens', method: 'post' });
    const error = makeNetworkError(config);

    await responseErrorHandler(error);

    expect(mockEnqueue).toHaveBeenCalledWith(
      expect.objectContaining({ endpoint: '/api/v1/device-tokens' }),
    );
  });

  it('does NOT queue auth endpoints', async () => {
    const config = makeConfig({ url: '/api/v1/auth/refresh', method: 'post' });
    const error = makeNetworkError(config);

    await expect(responseErrorHandler(error)).rejects.toBeDefined();
    expect(mockEnqueue).not.toHaveBeenCalled();
  });

  it('does NOT queue purchase verification', async () => {
    const config = makeConfig({ url: '/api/v1/purchases/verify', method: 'post' });
    const error = makeNetworkError(config);

    await expect(responseErrorHandler(error)).rejects.toBeDefined();
    expect(mockEnqueue).not.toHaveBeenCalled();
  });

  it('does NOT queue GET requests', async () => {
    const config = makeConfig({ url: '/api/v1/reports', method: 'get' });
    const error = makeNetworkError(config);

    await expect(responseErrorHandler(error)).rejects.toBeDefined();
    expect(mockEnqueue).not.toHaveBeenCalled();
  });

  it('does NOT queue when offlineEnqueue is not set', async () => {
    setOfflineEnqueue(null);
    const config = makeConfig({ url: '/api/v1/reports', method: 'post' });
    const error = makeNetworkError(config);

    await expect(responseErrorHandler(error)).rejects.toBeDefined();
  });

  it('does NOT queue server errors (non-network)', async () => {
    const config = makeConfig({ url: '/api/v1/reports', method: 'post' });
    const error = new AxiosError('Server Error', 'ERR_BAD_RESPONSE', config);
    error.response = {
      status: 500,
      data: {},
      headers: {},
      statusText: 'Internal Server Error',
      config,
    } as AxiosError['response'];

    await expect(responseErrorHandler(error)).rejects.toBeDefined();
    expect(mockEnqueue).not.toHaveBeenCalled();
  });

  it('does NOT queue DELETE to /users/me (account deletion)', async () => {
    const config = makeConfig({ url: '/api/v1/users/me', method: 'delete' });
    const error = makeNetworkError(config);

    await expect(responseErrorHandler(error)).rejects.toBeDefined();
    expect(mockEnqueue).not.toHaveBeenCalled();
  });
});
