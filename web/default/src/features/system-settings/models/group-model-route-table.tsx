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
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Pencil,
  RotateCcw,
} from 'lucide-react'
import { Fragment } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

import type { GroupModelRouteListItem } from '../types'
import {
  countGroupModelRouteChannels,
  getGroupModelRouteKey,
} from './group-model-route-utils'

type GroupModelRouteTableProps = {
  routes: GroupModelRouteListItem[]
  expandedKeys: Set<string>
  restoringKey: string
  onToggle: (route: GroupModelRouteListItem) => void
  onEdit: (route: GroupModelRouteListItem) => void
  onRestore: (route: GroupModelRouteListItem) => void
}

export function GroupModelRouteTable(props: GroupModelRouteTableProps) {
  const { t } = useTranslation()

  return (
    <div className='overflow-hidden rounded-lg border'>
      <Table className='min-w-[800px]'>
        <TableHeader>
          <TableRow>
            <TableHead className='w-10'>
              <span className='sr-only'>{t('Expand')}</span>
            </TableHead>
            <TableHead>{t('Group')}</TableHead>
            <TableHead>{t('Model')}</TableHead>
            <TableHead>{t('Route summary')}</TableHead>
            <TableHead>{t('Updated')}</TableHead>
            <TableHead className='w-24 text-right'>
              <span className='sr-only'>{t('Actions')}</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.routes.map((route) => {
            const routeKey = getGroupModelRouteKey(route.group, route.model)
            const expanded = props.expandedKeys.has(routeKey)
            const channelCount = countGroupModelRouteChannels(route)
            return (
              <Fragment key={routeKey}>
                <TableRow aria-expanded={expanded}>
                  <TableCell>
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon-sm'
                            aria-label={
                              expanded ? t('Collapse route') : t('Expand route')
                            }
                            aria-expanded={expanded}
                            onClick={() => props.onToggle(route)}
                          />
                        }
                      >
                        {expanded ? <ChevronDown /> : <ChevronRight />}
                      </TooltipTrigger>
                      <TooltipContent>
                        {expanded ? t('Collapse route') : t('Expand route')}
                      </TooltipContent>
                    </Tooltip>
                  </TableCell>
                  <TableCell>
                    <Badge variant='outline'>{route.group}</Badge>
                  </TableCell>
                  <TableCell className='max-w-64 font-mono'>
                    <span className='block truncate' title={route.model}>
                      {route.model}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className='flex flex-wrap items-center gap-1.5'>
                      <span>
                        {t('{{count}} priority tiers', {
                          count: route.tiers.length,
                        })}
                      </span>
                      <span className='text-muted-foreground'>·</span>
                      <span>
                        {t('{{count}} channels', { count: channelCount })}
                      </span>
                      <span className='text-muted-foreground font-mono text-xs'>
                        {route.tiers
                          .map((tier) => `P${tier.priority}`)
                          .join(' → ')}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className='text-muted-foreground'>
                    {route.updated_at
                      ? new Date(route.updated_at).toLocaleString()
                      : '-'}
                  </TableCell>
                  <TableCell>
                    <div className='flex justify-end gap-1'>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon-sm'
                              aria-label={t('Edit route')}
                              onClick={() => props.onEdit(route)}
                            />
                          }
                        >
                          <Pencil />
                        </TooltipTrigger>
                        <TooltipContent>{t('Edit route')}</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon-sm'
                              aria-label={t('Restore inherited route')}
                              disabled={props.restoringKey === routeKey}
                              onClick={() => props.onRestore(route)}
                            />
                          }
                        >
                          <RotateCcw
                            className={
                              props.restoringKey === routeKey
                                ? 'animate-spin'
                                : undefined
                            }
                          />
                        </TooltipTrigger>
                        <TooltipContent>
                          {t('Restore inherited route')}
                        </TooltipContent>
                      </Tooltip>
                    </div>
                  </TableCell>
                </TableRow>
                {expanded && (
                  <TableRow className='bg-muted/20 hover:bg-muted/20'>
                    <TableCell colSpan={6} className='p-0 whitespace-normal'>
                      <div className='divide-y'>
                        {route.tiers.map((tier) => (
                          <div
                            key={tier.priority}
                            className='grid gap-3 px-4 py-3 md:grid-cols-[8rem_minmax(0,1fr)]'
                          >
                            <div className='flex items-start'>
                              <Badge variant='secondary'>
                                {t('Priority {{priority}}', {
                                  priority: tier.priority,
                                })}
                              </Badge>
                            </div>
                            <div className='flex min-w-0 flex-wrap gap-2'>
                              {tier.channels.map((channel) => (
                                <div
                                  key={channel.channel_id}
                                  className='border-border bg-background flex min-w-0 items-center gap-2 rounded-md border px-2.5 py-1.5'
                                >
                                  {!channel.eligible && (
                                    <Tooltip>
                                      <TooltipTrigger
                                        render={
                                          <AlertTriangle
                                            className='text-destructive size-4 shrink-0'
                                            aria-label={t(
                                              'Currently unavailable'
                                            )}
                                          />
                                        }
                                      />
                                      <TooltipContent>
                                        {t('Currently unavailable')}
                                      </TooltipContent>
                                    </Tooltip>
                                  )}
                                  <span
                                    className='max-w-56 truncate'
                                    title={
                                      channel.channel_name ||
                                      t('Deleted or unavailable channel')
                                    }
                                  >
                                    <span className='text-muted-foreground font-mono'>
                                      #{channel.channel_id}
                                    </span>{' '}
                                    {channel.channel_name ||
                                      t('Deleted or unavailable channel')}
                                  </span>
                                  <Badge
                                    variant={
                                      channel.eligible
                                        ? 'outline'
                                        : 'destructive'
                                    }
                                  >
                                    {channel.eligible
                                      ? t('Weight {{weight}}', {
                                          weight: channel.weight,
                                        })
                                      : t('Unavailable')}
                                  </Badge>
                                </div>
                              ))}
                            </div>
                          </div>
                        ))}
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </Fragment>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
