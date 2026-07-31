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
import { Loader2, RotateCcw, Save } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import {
  deleteGroupModelRoute,
  getChannelModels,
  getGroupModelRoute,
  saveGroupModelRoute,
} from '../api'
import type { GroupModelRouteListItem } from '../types'
import { GroupModelRouteTierEditor } from './group-model-route-tier-editor'
import {
  cloneGroupModelRouteTiers,
  getGroupModelRouteKey,
  serializeGroupModelRouteTiers,
  type EditableGroupModelRouteTier,
} from './group-model-route-utils'

export type GroupModelRouteEditorTarget =
  | { mode: 'create' }
  | { mode: 'edit'; group: string; model: string }

type GroupModelRouteEditorSheetProps = {
  open: boolean
  target: GroupModelRouteEditorTarget
  groupOptions: string[]
  savedRoutes: GroupModelRouteListItem[]
  onOpenChange: (open: boolean) => void
  onSwitchToEdit: (group: string, model: string) => void
  onSaved: (group: string, model: string) => Promise<void>
  onRestored: (group: string, model: string) => Promise<void>
}

export function GroupModelRouteEditorSheet(
  props: GroupModelRouteEditorSheetProps
) {
  const { t } = useTranslation()
  const onSwitchToEdit = props.onSwitchToEdit
  const editing = props.target.mode === 'edit'
  const editGroup = props.target.mode === 'edit' ? props.target.group : ''
  const editModel = props.target.mode === 'edit' ? props.target.model : ''
  const [draftGroup, setDraftGroup] = useState(editGroup)
  const [draftModel, setDraftModel] = useState(editModel)
  const [committedModel, setCommittedModel] = useState(editModel)
  const [tiers, setTiers] = useState<EditableGroupModelRouteTier[]>([])
  const [loadedKey, setLoadedKey] = useState('')
  const [initialSnapshot, setInitialSnapshot] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [isRestoring, setIsRestoring] = useState(false)
  const [discardConfirmOpen, setDiscardConfirmOpen] = useState(false)
  const [restoreConfirmOpen, setRestoreConfirmOpen] = useState(false)

  const group = editing ? editGroup : draftGroup.trim()
  const model = editing ? editModel : committedModel.trim()
  const routeKey = group && model ? getGroupModelRouteKey(group, model) : ''
  const routeQuery = useQuery({
    queryKey: ['group-model-route', group, model],
    queryFn: () => getGroupModelRoute(group, model),
    enabled: Boolean(group && model),
  })
  const modelsQuery = useQuery({
    queryKey: ['channel-models'],
    queryFn: getChannelModels,
    staleTime: 60_000,
  })
  const modelOptions = useMemo(() => {
    if (!modelsQuery.data?.success || !modelsQuery.data.data) return []
    return [
      ...new Set(modelsQuery.data.data.map((item) => item.id).filter(Boolean)),
    ].sort()
  }, [modelsQuery.data])
  const savedRoute = useMemo(
    () =>
      props.savedRoutes.find(
        (route) => route.group === group && route.model === model
      ),
    [group, model, props.savedRoutes]
  )
  const savedChannelNames = useMemo(() => {
    const names = new Map<number, string>()
    for (const tier of savedRoute?.tiers ?? []) {
      for (const channel of tier.channels) {
        names.set(channel.channel_id, channel.channel_name)
      }
    }
    return names
  }, [savedRoute])
  const candidates = useMemo(
    () =>
      routeQuery.data?.success ? (routeQuery.data.data?.candidates ?? []) : [],
    [routeQuery.data]
  )
  const candidateIDs = useMemo(
    () => new Set(candidates.map((candidate) => candidate.channel_id)),
    [candidates]
  )
  const routeLoaded = Boolean(routeKey && loadedKey === routeKey)
  const currentSnapshot = serializeGroupModelRouteTiers(tiers)
  const dirty = routeLoaded && currentSnapshot !== initialSnapshot
  const explicit = routeQuery.data?.success
    ? Boolean(routeQuery.data.data?.explicit)
    : false

  useEffect(() => {
    if (editing) {
      setCommittedModel(editModel)
      return
    }
    const timeout = window.setTimeout(() => {
      setCommittedModel(draftModel.trim())
    }, 350)
    return () => window.clearTimeout(timeout)
  }, [draftModel, editModel, editing])

  useEffect(() => {
    setLoadedKey('')
    setInitialSnapshot('')
    setTiers([])
  }, [routeKey])

  useEffect(() => {
    if (
      !routeKey ||
      loadedKey === routeKey ||
      !routeQuery.data?.success ||
      !routeQuery.data.data
    ) {
      return
    }
    const clonedTiers = cloneGroupModelRouteTiers(
      routeQuery.data.data.route.tiers
    )
    setTiers(clonedTiers)
    setInitialSnapshot(serializeGroupModelRouteTiers(clonedTiers))
    setLoadedKey(routeKey)
  }, [loadedKey, routeKey, routeQuery.data])

  useEffect(() => {
    if (editing || !group || !model || !savedRoute) return
    onSwitchToEdit(group, model)
  }, [editing, group, model, onSwitchToEdit, savedRoute])

  const requestOpenChange = (open: boolean) => {
    if (!open && dirty) {
      setDiscardConfirmOpen(true)
      return
    }
    props.onOpenChange(open)
  }

  const validate = () => {
    if (!routeLoaded) {
      toast.error(t('Wait for the route to finish loading'))
      return false
    }
    if (tiers.length === 0) {
      toast.error(t('Add at least one priority tier'))
      return false
    }

    const priorities = new Set<number>()
    for (const tier of tiers) {
      if (!Number.isInteger(tier.priority) || priorities.has(tier.priority)) {
        toast.error(t('Every priority tier must use a unique integer'))
        return false
      }
      priorities.add(tier.priority)
      if (tier.channels.length === 0) {
        toast.error(t('Every priority tier must contain at least one channel'))
        return false
      }
      if (
        tier.channels.some(
          (channel) => !Number.isInteger(channel.weight) || channel.weight < 0
        )
      ) {
        toast.error(t('Channel weights must be non-negative integers'))
        return false
      }
      if (
        tier.channels.some((channel) => !candidateIDs.has(channel.channel_id))
      ) {
        toast.error(
          t(
            'This route contains unavailable channels. Remove or replace them before saving.'
          )
        )
        return false
      }
    }
    return true
  }

  const saveRoute = async () => {
    if (!validate()) return
    setIsSaving(true)
    try {
      const response = await saveGroupModelRoute({
        group,
        model,
        tiers: tiers.map((tier) => ({
          priority: tier.priority,
          channels: tier.channels,
        })),
      })
      if (!response.success) {
        toast.error(response.message || t('Failed to save group model route'))
        return
      }
      toast.success(t('Group model route saved'))
      await props.onSaved(group, model)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save group model route')
      )
    } finally {
      setIsSaving(false)
    }
  }

  const restoreRoute = async () => {
    setIsRestoring(true)
    try {
      const response = await deleteGroupModelRoute(group, model)
      if (!response.success) {
        toast.error(
          response.message || t('Failed to restore the inherited route')
        )
        return
      }
      toast.success(t('Inherited channel route restored'))
      setRestoreConfirmOpen(false)
      await props.onRestored(group, model)
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to restore the inherited route')
      )
    } finally {
      setIsRestoring(false)
    }
  }

  let routeContent: React.ReactNode
  if (!group || !model) {
    routeContent = (
      <div className='text-muted-foreground flex min-h-40 items-center justify-center text-sm'>
        {t('Select a group and enter an exact model name')}
      </div>
    )
  } else if (routeQuery.isPending || !routeLoaded) {
    routeContent = (
      <div className='flex min-h-40 items-center justify-center'>
        <Loader2 className='animate-spin' />
      </div>
    )
  } else if (routeQuery.isError || routeQuery.data?.success === false) {
    routeContent = (
      <div className='flex min-h-40 flex-col items-center justify-center gap-3 text-center'>
        <p className='text-destructive text-sm'>
          {routeQuery.data?.message || t('Failed to load group model route')}
        </p>
        <Button
          type='button'
          variant='outline'
          onClick={() => void routeQuery.refetch()}
        >
          {t('Retry')}
        </Button>
      </div>
    )
  } else {
    routeContent = (
      <div className='space-y-4'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant={explicit ? 'default' : 'secondary'}>
            {explicit ? t('Explicit route') : t('Inherited ability route')}
          </Badge>
          <span className='text-muted-foreground text-xs'>
            {t('{{count}} eligible channels', { count: candidates.length })}
          </span>
        </div>
        <GroupModelRouteTierEditor
          tiers={tiers}
          candidates={candidates}
          savedChannelNames={savedChannelNames}
          onChange={setTiers}
        />
      </div>
    )
  }

  return (
    <>
      <Sheet open={props.open} onOpenChange={requestOpenChange}>
        <SheetContent
          side='right'
          className={sideDrawerContentClassName('sm:max-w-3xl')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>
              {editing
                ? t('Edit group model route')
                : t('Create group model route')}
            </SheetTitle>
            <SheetDescription>
              {editing
                ? `${editGroup} / ${editModel}`
                : t('Configure an explicit route for a group and model.')}
            </SheetDescription>
          </SheetHeader>

          <div className={sideDrawerFormClassName()}>
            {editing ? (
              <div className='flex flex-wrap gap-2'>
                <Badge variant='outline'>{editGroup}</Badge>
                <Badge variant='secondary' className='font-mono'>
                  {editModel}
                </Badge>
              </div>
            ) : (
              <div className='grid gap-3 sm:grid-cols-2'>
                <label className='space-y-1.5 text-sm font-medium'>
                  <span>{t('Group')}</span>
                  <NativeSelect
                    className='w-full'
                    value={draftGroup}
                    onChange={(event) => setDraftGroup(event.target.value)}
                  >
                    <NativeSelectOption value=''>
                      {t('Select a group')}
                    </NativeSelectOption>
                    {props.groupOptions.map((option) => (
                      <NativeSelectOption key={option} value={option}>
                        {option}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </label>
                <label className='space-y-1.5 text-sm font-medium'>
                  <span>{t('Model')}</span>
                  <Input
                    value={draftModel}
                    list='group-model-route-models'
                    placeholder={t('Enter an exact model name')}
                    onChange={(event) => setDraftModel(event.target.value)}
                  />
                  <datalist id='group-model-route-models'>
                    {modelOptions.map((option) => (
                      <option key={option} value={option} />
                    ))}
                  </datalist>
                </label>
              </div>
            )}
            {routeContent}
          </div>

          <SheetFooter className={sideDrawerFooterClassName()}>
            <Button
              type='button'
              variant='outline'
              onClick={() => requestOpenChange(false)}
              disabled={isSaving || isRestoring}
            >
              {t('Cancel')}
            </Button>
            {editing && explicit && (
              <Button
                type='button'
                variant='outline'
                onClick={() => setRestoreConfirmOpen(true)}
                disabled={isSaving || isRestoring}
              >
                {isRestoring ? (
                  <Loader2 className='animate-spin' />
                ) : (
                  <RotateCcw />
                )}
                {t('Restore inherited route')}
              </Button>
            )}
            <Button
              type='button'
              onClick={() => void saveRoute()}
              disabled={
                isSaving || isRestoring || !routeLoaded || routeQuery.isFetching
              }
            >
              {isSaving ? <Loader2 className='animate-spin' /> : <Save />}
              {t('Save route')}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={discardConfirmOpen}
        onOpenChange={setDiscardConfirmOpen}
        title={t('Discard unsaved route changes?')}
        desc={t('Your unsaved priority and channel changes will be lost.')}
        destructive
        confirmText={t('Discard changes')}
        handleConfirm={() => {
          setDiscardConfirmOpen(false)
          props.onOpenChange(false)
        }}
      />
      <ConfirmDialog
        open={restoreConfirmOpen}
        onOpenChange={setRestoreConfirmOpen}
        title={t('Restore inherited route?')}
        desc={t(
          'This removes the explicit route and resumes using the current Ability priorities and weights.'
        )}
        destructive
        confirmText={t('Restore inherited route')}
        isLoading={isRestoring}
        handleConfirm={() => void restoreRoute()}
      />
    </>
  )
}
