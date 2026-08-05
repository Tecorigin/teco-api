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

import type {
  ApiResponse,
  KbStatus,
  KnowledgeBase,
  KbDoc,
  KbSession,
  KbTaskStatus,
  KbCreateParams,
  KbAdminUser,
} from './types'

const BASE = '/api/knowledge-base'

// Status / provisioning
export async function getKbStatus(): Promise<ApiResponse<KbStatus>> {
  const res = await api.get(`${BASE}/status`)
  return res.data
}

export async function enableKb(): Promise<ApiResponse<{ org_id: string; org_name: string }>> {
  const res = await api.post(`${BASE}/enable`)
  return res.data
}

export async function disableKb(): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE}/disable`)
  return res.data
}

// Knowledge base CRUD
export async function listKb(): Promise<ApiResponse<KnowledgeBase[]>> {
  const res = await api.post(`${BASE}/list`)
  return res.data
}

export async function createKb(params: KbCreateParams): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE}/create`, params)
  return res.data
}

export async function deleteKb(kbName: string): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE}/delete`, { kb_name: kbName })
  return res.data
}

export async function clearKb(kbName: string): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE}/clear`, { kb_name: kbName })
  return res.data
}

// Docs
export async function listKbDocs(kbName: string): Promise<ApiResponse<KbDoc[]>> {
  const res = await api.post(`${BASE}/list_docs`, { kb_name: kbName })
  return res.data
}

export async function deleteKbDocs(kbName: string, docIds: string[]): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE}/delete_docs`, { kb_name: kbName, doc_ids: docIds })
  return res.data
}

export async function uploadKbDocs(
  kbName: string,
  files: File[]
): Promise<ApiResponse<{ task_id: string }>> {
  const form = new FormData()
  form.append('kb_name', kbName)
  files.forEach((f) => form.append('files', f))
  const res = await api.post(`${BASE}/upload_docs`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data
}

export async function runKbIndex(
  kbName: string,
  indexType: string
): Promise<ApiResponse<{ task_id: string; hint?: string }>> {
  const res = await api.post(`${BASE}/run_index`, { kb_name: kbName, index_type: indexType })
  return res.data
}

export async function getKbTaskStatus(taskId: string): Promise<ApiResponse<KbTaskStatus>> {
  const res = await api.post(`${BASE}/task_status`, { task_id: taskId })
  return res.data
}

// Chat sessions
export async function createKbSession(): Promise<ApiResponse<{ session_id: string }>> {
  const res = await api.post(`${BASE}/chat/create_session`)
  return res.data
}

export async function deleteKbSession(sessionId: string): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE}/chat/delete_session`, { session_id: sessionId })
  return res.data
}

export async function listKbSessions(): Promise<ApiResponse<KbSession[]>> {
  const res = await api.post(`${BASE}/chat/list_sessions`)
  return res.data
}

// Admin
export async function adminListKbUsers(): Promise<ApiResponse<KbAdminUser[]>> {
  const res = await api.get(`${BASE}/admin/users`)
  return res.data
}
