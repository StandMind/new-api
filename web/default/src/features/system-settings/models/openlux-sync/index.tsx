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
import { useQueryClient } from '@tanstack/react-query'
import axios from 'axios'
import { Check, Loader2, RefreshCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  applyOpenLuxSync,
  getOpenLuxBindings,
  previewOpenLuxSync,
  saveOpenLuxBindings,
} from './api'
import { OpenLuxBindingEditor } from './binding-editor'
import { OpenLuxPreviewPanel } from './preview-panel'
import { resolveOpenLuxSelection } from './selection'
import type {
  OpenLuxBindings,
  OpenLuxPreview,
  SaveOpenLuxBindingsRequest,
} from './types'

type OpenLuxView = 'preview' | 'bindings'

function errorMessage(error: unknown, fallback: string) {
  if (axios.isAxiosError(error)) {
    const message = error.response?.data?.message
    if (typeof message === 'string' && message) return message
  }
  if (error instanceof Error && error.message) return error.message
  return fallback
}

function errorStatus(error: unknown) {
  return axios.isAxiosError(error) ? error.response?.status : undefined
}

export function OpenLuxPriceSync() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [view, setView] = useState<OpenLuxView>('preview')
  const [bindings, setBindings] = useState<OpenLuxBindings | null>(null)
  const [preview, setPreview] = useState<OpenLuxPreview | null>(null)
  const [selectedRoots, setSelectedRoots] = useState<Set<string>>(new Set())
  const [loadingBindings, setLoadingBindings] = useState(false)
  const [savingBindings, setSavingBindings] = useState(false)
  const [checking, setChecking] = useState(false)
  const [applying, setApplying] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  const selected = useMemo(
    () =>
      preview
        ? resolveOpenLuxSelection(preview, selectedRoots)
        : new Set<string>(),
    [preview, selectedRoots]
  )
  const selectedChanges = useMemo(
    () => preview?.changes.filter((change) => selected.has(change.id)) ?? [],
    [preview, selected]
  )
  const priceCount = selectedChanges.filter(
    (change) =>
      change.kind === 'model_billing_update' ||
      change.kind === 'group_model_price_update'
  ).length
  const associationRemovalCount = selectedChanges.filter(
    (change) => change.kind === 'group_model_remove'
  ).length
  const sourceRemovalCount = selectedChanges.filter(
    (change) => change.kind === 'source_group_remove'
  ).length
  const fullGroupRemovalCount = selectedChanges.filter(
    (change) =>
      change.kind === 'source_group_remove' &&
      change.remove_mode === 'delete_local_group'
  ).length
  const hasDestructiveChanges = selectedChanges.some(
    (change) => change.destructive
  )

  const refreshBindings = async () => {
    setLoadingBindings(true)
    try {
      setBindings(await getOpenLuxBindings())
    } catch (error) {
      toast.error(errorMessage(error, t('Failed to load OpenLux bindings')))
    } finally {
      setLoadingBindings(false)
    }
  }

  const checkPrices = async () => {
    setChecking(true)
    setSelectedRoots(new Set())
    try {
      const next = await previewOpenLuxSync()
      setPreview(next)
      if (next.changes.length === 0) {
        toast.success(t('No OpenLux price changes'))
      }
    } catch (error) {
      setPreview(null)
      toast.error(errorMessage(error, t('Failed to check OpenLux prices')))
    } finally {
      setChecking(false)
    }
  }

  const saveBindings = async (request: SaveOpenLuxBindingsRequest) => {
    setSavingBindings(true)
    try {
      const next = await saveOpenLuxBindings(request)
      setBindings(next)
      setPreview(null)
      setSelectedRoots(new Set())
      toast.success(t('OpenLux bindings saved'))
    } catch (error) {
      if (errorStatus(error) === 409) {
        setSelectedRoots(new Set())
        await refreshBindings()
      }
      toast.error(errorMessage(error, t('Failed to save OpenLux bindings')))
    } finally {
      setSavingBindings(false)
    }
  }

  const applyChanges = async () => {
    if (!preview || selected.size === 0) return
    setApplying(true)
    try {
      const result = await applyOpenLuxSync({
        source_hash: preview.source_hash,
        local_fingerprint: preview.local_fingerprint,
        binding_revision: preview.binding_revision,
        change_ids: [...selected].sort(),
      })
      setConfirmOpen(false)
      setSelectedRoots(new Set())
      toast.success(
        t('Applied {{count}} OpenLux changes', {
          count: result.applied_count,
        })
      )
      try {
        await queryClient.invalidateQueries({ queryKey: ['system-options'] })
        const [nextBindings, nextPreview] = await Promise.all([
          getOpenLuxBindings(),
          previewOpenLuxSync(),
        ])
        setBindings(nextBindings)
        setPreview(nextPreview)
      } catch (refreshError) {
        setPreview(null)
        toast.warning(
          errorMessage(
            refreshError,
            t('Changes applied, but the refreshed preview could not be loaded')
          )
        )
      }
    } catch (error) {
      if (errorStatus(error) === 409) {
        setConfirmOpen(false)
        setSelectedRoots(new Set())
        setPreview(null)
        toast.error(t('The preview is stale. Check OpenLux prices again.'))
      } else {
        toast.error(errorMessage(error, t('Failed to apply OpenLux changes')))
      }
    } finally {
      setApplying(false)
    }
  }

  const changeView = (value: string) => {
    const next = value as OpenLuxView
    setView(next)
    if (next === 'bindings' && bindings === null && !loadingBindings) {
      void refreshBindings()
    }
  }

  const busy = loadingBindings || savingBindings || checking || applying

  return (
    <>
      <Tabs value={view} onValueChange={changeView} className='gap-4'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <TabsList>
            <TabsTrigger value='preview'>{t('Sync preview')}</TabsTrigger>
            <TabsTrigger value='bindings'>{t('Source bindings')}</TabsTrigger>
          </TabsList>

          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant='outline'>api.openlux.ai</Badge>
            {view === 'preview' ? (
              <>
                <Button
                  type='button'
                  variant='outline'
                  disabled={busy}
                  onClick={() => void checkPrices()}
                >
                  {checking ? (
                    <Loader2 className='animate-spin' />
                  ) : (
                    <RefreshCcw />
                  )}
                  {checking ? t('Checking...') : t('Check OpenLux')}
                </Button>
                <Button
                  type='button'
                  disabled={busy || selected.size === 0}
                  onClick={() => setConfirmOpen(true)}
                >
                  {applying ? <Loader2 className='animate-spin' /> : <Check />}
                  {t('Apply selected ({{count}})', { count: selected.size })}
                </Button>
              </>
            ) : (
              <Button
                type='button'
                variant='outline'
                disabled={busy}
                onClick={() => void refreshBindings()}
              >
                <RefreshCcw className={loadingBindings ? 'animate-spin' : ''} />
                {t('Refresh')}
              </Button>
            )}
          </div>
        </div>

        <TabsContent value='preview'>
          {preview ? (
            <OpenLuxPreviewPanel
              preview={preview}
              selectedRoots={selectedRoots}
              onSelectedRootsChange={setSelectedRoots}
              disabled={busy}
            />
          ) : (
            <div className='text-muted-foreground flex min-h-48 flex-col items-center justify-center gap-3 border-y text-center'>
              {checking ? (
                <Loader2 className='size-6 animate-spin' />
              ) : (
                <RefreshCcw className='size-6' />
              )}
              <span>{t('No OpenLux preview')}</span>
            </div>
          )}
        </TabsContent>

        <TabsContent value='bindings'>
          {bindings ? (
            <OpenLuxBindingEditor
              data={bindings}
              saving={savingBindings}
              onSave={saveBindings}
            />
          ) : (
            <div className='text-muted-foreground flex min-h-48 items-center justify-center border-y'>
              {loadingBindings ? (
                <Loader2 className='size-6 animate-spin' />
              ) : (
                t('OpenLux bindings are not loaded')
              )}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('Apply OpenLux price changes?')}
        desc={
          <div className='space-y-2 text-left'>
            <div>
              {t('{{count}} price and billing changes', { count: priceCount })}
            </div>
            <div>
              {t('{{count}} OpenLux model removals', {
                count: associationRemovalCount,
              })}
            </div>
            <div>
              {t('{{count}} OpenLux source group removals', {
                count: sourceRemovalCount,
              })}
            </div>
            {fullGroupRemovalCount > 0 && (
              <div className='text-destructive font-medium'>
                {t('{{count}} local groups will be deleted', {
                  count: fullGroupRemovalCount,
                })}
              </div>
            )}
          </div>
        }
        destructive={hasDestructiveChanges}
        isLoading={applying}
        disabled={selected.size === 0}
        handleConfirm={() => void applyChanges()}
        confirmText={t('Apply changes')}
      />
    </>
  )
}
