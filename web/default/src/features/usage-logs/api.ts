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

import type { AdminUsageLog, UserUsageLog } from './data/schema'
import type {
  AdminMidjourneyLog,
  AdminTaskLog,
  GetLogsParams,
  GetLogsResponse,
  GetLogStatsParams,
  GetLogStatsResponse,
  GetMidjourneyLogsParams,
  GetUserMidjourneyLogsParams,
  GetTaskLogsParams,
  GetUserTaskLogsParams,
  UserMidjourneyLog,
  UserTaskLog,
  RequestDetailResponse,
  RequestDetailsResponse,
  UserInfo,
} from './types'

// ============================================================================
// Generic API Helpers
// ============================================================================

function buildQueryParams(params: Record<string, unknown>): URLSearchParams {
  const queryParams = new URLSearchParams()

  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      queryParams.append(key, String(value))
    }
  })

  return queryParams
}

function buildApiPath(endpoint: string, isAdmin: boolean): string {
  return isAdmin ? endpoint : `${endpoint}/self`
}

async function fetchLogs<
  T,
  TItem extends
    | AdminUsageLog
    | UserUsageLog
    | AdminMidjourneyLog
    | UserMidjourneyLog
    | AdminTaskLog
    | UserTaskLog,
>(
  endpoint: string,
  params: T,
  isAdmin: boolean
): Promise<GetLogsResponse<TItem>> {
  const paramRecord = params as unknown as Record<string, unknown>
  const queryParams = buildQueryParams({
    p: paramRecord.p || 1,
    page_size: paramRecord.page_size || 20,
    ...params,
  })
  const path = buildApiPath(endpoint, isAdmin)
  const res = await api.get(`${path}?${queryParams}`)
  return res.data
}

async function fetchLogStats<T>(
  endpoint: string,
  params: T,
  isAdmin: boolean
): Promise<GetLogStatsResponse> {
  const queryParams = buildQueryParams(
    params as unknown as Record<string, unknown>
  )
  const path = buildApiPath(endpoint, isAdmin)
  const res = await api.get(`${path}/stat?${queryParams}`)
  return res.data
}

// ============================================================================
// Common Log APIs
// ============================================================================

export const getAllLogs = (params: GetLogsParams = {}) =>
  fetchLogs<GetLogsParams, AdminUsageLog>('/api/log', params, true)

export const getUserLogs = (
  params: Omit<GetLogsParams, 'username' | 'channel'> = {}
) =>
  fetchLogs<Omit<GetLogsParams, 'username' | 'channel'>, UserUsageLog>(
    '/api/log',
    params,
    false
  )

export const getLogStats = (params: GetLogStatsParams = {}) =>
  fetchLogStats('/api/log', params, true)

export const getUserLogStats = (
  params: Omit<GetLogStatsParams, 'username' | 'channel'> = {}
) => fetchLogStats('/api/log', params, false)

export async function getUserInfo(
  userId: number
): Promise<{ success: boolean; message?: string; data?: UserInfo }> {
  const res = await api.get(`/api/user/${userId}`)
  return res.data
}

export async function getRequestDetails(params: {
  p?: number
  page_size?: number
  username?: string
  model_name?: string
  outcome?: string
  request_id?: string
}): Promise<RequestDetailsResponse> {
  const query = buildQueryParams(params)
  const res = await api.get(`/api/request-detail/?${query}`)
  return res.data
}

export async function getRequestDetail(
  requestId: string
): Promise<RequestDetailResponse> {
  const res = await api.get(
    `/api/request-detail/${encodeURIComponent(requestId)}`
  )
  return res.data
}

// ============================================================================
// MjProxy (Drawing) Logs API
// ============================================================================

export const getAllMidjourneyLogs = (params: GetMidjourneyLogsParams) =>
  fetchLogs<GetMidjourneyLogsParams, AdminMidjourneyLog>(
    '/api/mj',
    params,
    true
  )

export const getUserMidjourneyLogs = (params: GetUserMidjourneyLogsParams) =>
  fetchLogs<GetUserMidjourneyLogsParams, UserMidjourneyLog>(
    '/api/mj',
    params,
    false
  )

// ============================================================================
// Task Logs API
// ============================================================================

export const getAllTaskLogs = (params: GetTaskLogsParams) =>
  fetchLogs<GetTaskLogsParams, AdminTaskLog>('/api/task', params, true)

export const getUserTaskLogs = (params: GetUserTaskLogsParams) =>
  fetchLogs<GetUserTaskLogsParams, UserTaskLog>('/api/task', params, false)
