/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ImageGenerationRequest } from '@/features/image-generation/types'

export type PersistedGeneratedImage = {
  src: string
  elapsed?: string
  sourceType: 'url' | 'b64_json'
}

export type ImageGenerationHistoryItem = {
  id: string
  createdAt: number
  request: ImageGenerationRequest
  images: PersistedGeneratedImage[]
}

interface ImageGenerationState {
  history: ImageGenerationHistoryItem[]

  clearHistory: () => void
}

const MAX_HISTORY_ITEMS = 20

export const useImageGenerationStore = create<ImageGenerationState>()(
  persist(
    (set, get) => ({
      history: [],

      clearHistory: () => {
        set({ history: [] })
      },
    }),
    {
      name: 'image-generation-storage',
      partialize: (state) => ({
        history: state.history,
      }),
    }
  )
)
