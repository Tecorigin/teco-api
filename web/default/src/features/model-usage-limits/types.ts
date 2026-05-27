export type LimitItem = {
  id: number
  user_id: number
  username: string
  model_name: string
  daily_token_limit: number
  daily_request_limit: number
  weekly_token_limit: number
  weekly_request_limit: number
  monthly_token_limit: number
  monthly_request_limit: number
  daily_token_used: number
  daily_request_used: number
  weekly_token_used: number
  weekly_request_used: number
  monthly_token_used: number
  monthly_request_used: number
}

export type HitItem = {
  id: number
  created_at: number
  user_id: number
  username: string
  model_name: string
  limit_type: string
  period_type: string
  used_value: number
  limit_value: number
  request_id: string
}

export type GetLimitsParams = {
  keyword?: string
  p?: number
  page_size?: number
}

export type GetHitsParams = {
  keyword?: string
  period_type?: string
  limit_type?: string
  p?: number
  page_size?: number
}

export type PaginatedResponse<T> = {
  success: boolean
  message: string
  data: {
    items: T[]
    total: number
  }
}

export type LimitFormValues = {
  user_id: string
  model_name: string
  daily_token_limit: string
  daily_request_limit: string
  weekly_token_limit: string
  weekly_request_limit: string
  monthly_token_limit: string
  monthly_request_limit: string
}

export const LIMIT_FORM_DEFAULTS: LimitFormValues = {
  user_id: '',
  model_name: '',
  daily_token_limit: '0',
  daily_request_limit: '0',
  weekly_token_limit: '0',
  weekly_request_limit: '0',
  monthly_token_limit: '0',
  monthly_request_limit: '0',
}
