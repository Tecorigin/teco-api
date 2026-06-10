import { api } from '@/lib/api'
import { getModels } from '@/features/models/api'
import { parseModelTags } from '@/features/models/lib/model-utils'
import type {
  ModelOption,
  VideoGenerationTaskQueryResponse,
  VideoGenerationRequest,
  VideoGenerationResponse,
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