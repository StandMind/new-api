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
import { useQuery } from '@tanstack/react-query'
import { Check, Copy, Loader2 } from 'lucide-react'
import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { formatTimestampToDate } from '@/lib/format'

import { getRequestDetail } from '../../api'
import { RoutingDiagnostics } from '../routing-diagnostics'

type RequestDetailDialogProps = {
  requestId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

function MetadataRow(props: { label: string; value: ReactNode }) {
  return (
    <div className='grid min-w-0 grid-cols-[7rem_minmax(0,1fr)] gap-3 text-xs'>
      <span className='text-muted-foreground'>{props.label}</span>
      <span className='min-w-0 font-mono break-all'>{props.value}</span>
    </div>
  )
}

export function RequestDetailDialog(props: RequestDetailDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const query = useQuery({
    queryKey: ['request-detail', props.requestId],
    queryFn: () => getRequestDetail(props.requestId),
    enabled: props.open && props.requestId !== '',
    retry: false,
  })
  const detail = query.data?.data?.detail
  const routing = query.data?.data?.payload?.routing
  const requestPayloadText = useMemo(() => {
    const payload = query.data?.data?.payload
    if (!payload) return ''
    const requestPayload = {
      headers: payload.headers,
      query: payload.query,
      body: payload.body,
    }
    return Object.values(requestPayload).some((value) => value !== undefined)
      ? JSON.stringify(requestPayload, null, 2)
      : ''
  }, [query.data?.data?.payload])
  const response = query.data?.data?.payload?.response
  const responseBodyText = useMemo(() => {
    if (response?.body === undefined || response.body === null) return ''
    return typeof response.body === 'string'
      ? response.body
      : JSON.stringify(response.body, null, 2)
  }, [response?.body])

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Request Details')}
      description={t(
        'Sanitized request and response content with final routing diagnostics.'
      )}
      contentClassName='min-w-0 overflow-hidden sm:max-w-4xl'
      contentHeight='min(78dvh, 800px)'
      bodyClassName='pr-2 sm:pr-4'
    >
      {query.isLoading && (
        <div className='text-muted-foreground flex min-h-48 items-center justify-center gap-2 text-sm'>
          <Loader2 className='size-4 animate-spin' aria-hidden='true' />
          {t('Loading request details...')}
        </div>
      )}
      {query.isError && (
        <div className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'>
          {t('Request details are unavailable or have expired.')}
        </div>
      )}
      {detail && (
        <div className='min-w-0 space-y-4'>
          <div className='space-y-1.5 rounded-md border p-3'>
            <div className='mb-2 flex items-center justify-between gap-3'>
              <span className='text-sm font-medium'>{t('Final result')}</span>
              <StatusBadge
                label={detail.outcome === 'failed' ? t('Failed') : t('Success')}
                variant={detail.outcome === 'failed' ? 'red' : 'green'}
                copyable={false}
              />
            </div>
            <MetadataRow label={t('Request ID')} value={detail.request_id} />
            <MetadataRow
              label={t('Time')}
              value={formatTimestampToDate(detail.created_at, 'seconds')}
            />
            <MetadataRow
              label={t('Request')}
              value={`${detail.method} ${detail.path}`}
            />
            <MetadataRow label={t('User')} value={detail.username || '-'} />
            <MetadataRow label={t('Model')} value={detail.model_name || '-'} />
            <MetadataRow label={t('Group')} value={detail.group || '-'} />
            <MetadataRow
              label={t('Status Code')}
              value={String(detail.status_code)}
            />
            {detail.error_code && (
              <MetadataRow label={t('Error Code')} value={detail.error_code} />
            )}
            {detail.error_message && (
              <MetadataRow
                label={t('Error Message')}
                value={detail.error_message}
              />
            )}
            {detail.body_omitted_reason && (
              <MetadataRow
                label={t('Request Body')}
                value={t('Not stored: {{reason}}', {
                  reason: detail.body_omitted_reason,
                })}
              />
            )}
          </div>

          {routing && (
            <div className='min-w-0 space-y-2'>
              <span className='text-sm font-medium'>
                {t('Routing diagnostics')}
              </span>
              <RoutingDiagnostics routing={routing} />
            </div>
          )}

          {requestPayloadText && (
            <div className='min-w-0 space-y-2'>
              <div className='flex items-center justify-between gap-3'>
                <span className='text-sm font-medium'>{t('Request')}</span>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  onClick={() => copyToClipboard(requestPayloadText)}
                  title={t('Copy to clipboard')}
                  aria-label={t('Copy to clipboard')}
                >
                  {copiedText === requestPayloadText ? (
                    <Check className='size-4 text-green-600' />
                  ) : (
                    <Copy className='size-4' />
                  )}
                </Button>
              </div>
              <pre className='bg-muted/40 max-h-[28rem] min-w-0 overflow-auto rounded-md border p-3 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'>
                {requestPayloadText}
              </pre>
            </div>
          )}

          <div className='min-w-0 space-y-2'>
            <div className='flex items-center justify-between gap-3'>
              <span className='text-sm font-medium'>{t('Response')}</span>
              {responseBodyText && (
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  onClick={() => copyToClipboard(responseBodyText)}
                  title={t('Copy to clipboard')}
                  aria-label={t('Copy to clipboard')}
                >
                  {copiedText === responseBodyText ? (
                    <Check className='size-4 text-green-600' />
                  ) : (
                    <Copy className='size-4' />
                  )}
                </Button>
              )}
            </div>
            {response ? (
              <>
                <div className='space-y-1.5'>
                  <MetadataRow
                    label={t('Status Code')}
                    value={String(response.status_code)}
                  />
                  <MetadataRow
                    label='Content-Type'
                    value={response.content_type || '-'}
                  />
                  <MetadataRow
                    label={t('Size')}
                    value={`${response.body_size.toLocaleString()} bytes`}
                  />
                </div>
                {response.truncated && (
                  <div className='rounded-md border border-amber-500/40 bg-amber-500/5 p-3 text-xs text-amber-700 dark:text-amber-300'>
                    {t(
                      'The response body exceeded the capture limit and was truncated.'
                    )}
                  </div>
                )}
                {responseBodyText ? (
                  <pre className='bg-muted/40 max-h-[28rem] min-w-0 overflow-auto rounded-md border p-3 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'>
                    {responseBodyText}
                  </pre>
                ) : (
                  <div className='text-muted-foreground text-xs'>
                    {response.omitted_reason
                      ? t('Not stored: {{reason}}', {
                          reason: response.omitted_reason,
                        })
                      : t('The response body was empty.')}
                  </div>
                )}
              </>
            ) : (
              <div className='text-muted-foreground text-xs'>
                {t('The response body was not captured for this record.')}
              </div>
            )}
          </div>
        </div>
      )}
    </Dialog>
  )
}
