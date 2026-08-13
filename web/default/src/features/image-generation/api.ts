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
import { parseModelTags } from '@/features/models/lib/model-utils'
import type {
  ImageGenerationRequest,
  ImageGenerationResponse,
  ModelOption,
} from './types'

// 获取全部模型的基础元数据（模型名、标签）。
// 使用轻量接口 /api/models/tags：一次返回全部模型（无分页限制），
// 且普通登录用户即可访问，不再依赖管理员权限的 /api/models/ 分页接口。
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

  const userModels = userModelsRes.data
  const modelItems = modelsRes

  if (!userModels.success || !Array.isArray(userModels.data)) {
    return []
  }

  if (includeAll) {
    return userModels.data.map((model: string) => ({ label: model, value: model }))
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
