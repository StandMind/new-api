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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ChevronsDown,
  ChevronsUp,
  Loader2,
  Network,
  Plus,
  RefreshCw,
  Search,
} from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { deleteGroupModelRoute, listGroupModelRoutes } from '../api'
import type { GroupModelRouteListItem } from '../types'
import { safeJsonParse } from '../utils/json-parser'
import {
  GroupModelRouteEditorSheet,
  type GroupModelRouteEditorTarget,
} from './group-model-route-editor-sheet'
import { GroupModelRouteTable } from './group-model-route-table'
import {
  getGroupModelRouteKey,
  groupModelRouteMatchesSearch,
} from './group-model-route-utils'

type GroupModelRouteEditorProps = {
  groupRatio: string
}

const groupModelRoutesQueryKey = ['group-model-routes'] as const

export function GroupModelRouteEditor(props: GroupModelRouteEditorProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const savedRoutesQuery = useQuery({
    queryKey: groupModelRoutesQueryKey,
    queryFn: listGroupModelRoutes,
  })
  const savedRoutes = useMemo(
    () =>
      savedRoutesQuery.data?.success ? (savedRoutesQuery.data.data ?? []) : [],
    [savedRoutesQuery.data]
  )
  const groupOptions = useMemo(() => {
    const ratios = safeJsonParse<Record<string, number>>(props.groupRatio, {
      fallback: {},
      silent: true,
    })
    return [
      ...new Set([
        ...Object.keys(ratios),
        ...savedRoutes.map((route) => route.group),
      ]),
    ].sort()
  }, [props.groupRatio, savedRoutes])
  const routeGroupOptions = useMemo(
    () => [...new Set(savedRoutes.map((route) => route.group))].sort(),
    [savedRoutes]
  )
  const [search, setSearch] = useState('')
  const [groupFilter, setGroupFilter] = useState('')
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set())
  const [editorTarget, setEditorTarget] =
    useState<GroupModelRouteEditorTarget | null>(null)
  const [restoreTarget, setRestoreTarget] =
    useState<GroupModelRouteListItem | null>(null)
  const [restoringKey, setRestoringKey] = useState('')

  const filteredRoutes = useMemo(
    () =>
      savedRoutes.filter(
        (route) =>
          (!groupFilter || route.group === groupFilter) &&
          groupModelRouteMatchesSearch(route, search)
      ),
    [groupFilter, savedRoutes, search]
  )
  const allVisibleExpanded =
    filteredRoutes.length > 0 &&
    filteredRoutes.every((route) =>
      expandedKeys.has(getGroupModelRouteKey(route.group, route.model))
    )

  const toggleRoute = (route: GroupModelRouteListItem) => {
    const key = getGroupModelRouteKey(route.group, route.model)
    setExpandedKeys((current) => {
      const next = new Set(current)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const toggleAllVisible = () => {
    setExpandedKeys((current) => {
      const next = new Set(current)
      for (const route of filteredRoutes) {
        const key = getGroupModelRouteKey(route.group, route.model)
        if (allVisibleExpanded) {
          next.delete(key)
        } else {
          next.add(key)
        }
      }
      return next
    })
  }

  const refreshRoutes = async () => {
    await savedRoutesQuery.refetch()
  }

  const handleSaved = async (group: string, model: string) => {
    const key = getGroupModelRouteKey(group, model)
    setExpandedKeys((current) => new Set(current).add(key))
    setEditorTarget(null)
    await queryClient.invalidateQueries({ queryKey: groupModelRoutesQueryKey })
  }

  const handleRestored = async (group: string, model: string) => {
    const key = getGroupModelRouteKey(group, model)
    setExpandedKeys((current) => {
      const next = new Set(current)
      next.delete(key)
      return next
    })
    setEditorTarget(null)
    await queryClient.invalidateQueries({ queryKey: groupModelRoutesQueryKey })
  }

  const restoreRoute = async () => {
    if (!restoreTarget) return
    const key = getGroupModelRouteKey(restoreTarget.group, restoreTarget.model)
    setRestoringKey(key)
    try {
      const response = await deleteGroupModelRoute(
        restoreTarget.group,
        restoreTarget.model
      )
      if (!response.success) {
        toast.error(
          response.message || t('Failed to restore the inherited route')
        )
        return
      }
      toast.success(t('Inherited channel route restored'))
      setRestoreTarget(null)
      await handleRestored(restoreTarget.group, restoreTarget.model)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to restore the inherited route')
      )
    } finally {
      setRestoringKey('')
    }
  }

  const clearFilters = () => {
    setSearch('')
    setGroupFilter('')
  }
  const switchToEdit = useCallback((group: string, model: string) => {
    setEditorTarget({ mode: 'edit', group, model })
  }, [])

  let routeListContent: React.ReactNode
  if (savedRoutesQuery.isPending) {
    routeListContent = (
      <div className='flex min-h-48 items-center justify-center rounded-lg border'>
        <Loader2 className='animate-spin' />
      </div>
    )
  } else if (
    savedRoutesQuery.isError ||
    savedRoutesQuery.data?.success === false
  ) {
    routeListContent = (
      <Empty className='min-h-48 border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Network />
          </EmptyMedia>
          <EmptyTitle>{t('Failed to load group model routes')}</EmptyTitle>
          <EmptyDescription>
            {savedRoutesQuery.data?.message ||
              t('The saved route list could not be loaded.')}
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button
            type='button'
            variant='outline'
            onClick={() => void refreshRoutes()}
          >
            <RefreshCw />
            {t('Retry')}
          </Button>
        </EmptyContent>
      </Empty>
    )
  } else if (savedRoutes.length === 0) {
    routeListContent = (
      <Empty className='min-h-56 border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Network />
          </EmptyMedia>
          <EmptyTitle>{t('No explicit routes')}</EmptyTitle>
          <EmptyDescription>
            {t('Create a route to override inherited Ability priorities.')}
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button
            type='button'
            onClick={() => setEditorTarget({ mode: 'create' })}
          >
            <Plus />
            {t('Create route')}
          </Button>
        </EmptyContent>
      </Empty>
    )
  } else if (filteredRoutes.length === 0) {
    routeListContent = (
      <Empty className='min-h-48 border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Search />
          </EmptyMedia>
          <EmptyTitle>{t('No matching routes')}</EmptyTitle>
          <EmptyDescription>
            {t('No saved route matches the current search and group filter.')}
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button type='button' variant='outline' onClick={clearFilters}>
            {t('Clear filters')}
          </Button>
        </EmptyContent>
      </Empty>
    )
  } else {
    routeListContent = (
      <GroupModelRouteTable
        routes={filteredRoutes}
        expandedKeys={expandedKeys}
        restoringKey={restoringKey}
        onToggle={toggleRoute}
        onEdit={(route) =>
          setEditorTarget({
            mode: 'edit',
            group: route.group,
            model: route.model,
          })
        }
        onRestore={setRestoreTarget}
      />
    )
  }

  return (
    <div className='space-y-5'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
        <div className='space-y-1'>
          <div className='flex items-center gap-2'>
            <h3 className='text-sm font-semibold'>
              {t('Group model channel route')}
            </h3>
            <Badge variant='secondary'>{savedRoutes.length}</Badge>
          </div>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Review every explicit route here, then expand a row to inspect its priority tiers and channel weights.'
            )}
          </p>
        </div>
        <Button
          type='button'
          onClick={() => setEditorTarget({ mode: 'create' })}
        >
          <Plus />
          {t('Create route')}
        </Button>
      </div>

      {savedRoutes.length > 0 && (
        <div className='flex flex-col gap-2 lg:flex-row lg:items-center'>
          <div className='relative min-w-0 flex-1'>
            <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
            <Input
              className='pl-8'
              value={search}
              placeholder={t('Search group, model, channel name or ID')}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
          <NativeSelect
            className='w-full lg:w-48'
            value={groupFilter}
            aria-label={t('Filter by group')}
            onChange={(event) => setGroupFilter(event.target.value)}
          >
            <NativeSelectOption value=''>{t('All groups')}</NativeSelectOption>
            {routeGroupOptions.map((group) => (
              <NativeSelectOption key={group} value={group}>
                {group}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              className='flex-1 lg:flex-none'
              onClick={toggleAllVisible}
              disabled={filteredRoutes.length === 0}
            >
              {allVisibleExpanded ? <ChevronsUp /> : <ChevronsDown />}
              {allVisibleExpanded ? t('Collapse all') : t('Expand all')}
            </Button>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type='button'
                    variant='outline'
                    size='icon'
                    aria-label={t('Refresh')}
                    disabled={savedRoutesQuery.isFetching}
                    onClick={() => void refreshRoutes()}
                  />
                }
              >
                <RefreshCw
                  className={
                    savedRoutesQuery.isFetching ? 'animate-spin' : undefined
                  }
                />
              </TooltipTrigger>
              <TooltipContent>{t('Refresh')}</TooltipContent>
            </Tooltip>
          </div>
        </div>
      )}

      {routeListContent}

      {editorTarget && (
        <GroupModelRouteEditorSheet
          key={
            editorTarget.mode === 'edit'
              ? getGroupModelRouteKey(editorTarget.group, editorTarget.model)
              : 'create'
          }
          open
          target={editorTarget}
          groupOptions={groupOptions}
          savedRoutes={savedRoutes}
          onOpenChange={(open) => {
            if (!open) setEditorTarget(null)
          }}
          onSwitchToEdit={switchToEdit}
          onSaved={handleSaved}
          onRestored={handleRestored}
        />
      )}

      <ConfirmDialog
        open={restoreTarget !== null}
        onOpenChange={(open) => {
          if (!open && !restoringKey) setRestoreTarget(null)
        }}
        title={t('Restore inherited route?')}
        desc={t(
          'This removes the explicit route and resumes using the current Ability priorities and weights.'
        )}
        destructive
        confirmText={t('Restore inherited route')}
        isLoading={Boolean(restoringKey)}
        handleConfirm={() => void restoreRoute()}
      />
    </div>
  )
}
