export type AccessPolicyReference = {
  kind: string
  count: number
}

export type UserLevelRouteGroup = {
  user_level_code: string
  route_group_code: string
  price_ratio: number | null
}

export type UserLevel = {
  code: string
  name: string
  description: string
  is_default: boolean
  enabled: boolean
  topup_ratio: number
  request_limit: number
  success_request_limit: number
  route_groups: UserLevelRouteGroup[]
  references: AccessPolicyReference[]
}

export type RouteGroup = {
  code: string
  name: string
  description: string
  base_ratio: number
  enabled: boolean
  references: AccessPolicyReference[]
}

export type UserLevelInput = Omit<
  UserLevel,
  'code' | 'route_groups' | 'references'
>

export type RouteGroupInput = Omit<RouteGroup, 'code' | 'references'>

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data: T
}
