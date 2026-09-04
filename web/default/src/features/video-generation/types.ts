export interface ModelOption {
  label: string
  value: string
}

export interface VideoGenerationRequest {
  model: string
  prompt: string
  size: string
  seconds: number
  response_format: 'url'
  num_inference_steps: number
  seed: number
  guidance_scale: number
  guidance_scale_2: number
  fps: number
}

export interface VideoGenerationItem {
  url?: string
  download_url?: string
  b64_json?: string
}

export interface VideoGenerationTaskLogItem {
  id: number
  task_id: string
  status?: string
  progress?: string
  created_at?: number
  updated_at?: number
  finish_time?: number
  fail_reason?: string
  result_url?: string
  url?: string
  download_url?: string
}

export interface VideoGenerationTaskQueryResponse {
  success?: boolean
  message?: string
  data?: {
    items?: VideoGenerationTaskLogItem[]
  }
}

export interface VideoGenerationResponse {
  created: number
  data: VideoGenerationItem[]
  elapsed_time?: string
  id?: string
  task_id?: string
  object?: string
  model?: string
  status?: string
  progress?: number
  created_at?: number
  completed_at?: number
  expires_at?: number
  size?: string
  seconds?: string
  url?: string
  download_url?: string
  error?: {
    message?: string
    code?: string
  }
}

export interface PersistedGeneratedVideo {
  src: string
  elapsed?: string
  isExpired?: boolean
  isBase64?: boolean
}

export interface VideoGenerationHistoryItem {
  id: string
  createdAt: number
  request: VideoGenerationRequest
  videos: PersistedGeneratedVideo[]
}