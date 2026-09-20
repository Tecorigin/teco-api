import { api, getUserModels } from '@/lib/api'
import { getGroups, getUsers } from '@/features/users/api'

export type UserRateLimitScope = 'global' | 'group' | 'user' | 'user_model'

export type UserRateLimit = {
  id: number
  scope: UserRateLimitScope
  user_id: number
  username: string
  group_name: string
  model_name: string
  rpm: number
  tpm: number
}

export type UserRateLimitPayload = Omit<UserRateLimit, 'id' | 'username'>

export async function getUserRateLimits(): Promise<UserRateLimit[]> {
  const response = await api.get('/api/user_rate_limit?p=1&page_size=1000')
  if (!response.data.success) throw new Error(response.data.message)
  return response.data.data.items ?? []
}

export async function saveUserRateLimit(payload: UserRateLimitPayload, id?: number) {
  const response = id
    ? await api.put(`/api/user_rate_limit/${id}`, payload)
    : await api.post('/api/user_rate_limit', payload)
  if (!response.data.success) throw new Error(response.data.message)
}

export async function deleteUserRateLimit(id: number) {
  const response = await api.delete(`/api/user_rate_limit/${id}`)
  if (!response.data.success) throw new Error(response.data.message)
}

export async function getRateLimitUsers() {
  const response = await getUsers({ p: 1, page_size: 1000 })
  return (response.data?.items ?? []).map((user) => ({
    label: `${user.username || user.display_name || user.id} (${user.id})`,
    value: String(user.id),
  }))
}

export async function getRateLimitGroups() {
  const response = await getGroups()
  return response.data ?? []
}

export async function getRateLimitModels() {
  const response = await getUserModels()
  return response.data ?? []
}