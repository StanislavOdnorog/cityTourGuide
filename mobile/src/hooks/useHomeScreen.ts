import { useCallback, useEffect, useRef } from 'react';
import { createWalkingPipeline, type WalkingPipeline } from '@/services/pipeline';
import { getAuthenticatedUserId } from '@/store/authBootstrap';
import { useAuthStore } from '@/store/useAuthStore';
import { usePlayerStore } from '@/store/usePlayerStore';
import { useSettingsStore } from '@/store/useSettingsStore';
import { useWalkStore } from '@/store/useWalkStore';

interface HomeScreenState {
  isWalking: boolean;
  isPlaying: boolean;
  currentStoryName: string | null;
  listenedCount: number;
  progress: { position: number; duration: number };
}

interface HomeScreenActions {
  toggleWalking: () => Promise<void>;
}

export function useHomeScreen(): HomeScreenState & HomeScreenActions {
  const isWalking = useWalkStore((s) => s.isWalking);
  const currentStory = usePlayerStore((s) => s.currentStory);
  const isPlaying = usePlayerStore((s) => s.isPlaying);
  const progress = usePlayerStore((s) => s.progress);
  const listenedStoryIds = usePlayerStore((s) => s.listenedStoryIds);

  const language = useSettingsStore((s) => s.language);
  const authUserId = useAuthStore((s) => s.userId);
  const pipelineRef = useRef<WalkingPipeline | null>(null);

  useEffect(() => {
    return () => {
      if (pipelineRef.current) {
        void pipelineRef.current.destroy();
        pipelineRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (!pipelineRef.current) {
      return;
    }

    pipelineRef.current.updateConfig({
      language,
      userId: authUserId ?? '',
    });
  }, [authUserId, language]);

  const toggleWalking = useCallback(async () => {
    if (isWalking) {
      if (pipelineRef.current) {
        await pipelineRef.current.stop();
      }
    } else {
      const userId = await getAuthenticatedUserId();
      if (!userId) {
        return;
      }

      if (!pipelineRef.current) {
        pipelineRef.current = createWalkingPipeline({ language, userId });
      } else {
        pipelineRef.current.updateConfig({ language, userId });
      }
      try {
        await pipelineRef.current.start();
      } catch (err) {
        console.warn('Failed to start walking pipeline:', err);
        pipelineRef.current = null;
      }
    }
  }, [isWalking, language]);

  return {
    isWalking,
    isPlaying,
    currentStoryName: currentStory?.poi_name ?? null,
    listenedCount: listenedStoryIds.size,
    progress,
    toggleWalking,
  };
}
