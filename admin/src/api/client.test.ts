import { describe, expect, it, vi } from 'vitest';
import { API_BASE_URL } from '../constants';
import type { User } from '../types';

const { generatedClient } = vi.hoisted(() => ({
  generatedClient: {
    POST: vi.fn(),
  },
}));

vi.mock('./generated/runtime', () => ({
  createGeneratedApiClient: vi.fn(() => generatedClient),
}));

vi.mock('axios', () => {
  const requestHandlers: Array<(config: Record<string, unknown>) => unknown> = [];
  const responseErrorHandlers: Array<(error: Record<string, unknown>) => unknown> = [];
  const apiInstance = vi.fn() as ReturnType<typeof vi.fn> & {
    interceptors: {
      request: {
        use: ReturnType<typeof vi.fn>;
      };
      response: {
        use: ReturnType<typeof vi.fn>;
      };
    };
  };

  apiInstance.interceptors = {
    request: {
      use: vi.fn((fulfilled: (config: Record<string, unknown>) => unknown) => {
        requestHandlers.push(fulfilled);
        return requestHandlers.length - 1;
      }),
    },
    response: {
      use: vi.fn(
        (_fulfilled: unknown, rejected: (error: Record<string, unknown>) => unknown) => {
          responseErrorHandlers.push(rejected);
          return responseErrorHandlers.length - 1;
        },
      ),
    },
  };

  const axiosMock = Object.assign(vi.fn(), {
    create: vi.fn(() => apiInstance),
    post: vi.fn(),
    isAxiosError: vi.fn((error: unknown) => Boolean((error as { isAxiosError?: boolean })?.isAxiosError)),
    __apiInstance: apiInstance,
    __requestHandlers: requestHandlers,
    __responseErrorHandlers: responseErrorHandlers,
  });

  return {
    default: axiosMock,
  };
});

type AxiosMock = {
  create: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
  isAxiosError: ReturnType<typeof vi.fn>;
  __apiInstance: ReturnType<typeof vi.fn>;
  __requestHandlers: Array<(config: Record<string, unknown>) => unknown>;
  __responseErrorHandlers: Array<(error: Record<string, unknown>) => unknown>;
};

