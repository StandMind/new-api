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
  ApplyOpenLuxSyncRequest,
  ApplyOpenLuxSyncResult,
  OpenLuxApiResponse,
  OpenLuxBindings,
  OpenLuxPreview,
  SaveOpenLuxBindingsRequest,
} from './types'

function unwrap<T>(response: OpenLuxApiResponse<T>): T {
  if (!response.success || response.data === undefined) {
    throw new Error(response.message || 'OpenLux request failed')
  }
  return response.data
}

export async function getOpenLuxBindings() {
  const response = await api.get<OpenLuxApiResponse<OpenLuxBindings>>(
    '/api/openlux-price-sync/bindings',
    { skipErrorHandler: true }
  )
  return unwrap(response.data)
}

export async function saveOpenLuxBindings(request: SaveOpenLuxBindingsRequest) {
  const response = await api.put<OpenLuxApiResponse<OpenLuxBindings>>(
    '/api/openlux-price-sync/bindings',
    request,
    { skipErrorHandler: true }
  )
  return unwrap(response.data)
}

export async function previewOpenLuxSync() {
  const response = await api.post<OpenLuxApiResponse<OpenLuxPreview>>(
    '/api/openlux-price-sync/preview',
    {},
    { skipErrorHandler: true }
  )
  return unwrap(response.data)
}

export async function applyOpenLuxSync(request: ApplyOpenLuxSyncRequest) {
  const response = await api.post<OpenLuxApiResponse<ApplyOpenLuxSyncResult>>(
    '/api/openlux-price-sync/apply',
    request,
    { skipErrorHandler: true }
  )
  return unwrap(response.data)
}
