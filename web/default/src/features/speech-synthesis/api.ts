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
import { parseModelTags } from '@/features/models/lib/model-utils'
import { api } from '@/lib/api'

import type { ModelOption, SpeechSynthesisRequest } from './types'

async function getAllModels() {
  const response = await api.get('/api/models/tags')
  const items = response.data?.data || []
  return items as Array<{ model_name: string; tags?: string }>
}

export async function getUserModels(): Promise<ModelOption[]> {
  const [userModelsRes, modelsRes] = await Promise.all([
    api.get('/api/user/models'),
    getAllModels(),
  ])

  if (!userModelsRes.data?.success || !Array.isArray(userModelsRes.data.data)) {
    return []
  }

  const speechModelSet = new Set(
    modelsRes
      .filter((item) => parseModelTags(item.tags).includes('语音合成'))
      .map((item) => item.model_name)
  )

  return userModelsRes.data.data
    .filter((model: string) => speechModelSet.has(model))
    .map((model: string) => ({ label: model, value: model }))
}

export async function synthesizeSpeech(payload: SpeechSynthesisRequest): Promise<Blob> {
  const response = await api.post('/pg/audio/speech', payload, {
    responseType: 'blob',
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return response.data as Blob
}