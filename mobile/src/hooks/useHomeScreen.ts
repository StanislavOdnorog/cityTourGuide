import { useCallback, useEffect, useRef } from 'react';
import { createWalkingPipeline, type WalkingPipeline } from '@/services/pipeline';
import { usePlayerStore } from '@/store/usePlayerStore';
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

  const pipelineRef = useRef<WalkingPipeline | null>(null);

  useEffect(() => {
    return () => {
      if (pipelineRef.current) {
        void pipelineRef.current.destroy();
        pipelineRef.current = null;
      }
    };
  }, []);

  const toggleWalking = useCallback(async () => {
    if (isWalking) {
      if (pipelineRef.current) {
        await pipelineRef.current.stop();
      }
    } else {
      if (!pipelineRef.current) {
        pipelineRef.current = createWalkingPipeline({ language: 'en' });
      }
      await pipelineRef.current.start();
    }
  }, [isWalking]);

  return {
    isWalking,
    isPlaying,
    currentStoryName: currentStory?.poi_name ?? null,
    listenedCount: listenedStoryIds.size,
    progress,
    toggleWalking,
  };
}
