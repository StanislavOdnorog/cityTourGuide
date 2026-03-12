import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  getPOI,
  getStory,
  listAllStories,
  listInflationJobsByPOI,
  listReportsByPOI,
  triggerInflation as triggerInflationRequest,
  updatePOI as updatePOIRequest,
  updateStory as updateStoryRequest,
} from '../api';
import { handleMutationError } from '../api/errors';
import type {
  InflationJob,
  POI,
  Report,
  Story,
  StoryStatus,
} from '../types';

export function usePOIDetail(poiId: number | null) {
  const queryClient = useQueryClient();

  const poi = useQuery({
    queryKey: ['poi', poiId],
    queryFn: async () => {
      const response = await getPOI(poiId as number);
      return response.data as POI;
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const stories = useQuery({
    queryKey: ['poi-stories', poiId],
    queryFn: async () => {
      // Fetch stories for all languages by not specifying language filter
      const storyLists = await Promise.all(
        ['en', 'ru'].map((language) =>
          listAllStories({
            poi_id: poiId as number,
            language,
            limit: 100,
          }),
        ),
      );
      return storyLists.flat() as Story[];
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const reports = useQuery({
    queryKey: ['poi-reports', poiId],
    queryFn: async () => {
      const response = await listReportsByPOI(poiId as number);
      return response.data as Report[];
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const inflationJobs = useQuery({
    queryKey: ['poi-inflation-jobs', poiId],
    queryFn: async () => {
      const response = await listInflationJobsByPOI(poiId as number);
      return response.data as InflationJob[];
    },
    enabled: poiId !== null,
    staleTime: 30_000,
  });

  const updatePOI = useMutation({
    mutationFn: async (updates: Partial<POI>) => {
      const current = poi.data;
      if (!current) throw new Error('POI not loaded');
      const response = await updatePOIRequest(poiId as number, current, updates);
      return response.data as POI;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi', poiId] });
    },
    onError: handleMutationError,
  });

  const updateStory = useMutation({
    mutationFn: async ({ storyId, updates }: { storyId: number; updates: Partial<Story> }) => {
      const currentResponse = await getStory(storyId);
      const response = await updateStoryRequest(storyId, currentResponse.data as Story, updates);
      return response.data as Story;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi-stories', poiId] });
    },
    onError: handleMutationError,
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
      const currentResponse = await getStory(storyId);
      const response = await updateStoryRequest(storyId, currentResponse.data as Story, {
        status: newStatus,
      });
      return response.data as Story;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi-stories', poiId] });
    },
    onError: handleMutationError,
  });

  const triggerInflation = useMutation({
    mutationFn: async () => {
      const response = await triggerInflationRequest(poiId as number);
      return response.data as InflationJob;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['poi-inflation-jobs', poiId] });
    },
    onError: handleMutationError,
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
