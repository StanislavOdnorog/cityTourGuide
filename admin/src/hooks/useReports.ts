import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '../api/client';
import type { PaginatedResponse, Report, ReportStatus, Story } from '../types';

interface UseReportsOptions {
  status?: ReportStatus | '';
  page?: number;
  perPage?: number;
}

export function useReports({ status = '', page = 1, perPage = 20 }: UseReportsOptions = {}) {
  const queryClient = useQueryClient();

  const reports = useQuery({
    queryKey: ['reports', status, page, perPage],
    queryFn: async () => {
      const params: Record<string, unknown> = { page, per_page: perPage };
      if (status) {
        params.status = status;
      }
      const { data } = await apiClient.get<PaginatedResponse<Report>>('/api/v1/admin/reports', {
        params,
      });
      return data;
    },
    staleTime: 15_000,
  });

  const updateStatus = useMutation({
    mutationFn: async ({ reportId, newStatus }: { reportId: number; newStatus: ReportStatus }) => {
      const { data } = await apiClient.put<{ data: Report }>(
        `/api/v1/admin/reports/${reportId}`,
        { status: newStatus },
      );
      return data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ['reports-new-count'] });
    },
  });

  const disableStory = useMutation({
    mutationFn: async (storyId: number) => {
      const { data: currentData } = await apiClient.get<{ data: Story }>(
        `/api/v1/stories/${storyId}`,
      );
      const current = currentData.data;
      const { data } = await apiClient.put<{ data: Story }>(`/api/v1/admin/stories/${storyId}`, {
        ...current,
        status: 'disabled',
      });
      return data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ['poi-stories'] });
    },
  });

  return { reports, updateStatus, disableStory };
}

export function useNewReportsCount() {
  return useQuery({
    queryKey: ['reports-new-count'],
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedResponse<Report>>('/api/v1/admin/reports', {
        params: { status: 'new', page: 1, per_page: 1 },
      });
      return data.meta.total;
    },
    staleTime: 30_000,
  });
}
