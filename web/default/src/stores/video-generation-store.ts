import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { VideoGenerationHistoryItem } from '@/features/video-generation/types'

interface VideoGenerationState {
  history: VideoGenerationHistoryItem[]
  clearHistory: () => void
}

export const useVideoGenerationStore = create<VideoGenerationState>()(
  persist(
    (set) => ({
      history: [],
      clearHistory: () => {
        set({ history: [] })
      },
    }),
    {
      name: 'video-generation-storage',
      partialize: (state) => ({
        history: state.history,
      }),
    }
  )
)