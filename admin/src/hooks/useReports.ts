import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getStory, listReports, updateReportStatus, updateStory } from '../api';
import type { Report, ReportStatus, Story } from '../types';

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
      return listReports({
        page,
        per_page: perPage,
        ...(status ? { status } : {}),
      });
    },
    staleTime: 15_000,
  });

  const updateStatus = useMutation({
    mutationFn: async ({ reportId, newStatus }: { reportId: number; newStatus: ReportStatus }) => {
      const response = await updateReportStatus(reportId, { status: newStatus });
      return response.data as Report;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ['reports-new-count'] });
    },
  });

  const disableStory = useMutation({
    mutationFn: async (storyId: number) => {
      const currentResponse = await getStory(storyId);
      const response = await updateStory(storyId, currentResponse.data as Story, {
        status: 'disabled',
      });
      return response.data as Story;
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
      const response = await listReports({ status: 'new', page: 1, per_page: 1 });
      return response.meta.total;
    },
    staleTime: 30_000,
  });
}
