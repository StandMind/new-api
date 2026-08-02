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
import { AlertTriangle, Info, Link2 } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { resolveOpenLuxSelection } from './selection'
import type { OpenLuxChange, OpenLuxChangeKind, OpenLuxPreview } from './types'

type PreviewPanelProps = {
  preview: OpenLuxPreview
  selectedRoots: Set<string>
  onSelectedRootsChange: (next: Set<string>) => void
  disabled: boolean
}

type ChangeGroup = {
  key: string
  sourceGroup: string
  localGroup: string
  changes: OpenLuxChange[]
}

function groupPreviewChanges(changes: OpenLuxChange[]) {
  const groups = new Map<string, ChangeGroup>()
  for (const change of changes) {
    const sourceGroup = change.source_group ?? ''
    const localGroup = change.local_group ?? ''
    const key = sourceGroup
      ? `${sourceGroup}\u0000${localGroup}`
      : '__global_billing__'
    const current = groups.get(key) ?? {
      key,
      sourceGroup,
      localGroup,
      changes: [],
    }
    current.changes.push(change)
    groups.set(key, current)
  }
  return [...groups.values()].sort((left, right) => {
    if (left.key === '__global_billing__') return -1
    if (right.key === '__global_billing__') return 1
    return left.key.localeCompare(right.key)
  })
}

const changeLabels: Record<OpenLuxChangeKind, string> = {
  model_billing_update: 'Model billing',
  group_model_price_update: 'Group model price',
  group_model_remove: 'Remove model association',
  source_group_remove: 'Remove source group',
}

function displayValue(change: OpenLuxChange, target: boolean) {
  const price = target ? change.target_price : change.current_price
  const value = target ? change.target_value : change.current_value
  if (price) return `$${price}`
  if (value) return value
  return '—'
}

