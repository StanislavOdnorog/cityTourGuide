import { describe, expect, it, vi } from 'vitest';

const { mockGET, mockPOST, mockPUT, mockDELETE } = vi.hoisted(() => ({
  mockGET: vi.fn(),
  mockPOST: vi.fn(),
  mockPUT: vi.fn(),
  mockDELETE: vi.fn(),
}));

vi.mock('./client', () => ({
  generatedApiClient: {
    GET: mockGET,
    POST: mockPOST,
    PUT: mockPUT,
    DELETE: mockDELETE,
  },
}));

import {
  ApiRequestError,
  createCity,
  disableReportedStory,
  getStory,
  listAuditLogs,
  listCities,
  listAllCities,
  listPOIs,
  listReports,
  listStories,
} from './endpoints';

describe('trace ID propagation', () => {
  it('includes traceId on ApiRequestError when backend returns trace_id', async () => {
    mockGET.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'story not found', trace_id: 'abc-trace-1' },
      response: { status: 404 },
    });

    try {
      await getStory(1);
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as ApiRequestError;
      expect(apiErr.message).toBe('story not found');
      expect(apiErr.status).toBe(404);
      expect(apiErr.traceId).toBe('abc-trace-1');
    }
  });

  it('omits traceId when backend does not return trace_id', async () => {
    mockGET.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'story not found' },
      response: { status: 404 },
    });

    try {
      await getStory(1);
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as ApiRequestError;
      expect(apiErr.message).toBe('story not found');
      expect(apiErr.traceId).toBeUndefined();
    }
  });

  it('propagates traceId from list endpoints', async () => {
    mockGET.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'forbidden', trace_id: 'list-trace-99' },
    });

    try {
      await listCities();
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      expect((err as ApiRequestError).traceId).toBe('list-trace-99');
    }
  });

  it('propagates traceId from POST endpoints', async () => {
    mockPOST.mockResolvedValueOnce({
      data: undefined,
      error: { error: 'moderation failed', trace_id: 'post-trace-42' },
      response: { status: 500 },
    });

    try {
      await disableReportedStory(1);
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as ApiRequestError;
      expect(apiErr.traceId).toBe('post-trace-42');
      expect(apiErr.status).toBe(500);
    }
  });

  it('preserves validation details for city mutations', async () => {
    mockPOST.mockResolvedValueOnce({
      data: undefined,
      error: {
        error: 'validation_error',
        details: [{ field: 'name', message: 'must not be blank' }],
        trace_id: 'city-trace-12',
      },
      response: { status: 400 },
    });

    try {
      await createCity({
        name: '',
        country: 'GE',
        center_lat: 0,
        center_lng: 0,
        radius_km: 10,
        is_active: true,
        download_size_mb: 0,
      });
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as ApiRequestError;
      expect(apiErr.message).toBe('Validation failed');
      expect(apiErr.details).toEqual([{ field: 'name', message: 'must not be blank' }]);
      expect(apiErr.requestId).toBe('city-trace-12');
      expect(apiErr.status).toBe(400);
    }
  });

  it('uses fallback message when error.error is missing', async () => {
    mockGET.mockResolvedValueOnce({
      data: undefined,
      error: { trace_id: 'fallback-trace' },
      response: { status: 500 },
    });

    try {
      await getStory(1);
      expect.fail('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiRequestError);
      const apiErr = err as ApiRequestError;
      expect(apiErr.message).toBe('Failed to fetch story');
      expect(apiErr.traceId).toBe('fallback-trace');
    }
  });

  it('listCities calls the admin /admin/cities endpoint, not public /cities', async () => {
    mockGET.mockResolvedValueOnce({
      data: {
        items: [
          { id: 1, name: 'Active', is_active: true },
          { id: 2, name: 'Inactive', is_active: false },
        ],
        next_cursor: '',
        has_more: false,
      },
      error: undefined,
    });

    const result = await listCities();

    expect(mockGET).toHaveBeenCalledWith('/admin/cities', {
      params: { query: {} },
    });
    // Admin endpoint should return both active and inactive cities.
    expect(result.items).toHaveLength(2);
    expect(result.items.some((c: any) => c.is_active === false)).toBe(true);
  });

  it('listAllCities collects all pages from admin endpoint', async () => {
    mockGET
      .mockResolvedValueOnce({
        data: {
          items: [{ id: 1, name: 'City1', is_active: true }],
          next_cursor: 'cur-1',
          has_more: true,
        },
        error: undefined,
      })
      .mockResolvedValueOnce({
        data: {
          items: [{ id: 2, name: 'City2', is_active: false }],
          next_cursor: '',
          has_more: false,
        },
        error: undefined,
      });

    const result = await listAllCities();

    expect(result).toHaveLength(2);
    // Verify both calls went to admin endpoint.
    expect(mockGET).toHaveBeenNthCalledWith(1, '/admin/cities', {
      params: { query: { limit: 100 } },
    });
    expect(mockGET).toHaveBeenNthCalledWith(2, '/admin/cities', {
      params: { query: { cursor: 'cur-1', limit: 100 } },
    });
  });

  it('calls the admin audit log endpoint with filters', async () => {
    mockGET.mockResolvedValueOnce({
      data: {
        items: [],
        next_cursor: '',
        has_more: false,
      },
      error: undefined,
    });

    const result = await listAuditLogs({
      actor_id: 'admin-1',
      resource_type: 'city',
      action: 'create',
      created_from: '2026-01-01T00:00:00Z',
      created_to: '2026-01-31T23:59:59Z',
      limit: 25,
    });

    expect(result.items).toEqual([]);
    expect(mockGET).toHaveBeenCalledWith('/admin/audit-logs', {
      params: {
        query: {
          actor_id: 'admin-1',
          resource_type: 'city',
          action: 'create',
          created_from: '2026-01-01T00:00:00Z',
          created_to: '2026-01-31T23:59:59Z',
          limit: 25,
        },
      },
    });
  });

  it('passes sort params to the admin audit log endpoint', async () => {
    mockGET.mockResolvedValueOnce({
      data: { items: [], next_cursor: '', has_more: false },
      error: undefined,
    });

    await listAuditLogs({ sort_by: 'created_at', sort_dir: 'desc' });

    expect(mockGET).toHaveBeenCalledWith('/admin/audit-logs', {
      params: { query: { sort_by: 'created_at', sort_dir: 'desc' } },
    });
  });

  it('passes sort params to the admin city list endpoint', async () => {
    mockGET.mockResolvedValueOnce({
      data: { items: [], next_cursor: '', has_more: false },
      error: undefined,
    });

    await listCities({ sort_by: 'name', sort_dir: 'desc' });

    expect(mockGET).toHaveBeenCalledWith('/admin/cities', {
      params: { query: { sort_by: 'name', sort_dir: 'desc' } },
    });
  });

  it('calls the admin POI list endpoint with sort params', async () => {
    mockGET.mockResolvedValueOnce({
      data: { items: [], next_cursor: '', has_more: false },
      error: undefined,
    });

    await listPOIs({ city_id: 1, sort_by: 'interest_score', sort_dir: 'desc' });

    expect(mockGET).toHaveBeenCalledWith('/admin/pois', {
      params: { query: { city_id: 1, sort_by: 'interest_score', sort_dir: 'desc' } },
    });
  });

  it('calls the admin story list endpoint with sort params', async () => {
    mockGET.mockResolvedValueOnce({
      data: { items: [], next_cursor: '', has_more: false },
      error: undefined,
    });

    await listStories({ poi_id: 5, sort_by: 'order_index', sort_dir: 'asc' });

    expect(mockGET).toHaveBeenCalledWith('/admin/stories', {
      params: { query: { poi_id: 5, sort_by: 'order_index', sort_dir: 'asc' } },
    });
  });

  it('passes sort params to the admin reports list endpoint', async () => {
    mockGET.mockResolvedValueOnce({
      data: { items: [], next_cursor: '', has_more: false },
      error: undefined,
    });

    await listReports({ sort_by: 'created_at', sort_dir: 'desc' });

    expect(mockGET).toHaveBeenCalledWith('/admin/reports', {
      params: { query: { sort_by: 'created_at', sort_dir: 'desc' } },
    });
  });
});
