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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Eye, RefreshCw, Search } from 'lucide-react'
import { useMemo, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

import {
  StaticDataTable,
  type StaticDataTableColumn,
} from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { formatTimestampToDate } from '@/lib/format'

import { getRequestDetails } from '../api'
import type { RequestDetailRecord } from '../types'
import { RequestDetailDialog } from './dialogs/request-detail-dialog'

const PAGE_SIZE = 50

type RequestDetailFilters = {
  requestId: string
  username: string
  model: string
  outcome: 'all' | 'success' | 'failed'
}

const EMPTY_FILTERS: RequestDetailFilters = {
  requestId: '',
  username: '',
  model: '',
  outcome: 'all',
}

export function RequestDetailsTable() {
  const { t } = useTranslation()
  const [draftFilters, setDraftFilters] =
    useState<RequestDetailFilters>(EMPTY_FILTERS)
  const [filters, setFilters] = useState<RequestDetailFilters>(EMPTY_FILTERS)
  const [page, setPage] = useState(1)
  const [selectedRequestId, setSelectedRequestId] = useState('')
  const query = useQuery({
    queryKey: ['request-details', page, filters],
    queryFn: async () => {
      const result = await getRequestDetails({
        p: page,
        page_size: PAGE_SIZE,
        request_id: filters.requestId || undefined,
        username: filters.username || undefined,
        model_name: filters.model || undefined,
        outcome: filters.outcome === 'all' ? undefined : filters.outcome,
      })
      if (!result.success) {
        throw new Error(result.message || t('Failed to load request details'))
      }
      return result.data
    },
    placeholderData: keepPreviousData,
  })

  const columns = useMemo<StaticDataTableColumn<RequestDetailRecord>[]>(
    () => [
      {
        id: 'created_at',
        header: t('Time'),
        className: 'min-w-40',
        cell: (row) => formatTimestampToDate(row.created_at, 'seconds'),
      },
      {
        id: 'outcome',
        header: t('Result'),
        className: 'w-24',
        cell: (row) => (
          <StatusBadge
            label={row.outcome === 'failed' ? t('Failed') : t('Success')}
            variant={row.outcome === 'failed' ? 'red' : 'green'}
            copyable={false}
          />
        ),
      },
      {
        id: 'username',
        header: t('User'),
        className: 'min-w-28',
        cell: (row) => row.username || `#${row.user_id}`,
      },
      {
        id: 'model',
        header: t('Model'),
        className: 'min-w-40',
        cellClassName: 'font-mono text-xs',
        cell: (row) => row.model_name || '-',
      },
      {
        id: 'request',
        header: t('Request'),
        className: 'min-w-64',
        cellClassName: 'font-mono text-xs',
        cell: (row) => `${row.method} ${row.path}`,
      },
      {
        id: 'status',
        header: t('Status Code'),
        className: 'w-24',
        cellClassName: 'font-mono',
        cell: (row) => row.status_code,
      },
      {
        id: 'request_id',
        header: t('Request ID'),
        className: 'min-w-56',
        cellClassName: 'font-mono text-xs',
        cell: (row) => row.request_id,
      },
      {
        id: 'actions',
        header: t('Actions'),
        className: 'w-16 text-right',
        cellClassName: 'text-right',
        cell: (row) => (
          <Button
            type='button'
            variant='ghost'
            size='icon'
            onClick={() => setSelectedRequestId(row.request_id)}
            title={t('View request details')}
            aria-label={t('View request details')}
          >
            <Eye className='size-4' aria-hidden='true' />
          </Button>
        ),
      },
    ],
    [t]
  )

  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const applyFilters = (event: FormEvent) => {
    event.preventDefault()
    setPage(1)
    setFilters(draftFilters)
  }

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <form onSubmit={applyFilters} className='flex flex-wrap items-end gap-2'>
        <div className='min-w-52 flex-1'>
          <Input
            value={draftFilters.requestId}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                requestId: event.target.value,
              }))
            }
            placeholder={t('Request ID')}
          />
        </div>
        <div className='min-w-36 flex-1 sm:max-w-52'>
          <Input
            value={draftFilters.username}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                username: event.target.value,
              }))
            }
            placeholder={t('Username')}
          />
        </div>
        <div className='min-w-40 flex-1 sm:max-w-64'>
          <Input
            value={draftFilters.model}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                model: event.target.value,
              }))
            }
            placeholder={t('Model')}
          />
        </div>
        <Select
          value={draftFilters.outcome}
          onValueChange={(value) =>
            setDraftFilters((current) => ({
              ...current,
              outcome: value as RequestDetailFilters['outcome'],
            }))
          }
        >
          <SelectTrigger className='w-36'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='all'>{t('All results')}</SelectItem>
            <SelectItem value='failed'>{t('Failed')}</SelectItem>
            <SelectItem value='success'>{t('Success')}</SelectItem>
          </SelectContent>
        </Select>
        <Button type='submit'>
          <Search className='size-4' aria-hidden='true' />
          {t('Search')}
        </Button>
        <Button
          type='button'
          variant='outline'
          size='icon'
          onClick={() => query.refetch()}
          disabled={query.isFetching}
          title={t('Refresh')}
          aria-label={t('Refresh')}
        >
          <RefreshCw
            className={query.isFetching ? 'size-4 animate-spin' : 'size-4'}
            aria-hidden='true'
          />
        </Button>
      </form>

      {query.isError && (
        <div className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'>
          {query.error instanceof Error
            ? query.error.message
            : t('Failed to load request details')}
        </div>
      )}

      <div className='min-h-0 flex-1 overflow-auto'>
        <StaticDataTable
          columns={columns}
          data={query.data?.items ?? []}
          getRowKey={(row) => row.request_id}
          emptyContent={
            query.isLoading
              ? t('Loading request details...')
              : t('No request details found')
          }
          tableClassName='min-w-[76rem] text-[13px]'
        />
      </div>

      <div className='flex items-center justify-between gap-3 text-sm'>
        <span className='text-muted-foreground'>
          {t('{{count}} request details', { count: total })}
        </span>
        <div className='flex items-center gap-2'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={page <= 1}
            onClick={() => setPage((current) => Math.max(1, current - 1))}
          >
            {t('Previous')}
          </Button>
          <span className='min-w-20 text-center tabular-nums'>
            {page} / {totalPages}
          </span>
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={page >= totalPages}
            onClick={() => setPage((current) => current + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>

      <RequestDetailDialog
        requestId={selectedRequestId}
        open={selectedRequestId !== ''}
        onOpenChange={(open) => {
          if (!open) setSelectedRequestId('')
        }}
      />
    </div>
  )
}
