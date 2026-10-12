import { nanoid } from 'nanoid/non-secure'
import { z } from 'zod'

export const routingPaths = [
  '/v1/chat/completions',
  '/v1/messages',
  '/v1/responses',
  '/pg/chat/completions',
] as const

const instanceSchema = z
  .object({
    instance_id: z.string(),
    channel_ids: z.array(z.number().int().positive()),
    max_inflight: z.preprocess(
      (value) => value ?? 0,
      z.number().int().min(0).max(10000)
    ),
  })
  .passthrough()

const rangeSchema = z
  .object({
    min_tokens: z.number().int().min(0).max(10000000),
    max_tokens: z.number().int().min(1).max(10000000).nullable().optional(),
    instances: z.array(instanceSchema),
  })
  .passthrough()

const ruleSchema = z
  .object({
    id: z.string(),
    models: z.array(z.string()),
    groups: z.array(z.string()).default([]),
    ranges: z.array(rangeSchema).max(100),
  })
  .passthrough()

export const routingConfigurationSchema = z
  .object({
    enabled: z.boolean(),
    version: z.string().optional(),
    state_backend: z.string().optional(),
    rules: z.array(ruleSchema).max(100),
  })
  .passthrough()

export type RoutingConfiguration = z.infer<typeof routingConfigurationSchema>
export type RoutingRule = z.infer<typeof ruleSchema>
export type RoutingInstance = z.infer<typeof instanceSchema>
export type RoutingRange = z.infer<typeof rangeSchema>

export const emptyRoutingConfiguration: RoutingConfiguration = {
  enabled: false,
  rules: [],
}

export function newRoutingRule(): RoutingRule {
  return {
    id: `rule-${nanoid(12)}`,
    models: [],
    groups: [],
    ranges: [{ min_tokens: 0, max_tokens: null, instances: [] }],
  }
}

export function newRoutingInstance(): RoutingInstance {
  return {
    instance_id: `instance-${nanoid(12)}`,
    channel_ids: [],
    max_inflight: 0,
  }
}
