import { api } from '@/lib/api'

export type MyRouteGroup = {
  code: string
  name: string
  description: string
  base_ratio: number
  enabled: boolean
  price_ratio: number | null
}

export type MyRouteGroupsResponse = {
  success: boolean
  message?: string
  data?: {
    user_level: string
    route_groups: MyRouteGroup[]
  }
}

export async function getMyRouteGroups(): Promise<MyRouteGroupsResponse> {
  const response = await api.get('/api/user/route-groups')
  return response.data
}
