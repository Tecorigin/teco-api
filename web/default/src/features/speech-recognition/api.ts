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

import type { ModelOption, SpeechRecognitionRequest, SpeechRecognitionResponse } from './types'

async function getAllModels() {
  const response = await api.get('/api/models/tags')
  const items = response.data?.data || []
  return items as Array<{ model_name: string; tags?: string }>
}

export async function getUserModels(includeAll = false): Promise<ModelOption[]> {
  const [userModelsRes, modelsRes] = await Promise.all([
    api.get('/api/user/models'),
    includeAll ? Promise.resolve([]) : getAllModels(),
  ])

  if (!userModelsRes.data?.success || !Array.isArray(userModelsRes.data.data)) {
    return []
  }

  if (includeAll) {
    return userModelsRes.data.data.map((model: string) => ({ label: model, value: model }))
  }

  const speechModelSet = new Set(
    modelsRes
      .filter((item) => parseModelTags(item.tags).includes('语音识别'))
      .map((item) => item.model_name)
  )

  return userModelsRes.data.data
    .filter((model: string) => speechModelSet.has(model))
    .map((model: string) => ({ label: model, value: model }))
}

export async function recognizeSpeech(
  payload: SpeechRecognitionRequest
): Promise<SpeechRecognitionResponse> {
  const response = await api.post('/pg/chat/completions', payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return response.data as SpeechRecognitionResponse
}