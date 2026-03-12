import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { disableReportedStory, getAdminStats, listReports, updateReportStatus } from '../api';
import { handleMutationError } from '../api/errors';
import type { AdminReportListItem, ReportStatus } from '../types';

interface UseReportsOptions {
  status?: ReportStatus | '';
  cursor?: string;
  limit?: number;
}

export function useReports({ status = '', cursor, limit = 20 }: UseReportsOptions = {}) {
  const queryClient = useQueryClient();

  const reports = useQuery({
    queryKey: ['reports', status, cursor ?? null, limit],
    queryFn: () =>
      listReports({
        limit,
        ...(cursor ? { cursor } : {}),
        ...(status ? { status } : {}),
      }),
    staleTime: 15_000,
  });

  const updateStatus = useMutation({
    mutationFn: async ({ reportId, newStatus }: { reportId: number; newStatus: ReportStatus }) => {
      const response = await updateReportStatus(reportId, { status: newStatus });
      return response.data as AdminReportListItem;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
    },
    onError: handleMutationError,
  });

  const disableStory = useMutation({
    mutationFn: async (reportId: number) => {
      const response = await disableReportedStory(reportId);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ['poi-stories'] });
      queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
    },
    onError: handleMutationError,
  });

  return { reports, updateStatus, disableStory };
}

export function useNewReportsCount() {
  return useQuery({
    queryKey: ['admin', 'stats'],
    queryFn: getAdminStats,
    staleTime: 60_000,
    select: (data) => data.data.new_reports_count,
  });
}
