import { api, getUserModels } from '@/lib/api'
import { getUsers } from '@/features/users/api'
import type {
  GetLimitsParams,
  GetHitsParams,
  LimitItem,
  HitItem,
  PaginatedResponse,
} from './types'

export async function getLimits(
  params: GetLimitsParams = {}
): Promise<PaginatedResponse<LimitItem>> {
  const { keyword = '', p = 1, page_size = 10 } = params
  const res = await api.get(
    `/api/user_model_usage_limit?keyword=${keyword}&p=${p}&page_size=${page_size}`
  )
  return res.data
}

export async function createLimit(data: Record<string, unknown>) {
  const res = await api.post('/api/user_model_usage_limit', data)
  return res.data
}

export async function updateLimit(id: number, data: Record<string, unknown>) {
  const res = await api.put(`/api/user_model_usage_limit/${id}`, data)
  return res.data
}

export async function deleteLimit(id: number) {
  const res = await api.delete(`/api/user_model_usage_limit/${id}`)
  return res.data
}

export async function getHits(
  params: GetHitsParams = {}
): Promise<PaginatedResponse<HitItem>> {
  const {
    keyword = '',
    period_type = '',
    limit_type = '',
    p = 1,
    page_size = 10,
  } = params
  const res = await api.get(
    `/api/user_model_usage_limit/hits?keyword=${keyword}&period_type=${period_type}&limit_type=${limit_type}&p=${p}&page_size=${page_size}`
  )
  return res.data
}

export async function getLimitUserOptions() {
  const res = await getUsers({ p: 1, page_size: 1000 })
  const items = res?.data?.items || []
  return items.map((item: any) => ({
    label: `${item.username || item.display_name || item.id} (${item.id})`,
    value: String(item.id),
  }))
}

export async function getLimitModelOptions() {
  const res = await getUserModels()
  const items = res?.data || []
  return items.map((item) => ({
    label: item,
    value: item,
  }))
}