export function OpenLuxPreviewPanel({
  preview,
  selectedRoots,
  onSelectedRootsChange,
  disabled,
}: PreviewPanelProps) {
  const { t } = useTranslation()
  const groups = useMemo(
    () => groupPreviewChanges(preview.changes),
    [preview.changes]
  )
  const selected = useMemo(
    () => resolveOpenLuxSelection(preview, selectedRoots),
    [preview, selectedRoots]
  )

  const toggleRoot = (change: OpenLuxChange, checked: boolean) => {
    const next = new Set(selectedRoots)
    if (checked) next.add(change.id)
    else next.delete(change.id)
    onSelectedRootsChange(next)
  }

  const toggleGroup = (group: ChangeGroup, checked: boolean) => {
    const next = new Set(selectedRoots)
    for (const change of group.changes) {
      if (!change.actionable) continue
      if (checked) next.add(change.id)
      else next.delete(change.id)
    }
    onSelectedRootsChange(next)
  }

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-center gap-2'>
        <Badge variant='secondary'>
          {t('{{count}} executable', { count: preview.summary.actionable })}
        </Badge>
        <Badge
          variant={preview.summary.blocked > 0 ? 'destructive' : 'outline'}
        >
          {t('{{count}} blocked', { count: preview.summary.blocked })}
        </Badge>
        <Badge variant='outline'>
          {t('{{count}} notices', { count: preview.summary.notices })}
        </Badge>
        <span className='text-muted-foreground ml-auto text-xs tabular-nums'>
          {new Date(preview.fetched_at * 1000).toLocaleString()}
        </span>
      </div>

      {preview.notices.length > 0 && (
        <details className='group rounded-md border'>
          <summary className='flex cursor-pointer list-none items-center gap-2 px-3 py-2 font-medium'>
            <Info className='text-muted-foreground size-4' />
            {t('OpenLux notices')}
            <Badge variant='outline'>{preview.notices.length}</Badge>
          </summary>
          <div className='max-h-72 divide-y overflow-y-auto border-t'>
            {preview.notices.map((notice) => (
              <div
                key={`${notice.kind}-${notice.source_group ?? ''}-${notice.local_group ?? ''}-${notice.model ?? ''}-${notice.message}`}
                className='grid gap-1 px-3 py-2 sm:grid-cols-[minmax(10rem,0.7fr)_minmax(12rem,1fr)_minmax(16rem,2fr)]'
              >
                <Badge variant='outline'>{notice.kind}</Badge>
                <span className='break-all'>
                  {[notice.source_group, notice.local_group, notice.model]
                    .filter(Boolean)
                    .join(' / ') || '—'}
                </span>
                <span className='text-muted-foreground whitespace-normal'>
                  {notice.message}
                </span>
              </div>
            ))}
          </div>
        </details>
      )}

      {groups.length === 0 && (
        <div className='text-muted-foreground border-y px-3 py-10 text-center'>
          {t('No OpenLux price changes')}
        </div>
      )}

      {groups.map((group) => {
        const actionable = group.changes.filter((change) => change.actionable)
        const selectedCount = actionable.filter((change) =>
          selected.has(change.id)
        ).length
        const allSelected =
          actionable.length > 0 && selectedCount === actionable.length
        const mixedCount = Math.max(
          0,
          ...group.changes.map((change) => change.mixed_channel_count ?? 0)
        )
        return (
          <section
            key={group.key}
            className='overflow-hidden rounded-md border'
          >
            <div className='bg-muted/40 flex flex-wrap items-center gap-2 border-b px-3 py-2'>
              <Checkbox
                checked={allSelected}
                indeterminate={selectedCount > 0 && !allSelected}
                disabled={disabled || actionable.length === 0}
                onCheckedChange={(checked) =>
                  toggleGroup(group, checked === true)
                }
                aria-label={t('Select group changes')}
              />
              <span className='font-medium break-all'>
                {group.sourceGroup || t('Global model billing')}
              </span>
              {group.localGroup && (
                <>
                  <span className='text-muted-foreground'>→</span>
                  <span className='break-all'>{group.localGroup}</span>
                </>
              )}
              <Badge variant='outline'>{group.changes.length}</Badge>
              {mixedCount > 0 && (
                <div className='text-destructive ml-auto flex items-center gap-1 text-sm font-medium'>
                  <AlertTriangle className='size-4' />
                  {t('Price also applies to {{count}} other channels', {
                    count: mixedCount,
                  })}
                </div>
              )}
            </div>

            <Table className='min-w-[980px]'>
              <TableHeader>
                <TableRow>
                  <TableHead className='w-10' />
                  <TableHead>{t('Change')}</TableHead>
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Current')}</TableHead>
                  <TableHead>{t('Target')}</TableHead>
                  <TableHead>{t('Change rate')}</TableHead>
                  <TableHead className='min-w-64'>{t('Impact')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {group.changes.map((change) => {
                  const autoLinked =
                    selected.has(change.id) && !selectedRoots.has(change.id)
                  let tooltipLabel = t('Blocked change')
                  if (change.actionable) tooltipLabel = t('Select change')
                  if (autoLinked) {
                    tooltipLabel = t('Selected as a required dependency')
                  }
                  return (
                    <TableRow
                      key={change.id}
                      data-state={
                        selected.has(change.id) ? 'selected' : undefined
                      }
                    >
                      <TableCell>
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <span className='inline-flex'>
                                <Checkbox
                                  checked={selected.has(change.id)}
                                  disabled={
                                    disabled || !change.actionable || autoLinked
                                  }
                                  onCheckedChange={(checked) =>
                                    toggleRoot(change, checked === true)
                                  }
                                  aria-label={t('Select change')}
                                />
                              </span>
                            }
                          />
                          <TooltipContent>{tooltipLabel}</TooltipContent>
                        </Tooltip>
                      </TableCell>
                      <TableCell className='whitespace-normal'>
                        <div className='flex flex-wrap items-center gap-1.5'>
                          <Badge
                            variant={
                              change.destructive ? 'destructive' : 'secondary'
                            }
                          >
                            {t(changeLabels[change.kind])}
                          </Badge>
                          {autoLinked && (
                            <Badge variant='outline'>
                              <Link2 className='size-3' />
                              {t('Required')}
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className='max-w-56 whitespace-normal'>
                        <span className='font-medium break-all'>
                          {change.model || '—'}
                        </span>
                      </TableCell>
                      <TableCell className='max-w-48 whitespace-normal'>
                        <span className='break-all tabular-nums'>
                          {displayValue(change, false)}
                        </span>
                      </TableCell>
                      <TableCell className='max-w-48 whitespace-normal'>
                        <div className='break-all tabular-nums'>
                          {displayValue(change, true)}
                        </div>
                        {change.price_unit && (
                          <div className='text-muted-foreground text-xs'>
                            {change.price_unit}
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        {change.percent_change
                          ? `${Number(change.percent_change) > 0 ? '+' : ''}${change.percent_change}%`
                          : '—'}
                      </TableCell>
                      <TableCell className='whitespace-normal'>
                        {(change.blocked_reasons ?? []).length > 0 && (
                          <div className='text-destructive flex gap-1.5'>
                            <AlertTriangle className='mt-0.5 size-4 shrink-0' />
                            <span>
                              {(change.blocked_reasons ?? []).join('；')}
                            </span>
                          </div>
                        )}
                        {(change.details ?? []).length > 0 && (
                          <div className='space-y-1'>
                            {(change.details ?? []).map((detail) => (
                              <div key={detail.field} className='break-all'>
                                <span className='font-medium'>
                                  {detail.field}
                                </span>
                                <span className='text-muted-foreground'>
                                  :{' '}
                                </span>
                                <span className='tabular-nums'>
                                  {detail.current || '—'} →{' '}
                                  {detail.target || '—'}
                                </span>
                              </div>
                            ))}
                          </div>
                        )}
                        {(change.affected_groups ?? []).length > 0 && (
                          <div className='text-muted-foreground mt-1 break-all'>
                            {t('Affects groups: {{groups}}', {
                              groups: (change.affected_groups ?? []).join(', '),
                            })}
                          </div>
                        )}
                        {change.remove_mode && (
                          <Badge variant='outline' className='mt-1'>
                            {t(change.remove_mode)}
                          </Badge>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </section>
        )
      })}
    </div>
  )
}
