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
export type OpenLuxChannelSummary = {
  id: number
  name: string
  local_group: string
  status: number
  model_count: number
  bound_source_group?: string
}

export type OpenLuxBinding = {
  source_group: string
  local_group: string
  channels: OpenLuxChannelSummary[] | null
  mixed_channel_count: number
  healthy: boolean
  issues: string[] | null
  candidate?: boolean
}

export type OpenLuxSourceGroup = {
  name: string
  description: string
}

export type OpenLuxBindings = {
  binding_revision: number
  bindings: OpenLuxBinding[] | null
  candidates: OpenLuxBinding[] | null
  source_groups: OpenLuxSourceGroup[] | null
  openlux_channels: OpenLuxChannelSummary[] | null
}

export type SaveOpenLuxBindingsRequest = {
  expected_revision: number
  bindings: Array<{
    source_group: string
    channel_ids: number[]
  }>
}

export type OpenLuxNotice = {
  kind: string
  model?: string
  source_group?: string
  local_group?: string
  message: string
}

export type OpenLuxChangeDetail = {
  field: string
  current?: string
  target?: string
}

export type OpenLuxChangeKind =
  | 'model_billing_update'
  | 'group_model_price_update'
  | 'group_model_remove'
  | 'source_group_remove'

export type OpenLuxChange = {
  id: string
  kind: OpenLuxChangeKind
  model?: string
  source_group?: string
  local_group?: string
  current_value?: string
  target_value?: string
  current_price?: string
  target_price?: string
  price_unit?: string
  percent_change?: string
  actionable: boolean
  destructive: boolean
  requires: string[] | null
  blocked_reasons: string[] | null
  affected_groups?: string[] | null
  mixed_channel_count?: number
  remove_mode?: string
  details?: OpenLuxChangeDetail[] | null
}

export type OpenLuxPreview = {
  source_hash: string
  local_fingerprint: string
  binding_revision: number
  fetched_at: number
  changes: OpenLuxChange[]
  notices: OpenLuxNotice[]
  summary: {
    actionable: number
    blocked: number
    notices: number
    price: number
    removal: number
  }
}

export type ApplyOpenLuxSyncRequest = {
  source_hash: string
  local_fingerprint: string
  binding_revision: number
  change_ids: string[]
}

export type ApplyOpenLuxSyncResult = {
  applied_count: number
  updated_models: string[]
  updated_groups: string[]
  removed_channels: number[]
  binding_revision: number
  source_hash: string
}

export type OpenLuxApiResponse<T> = {
  success: boolean
  code?: string
  message: string
  data?: T
}