const user: User = {
  id: '550e8400-e29b-41d4-a716-446655440000',
  email: 'admin@example.com',
  name: 'Admin User',
  auth_provider: 'email',
  provider_id: null,
  language_pref: 'en',
  is_anonymous: false,
  is_admin: true,
  deleted_at: null,
  deletion_scheduled_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

async function loadClientModule() {
  vi.resetModules();

  const { default: axiosModule } = await import('axios');
  const axiosMock = axiosModule as unknown as AxiosMock;
  axiosMock.__requestHandlers.length = 0;
  axiosMock.__responseErrorHandlers.length = 0;
  axiosMock.__apiInstance.mockReset();
  axiosMock.post.mockReset();
  axiosMock.create.mockReset();
  axiosMock.create.mockReturnValue(axiosMock.__apiInstance);
  generatedClient.POST.mockReset();

  const [clientModule, authStoreModule] = await Promise.all([
    import('./client'),
    import('../store/authStore'),
  ]);

  return {
    axiosMock,
    useAuthStore: authStoreModule.useAuthStore,
    clientModule,
  };
}

function make401Error(url: string, extra?: Record<string, unknown>) {
  return {
    config: { url, headers: {}, ...extra },
    response: { status: 401 },
  };
}

describe('api client interceptors', () => {
  it('attaches the bearer token to requests', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('access-token', 'refresh-token', user);

    const requestHandler = axiosMock.__requestHandlers[0];
    const config = requestHandler({ headers: {} }) as { headers: Record<string, string> };

    expect(config.headers.Authorization).toBe('Bearer access-token');
  });

  it('refreshes tokens after a 401 and retries the original request', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('expired-access', 'refresh-token', user);
    axiosMock.post.mockResolvedValue({
      data: {
        tokens: {
          access_token: 'new-access',
          refresh_token: 'new-refresh',
        },
      },
    });
    axiosMock.__apiInstance.mockResolvedValue({ data: { ok: true } });

    const responseErrorHandler = axiosMock.__responseErrorHandlers[0];
    const result = await responseErrorHandler(make401Error('/admin/cities'));

    expect(axiosMock.post).toHaveBeenCalledWith(`${API_BASE_URL}/api/v1/auth/refresh`, {
      refresh_token: 'refresh-token',
    });
    expect(useAuthStore.getState()).toMatchObject({
      token: 'new-access',
      refreshToken: 'new-refresh',
      isAuthenticated: true,
    });
    expect(axiosMock.__apiInstance).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer new-access',
        }),
      }),
    );
    expect(result).toEqual({ data: { ok: true } });
  });

  it('logs out when token refresh fails', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('expired-access', 'refresh-token', user);
    axiosMock.post.mockRejectedValue(new Error('refresh failed'));

    const responseErrorHandler = axiosMock.__responseErrorHandlers[0];

    await expect(
      responseErrorHandler(make401Error('/admin/cities')),
    ).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(useAuthStore.getState()).toMatchObject({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
    });
    expect(localStorage.getItem('csg_admin_token')).toBeNull();
    expect(localStorage.getItem('csg_admin_refresh')).toBeNull();
    expect(localStorage.getItem('csg_admin_user')).toBeNull();
  });

  it('concurrent 401s share a single refresh request', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('expired-access', 'refresh-token', user);

    // Use a deferred promise so we can control when refresh resolves
    let resolveRefresh!: (value: unknown) => void;
    axiosMock.post.mockReturnValue(
      new Promise((resolve) => {
        resolveRefresh = resolve;
      }),
    );
    axiosMock.__apiInstance.mockResolvedValue({ data: { ok: true } });

    const handler = axiosMock.__responseErrorHandlers[0];

    // Fire three concurrent 401 errors
    const p1 = handler(make401Error('/admin/cities'));
    const p2 = handler(make401Error('/admin/pois'));
    const p3 = handler(make401Error('/admin/stories'));

    // Resolve the single refresh call
    resolveRefresh({
      data: {
        tokens: {
          access_token: 'new-access',
          refresh_token: 'new-refresh',
        },
      },
    });

    const [r1, r2, r3] = await Promise.all([p1, p2, p3]);

    // Only ONE refresh call was made
    expect(axiosMock.post).toHaveBeenCalledTimes(1);

    // All three requests resolved
    expect(r1).toEqual({ data: { ok: true } });
    expect(r2).toEqual({ data: { ok: true } });
    expect(r3).toEqual({ data: { ok: true } });

    // All retried requests used the new token
    const retryCallArgs = axiosMock.__apiInstance.mock.calls;
    expect(retryCallArgs).toHaveLength(3);
    for (const [config] of retryCallArgs) {
      expect(config.headers.Authorization).toBe('Bearer new-access');
    }
  });

  it('queued requests resume with the refreshed access token', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('expired-access', 'refresh-token', user);

    let resolveRefresh!: (value: unknown) => void;
    axiosMock.post.mockReturnValue(
      new Promise((resolve) => {
        resolveRefresh = resolve;
      }),
    );
    // Track the retry config for each call
    axiosMock.__apiInstance.mockImplementation((config: Record<string, unknown>) => {
      return Promise.resolve({ data: { retried: true, url: config.url } });
    });

    const handler = axiosMock.__responseErrorHandlers[0];

    // First request triggers refresh, second queues
    const p1 = handler(make401Error('/admin/cities'));
    const p2 = handler(make401Error('/admin/pois'));

    resolveRefresh({
      data: {
        tokens: {
          access_token: 'fresh-token',
          refresh_token: 'fresh-refresh',
        },
      },
    });

    const [r1, r2] = await Promise.all([p1, p2]);

    // Both retried with the new token
    expect(r1).toEqual({ data: { retried: true, url: '/admin/cities' } });
    expect(r2).toEqual({ data: { retried: true, url: '/admin/pois' } });

    // Verify both retry configs carry the new token
    for (const [config] of axiosMock.__apiInstance.mock.calls) {
      expect(config.headers.Authorization).toBe('Bearer fresh-token');
    }
  });

  it.each([
    '/auth/login',
    '/auth/refresh',
    '/api/v1/auth/login',
    '/api/v1/auth/refresh',
  ])('does not trigger refresh for 401 on %s', async (url) => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('access-token', 'refresh-token', user);

    const handler = axiosMock.__responseErrorHandlers[0];

    await expect(handler(make401Error(url))).rejects.toMatchObject({
      response: { status: 401 },
    });

    // No refresh call was attempted
    expect(axiosMock.post).not.toHaveBeenCalled();

    // Auth state preserved (no logout for auth-endpoint 401s)
    expect(useAuthStore.getState().token).toBe('access-token');
  });

  it('logs out immediately when no refresh token is present', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    // Set auth but with no refresh token — simulate partial state
    useAuthStore.setState({
      token: 'access-token',
      refreshToken: null,
      user,
      isAuthenticated: true,
    });

    const handler = axiosMock.__responseErrorHandlers[0];

    await expect(handler(make401Error('/admin/cities'))).rejects.toMatchObject({
      response: { status: 401 },
    });

    // No refresh call was attempted
    expect(axiosMock.post).not.toHaveBeenCalled();

    // Auth state was cleared
    expect(useAuthStore.getState()).toMatchObject({
      token: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
    });
  });

  it('stale refresh does not restore auth after a logout race', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('expired-access', 'refresh-token', user);

    let resolveRefresh!: (value: unknown) => void;
    axiosMock.post.mockReturnValue(
      new Promise((resolve) => {
        resolveRefresh = resolve;
      }),
    );
    axiosMock.__apiInstance.mockResolvedValue({ data: { ok: true } });

    const handler = axiosMock.__responseErrorHandlers[0];

    // Trigger a 401 which starts the refresh
    const p = handler(make401Error('/admin/cities'));

    // User logs out while refresh is in flight
    useAuthStore.getState().logout();

    // Now the refresh "succeeds" with stale tokens
    resolveRefresh({
      data: {
        tokens: {
          access_token: 'stale-new-access',
          refresh_token: 'stale-new-refresh',
        },
      },
    });

    // The request still completes (retried with new token)
    await p;

    // Auth store must remain logged out — stale refresh must NOT restore tokens
    expect(useAuthStore.getState()).toMatchObject({
      token: null,
      refreshToken: null,
      isAuthenticated: false,
    });
  });

  it('queued requests reject when logout happens during refresh', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('expired-access', 'refresh-token', user);

    let resolveRefresh!: (value: unknown) => void;
    axiosMock.post.mockReturnValue(
      new Promise((resolve) => {
        resolveRefresh = resolve;
      }),
    );
    axiosMock.__apiInstance.mockResolvedValue({ data: { ok: true } });

    const handler = axiosMock.__responseErrorHandlers[0];

    // First request triggers refresh, second queues
    const p1 = handler(make401Error('/admin/cities'));
    const p2 = handler(make401Error('/admin/pois'));

    // Logout while refresh is in flight
    useAuthStore.getState().logout();

    resolveRefresh({
      data: {
        tokens: {
          access_token: 'stale-access',
          refresh_token: 'stale-refresh',
        },
      },
    });

    // First request goes through (it does its own retry), but queued request
    // should detect the logout and reject
    await p1; // leader still retries (interceptor already has the new token)

    await expect(p2).rejects.toMatchObject({
      response: { status: 401 },
    });

    // Auth must stay logged out
    expect(useAuthStore.getState().token).toBeNull();
  });

  it('does not attempt refresh when already logged out', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    // Start with no auth
    expect(useAuthStore.getState().token).toBeNull();

    const handler = axiosMock.__responseErrorHandlers[0];

    await expect(handler(make401Error('/admin/cities'))).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(axiosMock.post).not.toHaveBeenCalled();
  });

  it('does not retry a request that already has _retry set', async () => {
    const { axiosMock, useAuthStore } = await loadClientModule();
    useAuthStore.getState().setAuth('access-token', 'refresh-token', user);

    const handler = axiosMock.__responseErrorHandlers[0];

    await expect(
      handler(make401Error('/admin/cities', { _retry: true })),
    ).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(axiosMock.post).not.toHaveBeenCalled();
  });

  it('error wrapping interceptor converts raw errors to ApiRequestError', async () => {
    const { axiosMock } = await loadClientModule();
    const { ApiRequestError } = await import('./errors');

    // The second response error handler is the error-wrapping interceptor
    const errorWrapHandler = axiosMock.__responseErrorHandlers[1];
    expect(errorWrapHandler).toBeDefined();

    const rawError = {
      response: {
        status: 422,
        data: {
          error: 'validation_error',
          details: [{ field: 'name', message: 'is required' }],
          trace_id: 'trace-wrap-1',
        },
      },
    };

    await expect(errorWrapHandler(rawError)).rejects.toBeInstanceOf(ApiRequestError);
    await expect(errorWrapHandler(rawError)).rejects.toMatchObject({
      message: 'Validation failed',
      details: [{ field: 'name', message: 'is required' }],
      status: 422,
    });
  });

  it('error wrapping interceptor passes through existing ApiRequestError instances', async () => {
    const { axiosMock } = await loadClientModule();
    const { ApiRequestError } = await import('./errors');

    const errorWrapHandler = axiosMock.__responseErrorHandlers[1];
    const existing = new ApiRequestError({
      message: 'Already wrapped',
      details: [],
      status: 500,
    });

    await expect(errorWrapHandler(existing)).rejects.toBe(existing);
  });

  it('error wrapping interceptor shows friendly message for network errors', async () => {
    const { axiosMock } = await loadClientModule();

    const errorWrapHandler = axiosMock.__responseErrorHandlers[1];
    const networkError = {
      isAxiosError: true,
      message: 'Network Error',
      toJSON: () => ({}),
    };

    await expect(errorWrapHandler(networkError)).rejects.toMatchObject({
      message: 'Network error. Please check your connection and try again.',
    });
  });

  it('login throws a normalized ApiRequestError for backend validation failures', async () => {
    const { clientModule } = await loadClientModule();
    generatedClient.POST.mockResolvedValueOnce({
      data: undefined,
      error: {
        error: 'validation_error',
        details: [{ field: 'email', message: 'must be a valid email address' }],
        trace_id: 'login-trace-1',
      },
      response: { status: 400 },
    });

    await expect(clientModule.login('bad-email', 'secret')).rejects.toMatchObject({
      name: 'ApiRequestError',
      message: 'Validation failed',
      details: [{ field: 'email', message: 'must be a valid email address' }],
      requestId: 'login-trace-1',
      status: 400,
    });
  });

  it('login throws a generic normalized message when tokens are missing from a success response', async () => {
    const { clientModule } = await loadClientModule();
    generatedClient.POST.mockResolvedValueOnce({
      data: { data: user },
      error: undefined,
      response: { status: 200 },
    });

    await expect(clientModule.login('admin@example.com', 'secret')).rejects.toThrow(
      'Login response is missing tokens',
    );
  });
});
