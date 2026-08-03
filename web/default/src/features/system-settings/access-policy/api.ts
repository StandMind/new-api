import { api } from '@/lib/api'

import type {
  ApiResponse,
  RouteGroup,
  RouteGroupInput,
  UserLevel,
  UserLevelInput,
} from './types'

function unwrap<T>(response: ApiResponse<T>): T {
  if (!response.success) throw new Error(response.message || 'Request failed')
  return response.data
}

export async function getUserLevels(): Promise<UserLevel[]> {
  const response = await api.get<ApiResponse<UserLevel[]>>('/api/user-levels')
  return unwrap(response.data)
}

export async function createUserLevel(
  code: string,
  input: UserLevelInput
): Promise<UserLevel> {
  const response = await api.post<ApiResponse<UserLevel>>('/api/user-levels', {
    code,
    ...input,
  })
  return unwrap(response.data)
}

export async function updateUserLevel(
  code: string,
  input: UserLevelInput
): Promise<UserLevel> {
  const response = await api.put<ApiResponse<UserLevel>>(
    `/api/user-levels/${encodeURIComponent(code)}`,
    input
  )
  return unwrap(response.data)
}

export async function deleteUserLevel(code: string): Promise<void> {
  const response = await api.delete<ApiResponse<unknown>>(
    `/api/user-levels/${encodeURIComponent(code)}`
  )
  unwrap(response.data)
}

export async function replaceUserLevelRouteGroups(
  code: string,
  routeGroups: { code: string; price_ratio: number | null }[]
): Promise<void> {
  const response = await api.put<ApiResponse<unknown>>(
    `/api/user-levels/${encodeURIComponent(code)}/route-groups`,
    { route_groups: routeGroups }
  )
  unwrap(response.data)
}

export async function getRouteGroups(): Promise<RouteGroup[]> {
  const response = await api.get<ApiResponse<RouteGroup[]>>('/api/route-groups')
  return unwrap(response.data)
}

export async function createRouteGroup(
  code: string,
  input: RouteGroupInput
): Promise<RouteGroup> {
  const response = await api.post<ApiResponse<RouteGroup>>(
    '/api/route-groups',
    {
      code,
      ...input,
    }
  )
  return unwrap(response.data)
}

export async function updateRouteGroup(
  code: string,
  input: RouteGroupInput
): Promise<RouteGroup> {
  const response = await api.put<ApiResponse<RouteGroup>>(
    `/api/route-groups/${encodeURIComponent(code)}`,
    input
  )
  return unwrap(response.data)
}

export async function deleteRouteGroup(code: string): Promise<void> {
  const response = await api.delete<ApiResponse<unknown>>(
    `/api/route-groups/${encodeURIComponent(code)}`
  )
  unwrap(response.data)
}
