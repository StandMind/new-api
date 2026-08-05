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
import type { TFunction } from 'i18next'

import type { LogOtherData, RoutingDiagnosticSnapshot } from '../types'

export function getRoutingModeLabel(
  t: TFunction,
  mode: string | undefined
): string {
  switch (mode) {
    case 'manual':
      return t('Manual group')
    case 'fixed_channel':
      return t('Fixed channel')
    case 'auto':
      return `${t('Smart routing')} · ${t('Auto')}`
    case 'price':
      return `${t('Smart routing')} · ${t('Price')}`
    case 'speed':
      return `${t('Smart routing')} · ${t('Speed')}`
    case 'success_rate':
      return `${t('Smart routing')} · ${t('Success rate')}`
    default:
      return mode || '-'
  }
}

export function getRoutingModeShortLabel(
  t: TFunction,
  mode: string | undefined
): string {
  switch (mode) {
    case 'manual':
      return t('Manual')
    case 'fixed_channel':
      return t('Fixed')
    case 'auto':
      return t('Smart · Auto')
    case 'price':
      return t('Smart · Price')
    case 'speed':
      return t('Smart · Speed')
    case 'success_rate':
      return t('Smart · Success rate')
    default:
      return mode || '-'
  }
}

export function getRoutingReasonLabel(t: TFunction, reason?: string): string {
  if (!reason) return '-'
  switch (reason) {
    case 'success':
      return t('Request completed')
    case 'client_context_canceled':
      return t('Client connection closed')
    case 'client_context_deadline_exceeded':
      return t('Client request deadline exceeded')
    case 'channel_affinity_retry_suppressed':
      return t('Channel affinity disabled retry')
    case 'retry_limit_reached':
      return t('Retry limit reached')
    case 'specific_channel':
      return t('A specific channel was required')
    case 'skip_retry_error':
      return t('The error is marked as non-retryable')
    case 'non_retryable_error_code':
      return t('The error code is configured as non-retryable')
    case 'non_retryable_status':
      return t('The status code is not retryable')
    case 'route_plan_exhausted':
      return t('No route candidates remain')
    case 'response_started':
      return t('The response had already started')
    case 'channel_initialization_failed':
      return t('Channel initialization failed')
    case 'no_available_channel':
      return t('No candidate channel was available')
    case 'no_candidate_groups':
      return t('No candidate group was available')
    case 'route_plan_build_failed':
      return t('The route plan could not be built')
    case 'get_channel_failed':
      return t('Channel selection failed')
    case 'request_failed_before_upstream':
      return t('The request failed before reaching upstream')
    case 'pricing_failed':
      return t('Pricing failed before the upstream request')
    case 'billing_reserve_failed':
      return t('Quota reservation failed')
    case 'request_body_unavailable':
      return t('The request body was unavailable')
    case 'local_task_error':
      return t('A local task error stopped retrying')
    case 'task_retry_not_safe':
      return t('The task could not be retried safely')
    default:
      return reason
  }
}

export function getRoutingSnapshot(
  other: LogOtherData | null
): RoutingDiagnosticSnapshot | undefined {
  return other?.admin_info?.routing
}
