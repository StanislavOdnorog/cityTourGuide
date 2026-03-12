import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getStory, listAllReports, updateReportStatus, updateStory } from '../api';
import type { Report, ReportStatus, Story } from '../types';

interface UseReportsOptions {
  status?: ReportStatus | '';
}

export function useReports({ status = '' }: UseReportsOptions = {}) {
  const queryClient = useQueryClient();

  const reports = useQuery({
    queryKey: ['reports', status, 'all'],
    queryFn: () =>
      listAllReports({
        limit: 100,
        ...(status ? { status } : {}),
      }) as Promise<Report[]>,
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
    queryFn: async () => (await listAllReports({ status: 'new', limit: 100 })).length,
    staleTime: 30_000,
  });
}
