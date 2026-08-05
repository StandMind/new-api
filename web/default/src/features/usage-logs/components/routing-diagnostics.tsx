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
import { AlertTriangle, CheckCircle2, CircleSlash2, Route } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { cn } from '@/lib/utils'

import { getRoutingModeLabel, getRoutingReasonLabel } from '../lib/routing'
import type {
  RouteAttempt,
  RouteAttemptDiagnostic,
  RoutingDiagnosticSnapshot,
} from '../types'

function channelLabel(attempt: RouteAttempt): string {
  return attempt.channel_name
    ? `${attempt.channel_name} #${attempt.channel_id}`
    : `#${attempt.channel_id}`
}

function attemptVariant(outcome: string) {
  if (outcome === 'succeeded') return 'green' as const
  if (outcome === 'failed') return 'red' as const
  return 'yellow' as const
}

function AttemptIcon({ outcome }: { outcome: string }) {
  if (outcome === 'succeeded') {
    return <CheckCircle2 className='size-3.5 text-emerald-500' />
  }
  if (outcome === 'failed') {
    return <AlertTriangle className='size-3.5 text-red-500' />
  }
  return <CircleSlash2 className='size-3.5 text-amber-500' />
}

function PlannedRoute({
  attempt,
  index,
}: {
  attempt: RouteAttempt
  index: number
}) {
  const { t } = useTranslation()
  return (
    <div className='grid min-w-0 grid-cols-[2rem_minmax(0,1fr)] gap-2 border-b py-2 last:border-b-0'>
      <span className='text-muted-foreground pt-0.5 text-right font-mono text-xs'>
        {index + 1}
      </span>
      <div className='min-w-0 space-y-1'>
        <div className='flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs'>
          <span className='font-medium break-all'>{channelLabel(attempt)}</span>
          <span className='text-muted-foreground font-mono'>
            {t('Priority')} {attempt.priority}
          </span>
          <StatusBadge
            label={attempt.explicit ? t('Configured') : t('Inherited')}
            variant='neutral'
            size='sm'
            copyable={false}
            showDot={false}
          />
        </div>
        <p className='text-muted-foreground font-mono text-[11px] break-all'>
          {attempt.group} · {attempt.model}
        </p>
      </div>
    </div>
  )
}

function AttemptDetail({ attempt }: { attempt: RouteAttemptDiagnostic }) {
  const { t } = useTranslation()
  const retryReason = attempt.retry_stop_reason || attempt.retry_reason
  let outcomeLabel = 'Skipped'
  if (attempt.outcome === 'succeeded') {
    outcomeLabel = 'Succeeded'
  } else if (attempt.outcome === 'failed') {
    outcomeLabel = 'Failed'
  }

  let retryDecisionLabel = 'Stop'
  if (attempt.retry_decision === 'retry') {
    retryDecisionLabel = 'Retry'
  } else if (attempt.retry_decision === 'complete') {
    retryDecisionLabel = 'Complete'
  }

  return (
    <div className='grid min-w-0 grid-cols-[2rem_minmax(0,1fr)] gap-2 border-b py-2.5 last:border-b-0'>
      <div className='flex flex-col items-center gap-1 pt-0.5'>
        <AttemptIcon outcome={attempt.outcome} />
        <span className='text-muted-foreground font-mono text-[10px]'>
          {attempt.sequence}
        </span>
      </div>
      <div className='min-w-0 space-y-1.5'>
        <div className='flex flex-wrap items-center gap-1.5'>
          <span className='text-xs font-medium break-all'>
            {channelLabel(attempt)}
          </span>
          <StatusBadge
            label={t(outcomeLabel)}
            variant={attemptVariant(attempt.outcome)}
            size='sm'
            copyable={false}
          />
          {attempt.status_code != null && attempt.status_code > 0 && (
            <StatusBadge
              label={String(attempt.status_code)}
              variant={attempt.status_code >= 400 ? 'red' : 'neutral'}
              size='sm'
              copyable={false}
              showDot={false}
              className='font-mono'
            />
          )}
          <span className='text-muted-foreground font-mono text-[11px]'>
            {attempt.duration_ms} ms
          </span>
        </div>
        <p className='text-muted-foreground font-mono text-[11px] break-all'>
          {attempt.group} · {attempt.phase}
          {attempt.upstream_model ? ` · ${attempt.upstream_model}` : ''}
          {attempt.request_format ? ` · ${attempt.request_format}` : ''}
        </p>
        {(attempt.error_code || attempt.error_type) && (
          <p className='font-mono text-[11px] break-all'>
            {[attempt.error_type, attempt.error_code]
              .filter(Boolean)
              .join(' · ')}
          </p>
        )}
        {attempt.error_message && (
          <p className='text-xs leading-relaxed break-all whitespace-pre-wrap text-red-600 dark:text-red-400'>
            {attempt.error_message}
          </p>
        )}
        {attempt.upstream_request_id && (
          <p className='text-muted-foreground text-[11px] break-all'>
            {t('Upstream Request ID')}: {attempt.upstream_request_id}
          </p>
        )}
        {attempt.retry_decision && (
          <p className='text-[11px]'>
            <span className='text-muted-foreground'>
              {t('Retry decision')}:{' '}
            </span>
            <span className='font-medium'>{t(retryDecisionLabel)}</span>
            {retryReason ? ` · ${getRoutingReasonLabel(t, retryReason)}` : ''}
          </p>
        )}
      </div>
    </div>
  )
}

