import { api } from '@/lib/api'
import { parseModelTags } from '@/features/models/lib/model-utils'
import type {
  ModelOption,
  VideoGenerationTaskQueryResponse,
  VideoGenerationRequest,
  VideoGenerationResponse,
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

  const videoModelSet = new Set(
    modelItems
      .filter((item) => parseModelTags(item.tags).includes('文生视频'))
      .map((item) => item.model_name)
  )

  return userModels.data
    .filter((model: string) => videoModelSet.has(model))
    .map((model: string) => ({
      label: model,
      value: model,
    }))
}

export async function generateVideos(
  payload: VideoGenerationRequest
): Promise<VideoGenerationResponse> {
  const res = await api.post('/pg/videos', payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function getVideoTask(taskId: string) {
  const res = await api.get('/api/task/self', {
    params: {
      task_id: taskId,
      p: 0,
      page_size: 1,
    },
    skipErrorHandler: true,
  } as Record<string, unknown>)

  const payload = res.data as VideoGenerationTaskQueryResponse
  return payload?.data?.items?.[0] ?? null
}