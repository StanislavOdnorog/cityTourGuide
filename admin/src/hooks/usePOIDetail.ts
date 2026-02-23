import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '../api/client';
import type {
  InflationJob,
  POI,
  PaginatedResponse,
  Report,
  Story,
  StoryStatus,
} from '../types';

export function usePOIDetail(poiId: number | null) {
  const queryClient = useQueryClient();

  const poi = useQuery({
    queryKey: ['poi', poiId],
    queryFn: async () => {
      const { data } = await apiClient.get<{ data: POI }>(`/api/v1/pois/${poiId}`);
      return data.data;
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const stories = useQuery({
    queryKey: ['poi-stories', poiId],
    queryFn: async () => {
      // Fetch stories for all languages by not specifying language filter
      const allStories: Story[] = [];
      for (const lang of ['en', 'ru']) {
        const { data } = await apiClient.get<PaginatedResponse<Story>>('/api/v1/stories', {
          params: { poi_id: poiId, language: lang, per_page: 100 },
        });
        allStories.push(...data.data);
      }
      return allStories;
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const reports = useQuery({
    queryKey: ['poi-reports', poiId],
    queryFn: async () => {
      const { data } = await apiClient.get<{ data: Report[] }>(
        `/api/v1/admin/pois/${poiId}/reports`,
      );
      return data.data;
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const inflationJobs = useQuery({
    queryKey: ['poi-inflation-jobs', poiId],
    queryFn: async () => {
      const { data } = await apiClient.get<{ data: InflationJob[] }>(
        `/api/v1/admin/pois/${poiId}/inflation-jobs`,
      );
      return data.data;
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const updatePOI = useMutation({
    mutationFn: async (updates: Partial<POI>) => {
      const current = poi.data;
      if (!current) throw new Error('POI not loaded');
      const { data } = await apiClient.put<{ data: POI }>(`/api/v1/admin/pois/${poiId}`, {
        ...current,
        ...updates,
      });
      return data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi', poiId] });
    },
  });

  const updateStory = useMutation({
    mutationFn: async ({ storyId, updates }: { storyId: number; updates: Partial<Story> }) => {
      // Fetch current story to merge updates
      const { data: currentData } = await apiClient.get<{ data: Story }>(
        `/api/v1/stories/${storyId}`,
      );
      const current = currentData.data;
      const { data } = await apiClient.put<{ data: Story }>(`/api/v1/admin/stories/${storyId}`, {
        ...current,
        ...updates,
      });
      return data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi-stories', poiId] });
    },
  });

  const toggleStoryStatus = useMutation({
    mutationFn: async ({
      storyId,
      currentStatus,
    }: {
      storyId: number;
      currentStatus: StoryStatus;
    }) => {
      const newStatus: StoryStatus = currentStatus === 'active' ? 'disabled' : 'active';
      const { data: currentData } = await apiClient.get<{ data: Story }>(
        `/api/v1/stories/${storyId}`,
      );
      const current = currentData.data;
      const { data } = await apiClient.put<{ data: Story }>(`/api/v1/admin/stories/${storyId}`, {
        ...current,
        status: newStatus,
      });
      return data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi-stories', poiId] });
    },
  });

  const triggerInflation = useMutation({
    mutationFn: async () => {
      const { data } = await apiClient.post<{ data: InflationJob }>(
        `/api/v1/admin/pois/${poiId}/inflate`,
      );
      return data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi-inflation-jobs', poiId] });
    },
  });

  return {
    poi,
    stories,
    reports,
    inflationJobs,
    updatePOI,
    updateStory,
    toggleStoryStatus,
    triggerInflation,
  };
}
