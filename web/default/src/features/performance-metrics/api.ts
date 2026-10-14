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
  ModelSLOMetricsData,
  PerformanceMetricsData,
  PerfSummaryAllData,
  SLAMetricsData,
} from './types'

export type PerfSummaryParams = {
  hours?: number
  start_timestamp?: number
  end_timestamp?: number
}

export type PerfMetricsParams = {
  group?: string
  hours?: number
  start_timestamp?: number
  end_timestamp?: number
}

export type ModelSLOMetricsParams = {
  hours?: number
  start_timestamp?: number
  end_timestamp?: number
  model_name?: string
  slo_ttft_threshold_ms?: number
  slo_tpot_threshold_ms?: number
}

export async function getPerfMetricsSummary(
  hoursOrParams: number | PerfSummaryParams = 24
): Promise<PerfSummaryAllData> {
  const params =
    typeof hoursOrParams === 'number'
      ? { hours: hoursOrParams }
      : (hoursOrParams ?? {})

  const res = await api.get<PerfSummaryAllData>('/api/perf-metrics/summary', {
    params,
  })
  return res.data
}

export async function getPerfMetrics(
  modelName: string,
  hoursOrParams: number | PerfMetricsParams = 24
): Promise<PerformanceMetricsData> {
  const params =
    typeof hoursOrParams === 'number'
      ? { hours: hoursOrParams }
      : (hoursOrParams ?? {})

  const res = await api.get<PerformanceMetricsData>('/api/perf-metrics', {
    params: {
      model: modelName,
      ...params,
    },
  })
  return res.data
}

export async function getModelSLOMetrics(
  params: ModelSLOMetricsParams
): Promise<ModelSLOMetricsData> {
  const res = await api.get<ModelSLOMetricsData>(
    '/api/perf-metrics/model-slo',
    {
      params,
    }
  )
  return res.data
}

export type SLAMetricsParams = {
  mode?: 'model' | 'user'
  hours?: number
  start_timestamp?: number
  end_timestamp?: number
  username?: string
  model_name?: string
}

export async function getSLAMetrics(
  params: SLAMetricsParams
): Promise<SLAMetricsData> {
  const res = await api.get<SLAMetricsData>('/api/perf-metrics/model-sla', {
    params,
  })
  return res.data
}
