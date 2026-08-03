import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound, Loader2, Pencil, Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  createUserLevel,
  deleteUserLevel,
  getRouteGroups,
  getUserLevels,
  replaceUserLevelRouteGroups,
  updateUserLevel,
} from './api'
import type { RouteGroup, UserLevel, UserLevelInput } from './types'

const emptyLevel: UserLevelInput = {
  name: '',
  description: '',
  is_default: false,
  enabled: true,
  topup_ratio: 1,
  request_limit: 0,
  success_request_limit: 0,
}

function referenceCount(level: UserLevel) {
  return level.references.reduce((sum, item) => sum + item.count, 0)
}

export function UserLevelsSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const levelsQuery = useQuery({
    queryKey: ['user-levels'],
    queryFn: getUserLevels,
  })
  const groupsQuery = useQuery({
    queryKey: ['route-groups'],
    queryFn: getRouteGroups,
  })
  const [editing, setEditing] = useState<UserLevel | 'new' | null>(null)
  const [permissionsLevel, setPermissionsLevel] = useState<UserLevel | null>(
    null
  )
  const [deleteTarget, setDeleteTarget] = useState<UserLevel | null>(null)

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['user-levels'] }),
      queryClient.invalidateQueries({ queryKey: ['route-groups'] }),
      queryClient.invalidateQueries({ queryKey: ['user-groups'] }),
    ])
  }

  const deleteMutation = useMutation({
    mutationFn: (code: string) => deleteUserLevel(code),
    onSuccess: async () => {
      toast.success(t('User level deleted'))
      setDeleteTarget(null)
      await refresh()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h2 className='text-lg font-semibold'>{t('User Levels')}</h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t(
              'Account levels control top-up pricing, request limits, and route access.'
            )}
          </p>
        </div>
        <Button onClick={() => setEditing('new')}>
          <Plus className='size-4' />
          {t('Add user level')}
        </Button>
      </div>

      <div className='overflow-x-auto rounded-lg border'>
        <Table className='min-w-[920px]'>
          <TableHeader>
            <TableRow>
              <TableHead>{t('User level')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Top-up ratio')}</TableHead>
              <TableHead>{t('Request limits')}</TableHead>
              <TableHead>{t('Route access')}</TableHead>
              <TableHead>{t('References')}</TableHead>
              <TableHead className='w-36 text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {levelsQuery.isLoading ? (
              <TableRow>
                <TableCell colSpan={7} className='h-24 text-center'>
                  <Loader2 className='mx-auto size-5 animate-spin' />
                </TableCell>
              </TableRow>
            ) : (
              (levelsQuery.data ?? []).map((level) => (
                <TableRow key={level.code}>
                  <TableCell>
                    <div className='font-medium'>{level.name}</div>
                    <div className='text-muted-foreground font-mono text-xs'>
                      {level.code}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className='flex gap-1.5'>
                      <Badge variant={level.enabled ? 'default' : 'secondary'}>
                        {level.enabled ? t('Enabled') : t('Disabled')}
                      </Badge>
                      {level.is_default && (
                        <Badge variant='outline'>{t('Default')}</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{level.topup_ratio}x</TableCell>
                  <TableCell className='text-sm'>
                    <div>
                      {t('Total')}: {level.request_limit || t('Unlimited')}
                    </div>
                    <div className='text-muted-foreground'>
                      {t('Successful')}:{' '}
                      {level.success_request_limit || t('Unlimited')}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setPermissionsLevel(level)}
                    >
                      <KeyRound className='size-4' />
                      {t('{{count}} route groups', {
                        count: level.route_groups?.length ?? 0,
                      })}
                    </Button>
                  </TableCell>
                  <TableCell>{referenceCount(level)}</TableCell>
                  <TableCell>
                    <div className='flex justify-end gap-1'>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              variant='ghost'
                              size='icon-sm'
                              onClick={() => setEditing(level)}
                            />
                          }
                        >
                          <Pencil className='size-4' />
                        </TooltipTrigger>
                        <TooltipContent>{t('Edit')}</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Button
                              variant='ghost'
                              size='icon-sm'
                              disabled={level.is_default}
                              onClick={() => setDeleteTarget(level)}
                            />
                          }
                        >
                          <Trash2 className='size-4' />
                        </TooltipTrigger>
                        <TooltipContent>{t('Delete')}</TooltipContent>
                      </Tooltip>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <UserLevelSheet
        target={editing}
        onOpenChange={(open) => !open && setEditing(null)}
        onSaved={refresh}
      />
      <RouteAccessSheet
        level={permissionsLevel}
        groups={groupsQuery.data ?? []}
        onOpenChange={(open) => !open && setPermissionsLevel(null)}
        onSaved={refresh}
      />
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('Delete user level?')}
        desc={t(
          'Deletion is blocked while users, subscriptions, or other business records reference this level.'
        )}
        confirmText={t('Delete')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() =>
          deleteTarget && deleteMutation.mutate(deleteTarget.code)
        }
      />
    </div>
  )
}

function UserLevelSheet(props: {
  target: UserLevel | 'new' | null
  onOpenChange: (open: boolean) => void
  onSaved: () => Promise<void>
}) {
  const { t } = useTranslation()
  const existing = props.target && props.target !== 'new' ? props.target : null
  const [code, setCode] = useState('')
  const [draft, setDraft] = useState<UserLevelInput>(emptyLevel)

  useEffect(() => {
    setCode(existing?.code ?? '')
    setDraft(
      existing
        ? {
            name: existing.name,
            description: existing.description,
            is_default: existing.is_default,
            enabled: existing.enabled,
            topup_ratio: existing.topup_ratio,
            request_limit: existing.request_limit,
            success_request_limit: existing.success_request_limit,
          }
        : emptyLevel
    )
  }, [existing, props.target])

  const saveMutation = useMutation({
    mutationFn: () =>
      existing
        ? updateUserLevel(existing.code, draft)
        : createUserLevel(code.trim(), draft),
    onSuccess: async () => {
      toast.success(t('User level saved'))
      props.onOpenChange(false)
      await props.onSaved()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const numberField = (
    key: 'topup_ratio' | 'request_limit' | 'success_request_limit',
    value: string
  ) =>
    setDraft((current) => ({
      ...current,
      [key]: Number(value),
    }))

  return (
    <Sheet open={Boolean(props.target)} onOpenChange={props.onOpenChange}>
      <SheetContent className='sm:max-w-lg'>
        <SheetHeader>
          <SheetTitle>
            {existing ? t('Edit user level') : t('Add user level')}
          </SheetTitle>
          <SheetDescription>
            {t('The code is permanent after the level is created.')}
          </SheetDescription>
        </SheetHeader>
        <div className='grid gap-4 overflow-y-auto px-4 py-2'>
          <div className='grid gap-1.5'>
            <Label htmlFor='level-code'>{t('Code')}</Label>
            <Input
              id='level-code'
              value={code}
              disabled={Boolean(existing)}
              onChange={(event) => setCode(event.target.value)}
            />
          </div>
          <div className='grid gap-1.5'>
            <Label htmlFor='level-name'>{t('Display name')}</Label>
            <Input
              id='level-name'
              value={draft.name}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
            />
          </div>
          <div className='grid gap-1.5'>
            <Label htmlFor='level-description'>{t('Description')}</Label>
            <Textarea
              id='level-description'
              value={draft.description}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  description: event.target.value,
                }))
              }
            />
          </div>
          <div className='grid grid-cols-1 gap-4 sm:grid-cols-3'>
            <div className='grid gap-1.5'>
              <Label htmlFor='level-topup'>{t('Top-up ratio')}</Label>
              <Input
                id='level-topup'
                type='number'
                min='0.000001'
                step='0.01'
                value={draft.topup_ratio}
                onChange={(event) =>
                  numberField('topup_ratio', event.target.value)
                }
              />
            </div>
            <div className='grid gap-1.5'>
              <Label htmlFor='level-total-limit'>{t('Total limit')}</Label>
              <Input
                id='level-total-limit'
                type='number'
                min='0'
                value={draft.request_limit}
                onChange={(event) =>
                  numberField('request_limit', event.target.value)
                }
              />
            </div>
            <div className='grid gap-1.5'>
              <Label htmlFor='level-success-limit'>{t('Success limit')}</Label>
              <Input
                id='level-success-limit'
                type='number'
                min='0'
                value={draft.success_request_limit}
                onChange={(event) =>
                  numberField('success_request_limit', event.target.value)
                }
              />
            </div>
          </div>
          <div className='flex items-center justify-between border-t py-3'>
            <div>
              <Label>{t('Enabled')}</Label>
              <p className='text-muted-foreground text-xs'>
                {t('Referenced levels cannot be disabled.')}
              </p>
            </div>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(enabled) =>
                setDraft((current) => ({ ...current, enabled }))
              }
            />
          </div>
          <div className='flex items-center justify-between border-t py-3'>
            <div>
              <Label>{t('Default level')}</Label>
              <p className='text-muted-foreground text-xs'>
                {t('New users are assigned to this level.')}
              </p>
            </div>
            <Switch
              checked={draft.is_default}
              onCheckedChange={(is_default) =>
                setDraft((current) => ({ ...current, is_default }))
              }
            />
          </div>
        </div>
        <SheetFooter>
          <Button
            disabled={
              saveMutation.isPending || !code.trim() || !draft.name.trim()
            }
            onClick={() => saveMutation.mutate()}
          >
            {saveMutation.isPending ? (
              <Loader2 className='size-4 animate-spin' />
            ) : (
              <Save className='size-4' />
            )}
            {t('Save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

function RouteAccessSheet(props: {
  level: UserLevel | null
  groups: RouteGroup[]
  onOpenChange: (open: boolean) => void
  onSaved: () => Promise<void>
}) {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<Record<string, string>>({})

  useEffect(() => {
    const next: Record<string, string> = {}
    for (const grant of props.level?.route_groups ?? []) {
      next[grant.route_group_code] =
        grant.price_ratio == null ? '' : String(grant.price_ratio)
    }
    setSelected(next)
  }, [props.level])

  const orderedGroups = useMemo(
    () => [...props.groups].sort((a, b) => a.name.localeCompare(b.name)),
    [props.groups]
  )
  const saveMutation = useMutation({
    mutationFn: () =>
      replaceUserLevelRouteGroups(
        props.level?.code ?? '',
        Object.entries(selected).map(([code, ratio]) => ({
          code,
          price_ratio: ratio.trim() === '' ? null : Number(ratio),
        }))
      ),
    onSuccess: async () => {
      toast.success(t('Route access saved'))
      props.onOpenChange(false)
      await props.onSaved()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Sheet open={Boolean(props.level)} onOpenChange={props.onOpenChange}>
      <SheetContent className='sm:max-w-xl'>
        <SheetHeader>
          <SheetTitle>{t('Route access')}</SheetTitle>
          <SheetDescription>
            {props.level?.name}
            {' · '}
            {t('New route groups are not selected automatically.')}
          </SheetDescription>
        </SheetHeader>
        <div className='flex-1 space-y-2 overflow-y-auto px-4 py-2'>
          {orderedGroups
            .filter((group) => group.code !== 'default')
            .map((group) => {
              const checked = Object.hasOwn(selected, group.code)
              return (
                <div
                  key={group.code}
                  className='grid grid-cols-[auto_minmax(0,1fr)_9rem] items-center gap-3 rounded-md border p-3'
                >
                  <Checkbox
                    checked={checked}
                    onCheckedChange={(value) =>
                      setSelected((current) => {
                        const next = { ...current }
                        if (value) next[group.code] = ''
                        else delete next[group.code]
                        return next
                      })
                    }
                  />
                  <div className='min-w-0'>
                    <div className='truncate text-sm font-medium'>
                      {group.name}
                    </div>
                    <div className='text-muted-foreground truncate font-mono text-xs'>
                      {group.code}
                      {!group.enabled ? ` · ${t('Disabled')}` : ''}
                    </div>
                  </div>
                  <Input
                    type='number'
                    min='0'
                    step='0.01'
                    disabled={!checked}
                    value={selected[group.code] ?? ''}
                    placeholder={`${group.base_ratio}x`}
                    aria-label={t('Absolute price ratio')}
                    onChange={(event) =>
                      setSelected((current) => ({
                        ...current,
                        [group.code]: event.target.value,
                      }))
                    }
                  />
                </div>
              )
            })}
        </div>
        <SheetFooter>
          <Button
            disabled={saveMutation.isPending || !props.level}
            onClick={() => saveMutation.mutate()}
          >
            {saveMutation.isPending ? (
              <Loader2 className='size-4 animate-spin' />
            ) : (
              <Save className='size-4' />
            )}
            {t('Save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
