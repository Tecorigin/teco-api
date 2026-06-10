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
import { api } from '@/lib/api'
import { getModels } from '@/features/models/api'
import { parseModelTags } from '@/features/models/lib/model-utils'
import type {
  ImageGenerationRequest,
  ImageGenerationResponse,
  ModelOption,
} from './types'

async function getAllModels() {
  const pageSize = 100
  let page = 1
  const items = [] as Array<{ model_name: string; tags?: string }>

  while (true) {
    const response = await getModels({ p: page, page_size: pageSize })
    const currentItems = response.data?.items || []
    items.push(...currentItems)

    if (currentItems.length < pageSize) {
      break
    }

    page += 1
  }

  return items
}

export async function getUserModels(): Promise<ModelOption[]> {
  const [userModelsRes, modelsRes] = await Promise.all([
    api.get('/api/user/models'),
    getAllModels(),
  ])

  const userModels = userModelsRes.data
  const modelItems = modelsRes

  if (!userModels.success || !Array.isArray(userModels.data)) {
    return []
  }

  const imageModelSet = new Set(
    modelItems
      .filter((item) => parseModelTags(item.tags).includes('文生图'))
      .map((item) => item.model_name)
  )

  return userModels.data
    .filter((model: string) => imageModelSet.has(model))
    .map((model: string) => ({
      label: model,
      value: model,
    }))
}

export async function generateImages(
  payload: ImageGenerationRequest
): Promise<ImageGenerationResponse> {
  const res = await api.post('/pg/images/generations', payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}
