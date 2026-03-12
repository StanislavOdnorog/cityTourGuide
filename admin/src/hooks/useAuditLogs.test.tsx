import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { listAuditLogs } from '../api';
import { createQueryClientWrapper, createTestQueryClient } from '../test/queryClient';
import { useAuditLogs } from './useAuditLogs';

vi.mock('../api', () => ({
  listAuditLogs: vi.fn(),
}));

const logItem = {
  id: 1,
  actor_id: 'admin-1',
  action: 'create',
  resource_type: 'city',
  resource_id: '5',
  http_method: 'POST',
  request_path: '/api/v1/admin/cities',
  trace_id: 'trace-1',
  payload: { name: 'Tbilisi' },
  status: 'success',
  created_at: '2026-01-01T00:00:00Z',
};

describe('useAuditLogs', () => {
  it('fetches audit logs with default options', async () => {
    vi.mocked(listAuditLogs).mockResolvedValue({
      items: [logItem],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useAuditLogs(), { wrapper });

    await waitFor(() => {
      expect(result.current.logs.data?.items).toEqual([logItem]);
    });

    expect(listAuditLogs).toHaveBeenCalledWith({ limit: 20 });
  });

  it('passes audit log filters through to the API client', async () => {
    vi.mocked(listAuditLogs).mockResolvedValue({
      items: [],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(
      () =>
        useAuditLogs({
          actorId: 'admin-1',
          action: 'create',
          resourceType: 'city',
          createdFrom: '2026-01-01T00:00:00Z',
          createdTo: '2026-01-31T23:59:59Z',
          cursor: 'abc',
          limit: 10,
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.logs.data).toBeDefined();
    });

    expect(listAuditLogs).toHaveBeenCalledWith({
      actor_id: 'admin-1',
      action: 'create',
      resource_type: 'city',
      created_from: '2026-01-01T00:00:00Z',
      created_to: '2026-01-31T23:59:59Z',
      cursor: 'abc',
      limit: 10,
    });
  });

  it('omits empty filter values from the request', async () => {
    vi.mocked(listAuditLogs).mockResolvedValue({
      items: [],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(
      () =>
        useAuditLogs({
          actorId: '',
          action: '',
          resourceType: '',
          createdFrom: '',
          createdTo: '',
          limit: 20,
        }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.logs.data).toBeDefined();
    });

    expect(listAuditLogs).toHaveBeenCalledWith({ limit: 20 });
  });

  it('returns loading state initially', () => {
    vi.mocked(listAuditLogs).mockReturnValue(new Promise(() => {}));

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useAuditLogs(), { wrapper });

    expect(result.current.logs.isLoading).toBe(true);
    expect(result.current.logs.data).toBeUndefined();
  });
});
