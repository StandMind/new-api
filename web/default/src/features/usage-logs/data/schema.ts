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
/**
 * Zod schemas for common logs
 * This file should only contain Zod schemas and types inferred from them
 */
import { z } from 'zod'

const usageLogBaseSchema = z.object({
  id: z.number(),
  user_id: z.number(),
  created_at: z.number(),
  type: z.number(),
  content: z.string(),
  username: z.string().default(''),
  token_name: z.string().default(''),
  model_name: z.string().default(''),
  quota: z.number().default(0),
  prompt_tokens: z.number().default(0),
  completion_tokens: z.number().default(0),
  use_time: z.number().default(0),
  is_stream: z.boolean().default(false),
  token_id: z.number().default(0),
  group: z.string().default(''),
  ip: z.string().default(''),
  other: z.string().default(''),
  request_id: z.string().default(''),
  upstream_request_id: z.string().default(''),
})

export const userUsageLogSchema = usageLogBaseSchema

export const adminUsageLogSchema = usageLogBaseSchema.extend({
  channel: z.number(),
  channel_name: z.string().nullish().default(''),
})

// Admin is checked first so callers that choose to parse the union retain the
// administrator-only fields instead of having them stripped by the base schema.
export const usageLogSchema = z.union([adminUsageLogSchema, userUsageLogSchema])

export type UserUsageLog = z.infer<typeof userUsageLogSchema>
export type AdminUsageLog = z.infer<typeof adminUsageLogSchema>
export type UsageLog = UserUsageLog | AdminUsageLog

export function isAdminUsageLog(log: UsageLog): log is AdminUsageLog {
  return 'channel' in log && typeof log.channel === 'number'
}
