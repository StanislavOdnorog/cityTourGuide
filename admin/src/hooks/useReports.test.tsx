import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { disableReportedStory, getAdminStats, listReports, updateReportStatus } from '../api';
import type { AdminReportListItem } from '../types';
import { createQueryClientWrapper, createTestQueryClient } from '../test/queryClient';
import { useNewReportsCount, useReports } from './useReports';

vi.mock('../api', () => ({
  disableReportedStory: vi.fn(),
  getAdminStats: vi.fn(),
  listReports: vi.fn(),
  updateReportStatus: vi.fn(),
}));

const report: AdminReportListItem = {
  id: 1,
  story_id: 10,
  user_id: 'user-1',
  poi_id: 100,
  poi_name: 'Old Town',
  story_language: 'en',
  story_status: 'active',
  type: 'wrong_fact',
  comment: 'Needs correction',
  status: 'new',
  created_at: '2026-01-01T00:00:00Z',
};

describe('useReports', () => {
  it('fetches a single cursor page using status, cursor, and limit', async () => {
    vi.mocked(listReports).mockResolvedValue({
      items: [report],
      next_cursor: '',
      has_more: false,
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(
      () => useReports({ status: 'new', cursor: 'cursor-2', limit: 5 }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.reports.data?.items).toEqual([report]);
    });

    expect(listReports).toHaveBeenCalledWith({
      status: 'new',
      cursor: 'cursor-2',
      limit: 5,
    });
  });

  it('refetches the active reports page after a report status mutation succeeds', async () => {
    vi.mocked(listReports).mockResolvedValue({
      items: [report],
      next_cursor: '',
      has_more: false,
    });
    vi.mocked(updateReportStatus).mockResolvedValue({ data: { ...report, status: 'resolved' } });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useReports({ status: 'new', cursor: 'cursor-1', limit: 20 }), {
      wrapper,
    });

    await waitFor(() => {
      expect(listReports).toHaveBeenCalledTimes(1);
    });

    await act(async () => {
      await result.current.updateStatus.mutateAsync({ reportId: report.id, newStatus: 'resolved' });
    });

    await waitFor(() => {
      expect(listReports).toHaveBeenCalledTimes(2);
    });
  });

  it('refetches the active reports page after atomic moderation succeeds', async () => {
    vi.mocked(listReports).mockResolvedValue({
      items: [report],
      next_cursor: '',
      has_more: false,
    });
    vi.mocked(disableReportedStory).mockResolvedValue({
      data: {
        report: { ...report, status: 'resolved' },
        story: { id: report.story_id, poi_id: report.poi_id!, language: 'en', status: 'disabled' },
      },
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useReports({ status: '', cursor: undefined, limit: 20 }), {
      wrapper,
    });

    await waitFor(() => {
      expect(listReports).toHaveBeenCalledTimes(1);
    });

    await act(async () => {
      await result.current.disableStory.mutateAsync(report.id);
    });

    await waitFor(() => {
      expect(disableReportedStory).toHaveBeenCalledWith(report.id);
      expect(listReports).toHaveBeenCalledTimes(2);
    });
  });
});

describe('useNewReportsCount', () => {
  it('returns new_reports_count from admin stats', async () => {
    vi.mocked(getAdminStats).mockResolvedValue({
      data: {
        cities_count: 1,
        pois_count: 5,
        stories_count: 9,
        reports_count: 3,
        new_reports_count: 2,
      },
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useNewReportsCount(), { wrapper });

    await waitFor(() => {
      expect(result.current.data).toBe(2);
    });
  });

  it('returns zero when there are no new reports', async () => {
    vi.mocked(getAdminStats).mockResolvedValue({
      data: {
        cities_count: 1,
        pois_count: 5,
        stories_count: 9,
        reports_count: 0,
        new_reports_count: 0,
      },
    });

    const queryClient = createTestQueryClient();
    const wrapper = createQueryClientWrapper(queryClient);
    const { result } = renderHook(() => useNewReportsCount(), { wrapper });

    await waitFor(() => {
      expect(result.current.data).toBe(0);
    });
  });
});
