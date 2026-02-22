import { create } from 'zustand';
import type { ScoredCandidate } from '@/services/story-engine';

interface PlayerState {
  currentStory: ScoredCandidate | null;
  isPlaying: boolean;
  progress: { position: number; duration: number };
  listenedStoryIds: Set<number>;
}

interface PlayerActions {
  setCurrentStory: (story: ScoredCandidate | null) => void;
  setIsPlaying: (playing: boolean) => void;
  setProgress: (position: number, duration: number) => void;
  addListenedStory: (storyId: number) => void;
  reset: () => void;
}

const initialState: PlayerState = {
  currentStory: null,
  isPlaying: false,
  progress: { position: 0, duration: 0 },
  listenedStoryIds: new Set<number>(),
};

export const usePlayerStore = create<PlayerState & PlayerActions>((set) => ({
  ...initialState,

  setCurrentStory: (story) => set({ currentStory: story }),
  setIsPlaying: (playing) => set({ isPlaying: playing }),
  setProgress: (position, duration) => set({ progress: { position, duration } }),
  addListenedStory: (storyId) =>
    set((state) => ({
      listenedStoryIds: new Set(state.listenedStoryIds).add(storyId),
    })),
  reset: () =>
    set({
      currentStory: null,
      isPlaying: false,
      progress: { position: 0, duration: 0 },
      listenedStoryIds: new Set<number>(),
    }),
}));
