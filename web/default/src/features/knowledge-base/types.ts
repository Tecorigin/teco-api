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

export interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
}

export interface KbStatus {
  configured: boolean
  enabled: boolean
  org_id?: string
  org_name?: string
  created_at?: number
}

export interface KnowledgeBase {
  name: string
  description?: string
  embedding_model?: string
  document_count?: number
  chunk_count?: number
}

export interface KbDoc {
  id: string
  name: string
  size?: number
  chunk_count?: number
  run?: 'UNSTART' | 'RUNNING' | 'DONE' | 'FAIL' | string
}

export interface KbSession {
  session_id: string
  name?: string
}

export interface KbTaskStatus {
  task_id: string
  task_type?: string
  kb_name?: string
  status: 'pending' | 'running' | 'completed' | 'failed' | string
  progress?: number
  message?: string
  hint?: string
}

export interface KbCreateParams {
  kb_name: string
  description?: string
  chunk_method?: string
  chunk_token_num?: number
  overlapped_percent?: number
  auto_keywords?: number
  auto_questions?: number
  html4excel?: boolean
  toc_extraction?: boolean
  graphrag?: Record<string, unknown> | null
  raptor?: Record<string, unknown> | null
  parent_child?: Record<string, unknown> | null
  metadata?: Record<string, unknown> | null
  tag_kb_ids?: string[] | null
}

export interface KbAdminUser {
  id: number
  user_id: number
  org_id: string
  org_name: string
  kb_user_id: string
  status: string
  created_at: number
}

export interface KbReference {
  chunk_content?: string
  document_name?: string
  document_id?: string
  similarity?: number
  vector_similarity?: number
  term_similarity?: number
}

export interface KbChatMessage {
  role: 'user' | 'assistant'
  content: string
  references?: KbReference[]
}
