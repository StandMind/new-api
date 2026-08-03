import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Pencil, Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

import { GroupModelRouteEditor } from '../models/group-model-route-editor'
import {
  createRouteGroup,
  deleteRouteGroup,
  getRouteGroups,
  updateRouteGroup,
} from './api'
import type { RouteGroup, RouteGroupInput } from './types'

const emptyGroup: RouteGroupInput = {
  name: '',
  description: '',
  base_ratio: 1,
  enabled: true,
}

function referenceCount(group: RouteGroup) {
  return group.references.reduce((sum, item) => sum + item.count, 0)
}

export function RouteGroupsSection(props: { groupRatio: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const groupsQuery = useQuery({
    queryKey: ['route-groups'],
    queryFn: getRouteGroups,
  })
  const [editing, setEditing] = useState<RouteGroup | 'new' | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<RouteGroup | null>(null)

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['route-groups'] }),
      queryClient.invalidateQueries({ queryKey: ['user-levels'] }),
      queryClient.invalidateQueries({ queryKey: ['my-route-groups'] }),
      queryClient.invalidateQueries({ queryKey: ['user-groups'] }),
      queryClient.invalidateQueries({ queryKey: ['pricing'] }),
    ])
  }

  const deleteMutation = useMutation({
    mutationFn: (code: string) => deleteRouteGroup(code),
    onSuccess: async () => {
      toast.success(t('Route group deleted'))
      setDeleteTarget(null)
      await refresh()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <div className='space-y-8'>
      <section className='space-y-5'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <h2 className='text-lg font-semibold'>{t('Route Groups')}</h2>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t(
                'Route groups control channels, model availability, pricing, and request routing.'
              )}
            </p>
          </div>
          <Button onClick={() => setEditing('new')}>
            <Plus className='size-4' />
            {t('Add route group')}
          </Button>
        </div>

        <div className='overflow-x-auto rounded-lg border'>
          <Table className='min-w-[820px]'>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Route group')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Base price ratio')}</TableHead>
                <TableHead>{t('References')}</TableHead>
                <TableHead className='w-28 text-right'>
                  {t('Actions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groupsQuery.isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className='h-24 text-center'>
                    <Loader2 className='mx-auto size-5 animate-spin' />
                  </TableCell>
                </TableRow>
              ) : (
                (groupsQuery.data ?? []).map((group) => (
                  <TableRow key={group.code}>
                    <TableCell>
                      <div className='font-medium'>{group.name}</div>
                      <div className='text-muted-foreground font-mono text-xs'>
                        {group.code}
                      </div>
                      {group.description && (
                        <div className='text-muted-foreground mt-1 max-w-md text-xs'>
                          {group.description}
                        </div>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={group.enabled ? 'default' : 'secondary'}>
                        {group.enabled ? t('Enabled') : t('Disabled')}
                      </Badge>
                    </TableCell>
                    <TableCell>{group.base_ratio}x</TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <span className='cursor-help tabular-nums underline decoration-dotted underline-offset-4' />
                          }
                        >
                          {referenceCount(group)}
                        </TooltipTrigger>
                        <TooltipContent className='max-w-xs'>
                          {group.references.length === 0
                            ? t('No references')
                            : group.references
                                .map(
                                  (reference) =>
                                    `${reference.kind}: ${reference.count}`
                                )
                                .join('\n')}
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <div className='flex justify-end gap-1'>
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <Button
                                variant='ghost'
                                size='icon-sm'
                                disabled={group.code === 'default'}
                                onClick={() => setEditing(group)}
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
                                disabled={
                                  group.code === 'default' ||
                                  referenceCount(group) > 0
                                }
                                onClick={() => setDeleteTarget(group)}
                              />
                            }
                          >
                            <Trash2 className='size-4' />
                          </TooltipTrigger>
                          <TooltipContent>
                            {referenceCount(group) > 0
                              ? t('Referenced route groups cannot be deleted.')
                              : t('Delete')}
                          </TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </section>

      <section className='space-y-4 border-t pt-7'>
        <div>
          <h2 className='text-lg font-semibold'>{t('Model channel routes')}</h2>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t(
              'Configure explicit channel priority and weight within each route group.'
            )}
          </p>
        </div>
        <GroupModelRouteEditor groupRatio={props.groupRatio} />
      </section>

      <RouteGroupSheet
        target={editing}
        onOpenChange={(open) => !open && setEditing(null)}
        onSaved={refresh}
      />
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('Delete route group?')}
        desc={t(
          'Deletion is blocked while channels, abilities, tokens, routes, level grants, or OpenLux bindings reference this route group.'
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

function RouteGroupSheet(props: {
  target: RouteGroup | 'new' | null
  onOpenChange: (open: boolean) => void
  onSaved: () => Promise<void>
}) {
  const { t } = useTranslation()
  const existing = props.target && props.target !== 'new' ? props.target : null
  const [code, setCode] = useState('')
  const [draft, setDraft] = useState<RouteGroupInput>(emptyGroup)

  useEffect(() => {
    setCode(existing?.code ?? '')
    setDraft(
      existing
        ? {
            name: existing.name,
            description: existing.description,
            base_ratio: existing.base_ratio,
            enabled: existing.enabled,
          }
        : emptyGroup
    )
  }, [existing, props.target])

  const saveMutation = useMutation({
    mutationFn: () =>
      existing
        ? updateRouteGroup(existing.code, draft)
        : createRouteGroup(code.trim(), draft),
    onSuccess: async () => {
      toast.success(t('Route group saved'))
      props.onOpenChange(false)
      await props.onSaved()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Sheet open={Boolean(props.target)} onOpenChange={props.onOpenChange}>
      <SheetContent className='sm:max-w-lg'>
        <SheetHeader>
          <SheetTitle>
            {existing ? t('Edit route group') : t('Add route group')}
          </SheetTitle>
          <SheetDescription>
            {t('The code is permanent after the route group is created.')}
          </SheetDescription>
        </SheetHeader>
        <div className='grid gap-4 overflow-y-auto px-4 py-2'>
          <div className='grid gap-1.5'>
            <Label htmlFor='route-group-code'>{t('Code')}</Label>
            <Input
              id='route-group-code'
              value={code}
              disabled={Boolean(existing)}
              onChange={(event) => setCode(event.target.value)}
            />
          </div>
          <div className='grid gap-1.5'>
            <Label htmlFor='route-group-name'>{t('Display name')}</Label>
            <Input
              id='route-group-name'
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
            <Label htmlFor='route-group-description'>{t('Description')}</Label>
            <Textarea
              id='route-group-description'
              value={draft.description}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  description: event.target.value,
                }))
              }
            />
          </div>
          <div className='grid gap-1.5'>
            <Label htmlFor='route-group-ratio'>{t('Base price ratio')}</Label>
            <Input
              id='route-group-ratio'
              type='number'
              min='0'
              step='0.01'
              value={draft.base_ratio}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  base_ratio: Number(event.target.value),
                }))
              }
            />
          </div>
          <div className='flex items-center justify-between border-t py-3'>
            <div>
              <Label>{t('Enabled')}</Label>
              <p className='text-muted-foreground text-xs'>
                {t('Disabled route groups are excluded from request routing.')}
              </p>
            </div>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(enabled) =>
                setDraft((current) => ({ ...current, enabled }))
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