export function RoutingDiagnostics({
  routing,
  className,
}: {
  routing: RoutingDiagnosticSnapshot
  className?: string
}) {
  const { t } = useTranslation()
  const planned = routing.planned ?? []
  const attempts = routing.attempts ?? []

  return (
    <div className={cn('min-w-0 space-y-3', className)}>
      <div className='grid gap-x-4 gap-y-1.5 text-xs sm:grid-cols-2'>
        <div>
          <span className='text-muted-foreground'>{t('Routing')}: </span>
          <span className='font-medium'>
            {getRoutingModeLabel(t, routing.mode)}
          </span>
        </div>
        {routing.basis && (
          <div>
            <span className='text-muted-foreground'>
              {t('Routing basis')}:{' '}
            </span>
            <span className='font-mono break-all'>{routing.basis}</span>
          </div>
        )}
        {routing.groups && routing.groups.length > 0 && (
          <div className='sm:col-span-2'>
            <span className='text-muted-foreground'>
              {t('Candidate groups')}:{' '}
            </span>
            <span className='font-mono break-all'>
              {routing.groups.join(' → ')}
            </span>
          </div>
        )}
        {routing.final_group && (
          <div>
            <span className='text-muted-foreground'>{t('Final group')}: </span>
            <span className='font-mono break-all'>{routing.final_group}</span>
          </div>
        )}
        {routing.final_channel_id != null && routing.final_channel_id > 0 && (
          <div>
            <span className='text-muted-foreground'>
              {t('Final channel')}:{' '}
            </span>
            <span className='font-mono break-all'>
              {routing.final_channel_name
                ? `${routing.final_channel_name} #${routing.final_channel_id}`
                : `#${routing.final_channel_id}`}
            </span>
          </div>
        )}
        {routing.final_stop_reason && (
          <div className='sm:col-span-2'>
            <span className='text-muted-foreground'>
              {t('Final stop reason')}:{' '}
            </span>
            <span className='font-medium'>
              {getRoutingReasonLabel(t, routing.final_stop_reason)}
            </span>
            <span className='text-muted-foreground ml-1 font-mono'>
              ({routing.final_stop_reason})
            </span>
          </div>
        )}
      </div>

      {planned.length > 0 && (
        <div className='min-w-0'>
          <div className='mb-1 flex items-center gap-1.5 text-xs font-semibold'>
            <Route className='size-3.5' aria-hidden='true' />
            {t('Planned route')} ({planned.length})
            {routing.planned_truncated ? ` · ${t('Truncated')}` : ''}
          </div>
          <div className='bg-muted/20 min-w-0 rounded-md border px-2'>
            {planned.map((attempt, index) => (
              <PlannedRoute
                key={`${attempt.group}-${attempt.model}-${attempt.priority}-${attempt.channel_id}`}
                attempt={attempt}
                index={index}
              />
            ))}
          </div>
        </div>
      )}

      <div className='min-w-0'>
        <div className='mb-1 text-xs font-semibold'>
          {t('Attempt chain')} ({attempts.length})
          {routing.attempts_truncated ? ` · ${t('Truncated')}` : ''}
        </div>
        {attempts.length > 0 ? (
          <div className='bg-muted/20 min-w-0 rounded-md border px-2'>
            {attempts.map((attempt) => (
              <AttemptDetail key={attempt.sequence} attempt={attempt} />
            ))}
          </div>
        ) : (
          <p className='text-muted-foreground text-xs'>
            {t('No channel attempt was recorded.')}
          </p>
        )}
      </div>
    </div>
  )
}
