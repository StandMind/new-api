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
import { Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type {
  OpenLuxBinding,
  OpenLuxBindings,
  OpenLuxChannelSummary,
  SaveOpenLuxBindingsRequest,
} from './types'

type BindingDraft = {
  id: string
  sourceGroup: string
  localGroup: string
  channelIds: number[]
}

type BindingEditorProps = {
  data: OpenLuxBindings
  saving: boolean
  onSave: (request: SaveOpenLuxBindingsRequest) => Promise<void>
}

let nextDraftId = 0

function bindingToDraft(binding: OpenLuxBinding): BindingDraft {
  return {
    id: `saved-${binding.source_group}`,
    sourceGroup: binding.source_group,
    localGroup: binding.local_group,
    channelIds: (binding.channels ?? []).map((channel) => channel.id),
  }
}

function normalizedBindingKey(drafts: BindingDraft[]) {
  return JSON.stringify(
    drafts
      .map((draft) => ({
        source_group: draft.sourceGroup,
        channel_ids: [...draft.channelIds].sort((a, b) => a - b),
      }))
      .sort((a, b) => a.source_group.localeCompare(b.source_group))
  )
}

export function OpenLuxBindingEditor({
  data,
  saving,
  onSave,
}: BindingEditorProps) {
  const { t } = useTranslation()
  const savedDrafts = useMemo(
    () => (data.bindings ?? []).map(bindingToDraft),
    [data.bindings]
  )
  const [drafts, setDrafts] = useState<BindingDraft[]>(savedDrafts)

  useEffect(() => setDrafts(savedDrafts), [savedDrafts])

  const channelById = useMemo(() => {
    const result = new Map<number, OpenLuxChannelSummary>()
    for (const channel of data.openlux_channels ?? []) {
      result.set(channel.id, channel)
    }
    for (const binding of data.bindings ?? []) {
      for (const channel of binding.channels ?? []) {
        result.set(channel.id, channel)
      }
    }
    return result
  }, [data.bindings, data.openlux_channels])

  const localGroups = useMemo(
    () =>
      [
        ...new Set(
          [...channelById.values()]
            .map((channel) => channel.local_group)
            .filter(Boolean)
        ),
      ].sort((a, b) => a.localeCompare(b)),
    [channelById]
  )
  const sourceGroups = data.source_groups ?? []
  const usedSources = new Set(drafts.map((draft) => draft.sourceGroup))
  const usedChannels = new Set(drafts.flatMap((draft) => draft.channelIds))
  const savedKey = normalizedBindingKey(savedDrafts)
  const draftKey = normalizedBindingKey(drafts)

  const valid =
    drafts.every(
      (draft) =>
        draft.sourceGroup !== '' &&
        draft.localGroup !== '' &&
        draft.channelIds.length > 0 &&
        sourceGroups.some((source) => source.name === draft.sourceGroup)
    ) &&
    usedSources.size === drafts.length &&
    usedChannels.size ===
      drafts.reduce((sum, item) => sum + item.channelIds.length, 0)

  const updateDraft = (id: string, update: Partial<BindingDraft>) => {
    setDrafts((current) =>
      current.map((draft) =>
        draft.id === id ? { ...draft, ...update } : draft
      )
    )
  }

  const selectLocalGroup = (draft: BindingDraft, localGroup: string) => {
    const claimedByOthers = new Set(
      drafts
        .filter((item) => item.id !== draft.id)
        .flatMap((item) => item.channelIds)
    )
    const channelIds = [...channelById.values()]
      .filter(
        (channel) =>
          channel.local_group === localGroup && !claimedByOthers.has(channel.id)
      )
      .map((channel) => channel.id)
      .sort((a, b) => a - b)
    updateDraft(draft.id, { localGroup, channelIds })
  }

  const toggleChannel = (
    draft: BindingDraft,
    channelId: number,
    checked: boolean
  ) => {
    const next = checked
      ? [...new Set([...draft.channelIds, channelId])]
      : draft.channelIds.filter((id) => id !== channelId)
    updateDraft(draft.id, { channelIds: next.sort((a, b) => a - b) })
  }

  const addDraft = () => {
    const sourceGroup =
      sourceGroups.find((source) => !usedSources.has(source.name))?.name ?? ''
    setDrafts((current) => [
      ...current,
      {
        id: `new-${++nextDraftId}`,
        sourceGroup,
        localGroup: '',
        channelIds: [],
      },
    ])
  }

  const adoptCandidate = (candidate: OpenLuxBinding) => {
    if (usedSources.has(candidate.source_group)) return
    setDrafts((current) => [
      ...current,
      {
        id: `candidate-${candidate.source_group}-${++nextDraftId}`,
        sourceGroup: candidate.source_group,
        localGroup: candidate.local_group,
        channelIds: (candidate.channels ?? []).map((channel) => channel.id),
      },
    ])
  }

  const submit = async () => {
    await onSave({
      expected_revision: data.binding_revision,
      bindings: drafts
        .map((draft) => ({
          source_group: draft.sourceGroup,
          channel_ids: [...draft.channelIds].sort((a, b) => a - b),
        }))
        .sort((a, b) => a.source_group.localeCompare(b.source_group)),
    })
  }

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='flex items-center gap-2'>
          <span className='text-sm font-medium'>{t('Source bindings')}</span>
          <Badge variant='outline'>r{data.binding_revision}</Badge>
        </div>
        <div className='flex gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={addDraft}
            disabled={saving || usedSources.size >= sourceGroups.length}
          >
            <Plus />
            {t('Add binding')}
          </Button>
          <Button
            type='button'
            onClick={() => void submit()}
            disabled={saving || !valid || savedKey === draftKey}
          >
            <Save />
            {saving ? t('Saving...') : t('Save bindings')}
          </Button>
        </div>
      </div>

      <div className='divide-y rounded-md border'>
        {drafts.length === 0 && (
          <div className='text-muted-foreground px-3 py-8 text-center'>
            {t('No OpenLux bindings')}
          </div>
        )}
        {drafts.map((draft) => {
          const saved = (data.bindings ?? []).find(
            (binding) => binding.source_group === draft.sourceGroup
          )
          const channels = [...channelById.values()]
            .filter((channel) => channel.local_group === draft.localGroup)
            .sort((a, b) => a.id - b.id)
          return (
            <div key={draft.id} className='space-y-3 p-3'>
              <div className='grid gap-3 lg:grid-cols-[minmax(12rem,1fr)_minmax(12rem,1fr)_auto]'>
                <Select
                  value={draft.sourceGroup || null}
                  onValueChange={(value) =>
                    updateDraft(draft.id, { sourceGroup: value ?? '' })
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder={t('Source group')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {sourceGroups.map((source) => (
                        <SelectItem
                          key={source.name}
                          value={source.name}
                          disabled={
                            source.name !== draft.sourceGroup &&
                            usedSources.has(source.name)
                          }
                        >
                          {source.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>

                <Select
                  value={draft.localGroup || null}
                  onValueChange={(value) =>
                    selectLocalGroup(draft, value ?? '')
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder={t('Local group')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {localGroups.map((group) => (
                        <SelectItem key={group} value={group}>
                          {group}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>

                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon'
                        aria-label={t('Remove binding')}
                        disabled={saving}
                        onClick={() =>
                          setDrafts((current) =>
                            current.filter((item) => item.id !== draft.id)
                          )
                        }
                      />
                    }
                  >
                    <Trash2 />
                  </TooltipTrigger>
                  <TooltipContent>{t('Remove binding')}</TooltipContent>
                </Tooltip>
              </div>

              {channels.length > 0 && (
                <div className='flex flex-wrap gap-x-5 gap-y-2'>
                  {channels.map((channel) => {
                    const claimedElsewhere = drafts.some(
                      (item) =>
                        item.id !== draft.id &&
                        item.channelIds.includes(channel.id)
                    )
                    return (
                      <label
                        key={channel.id}
                        className='flex min-w-0 items-center gap-2 text-sm'
                      >
                        <Checkbox
                          checked={draft.channelIds.includes(channel.id)}
                          disabled={saving || claimedElsewhere}
                          onCheckedChange={(checked) =>
                            toggleChannel(draft, channel.id, checked === true)
                          }
                        />
                        <span className='break-all'>
                          {channel.name} #{channel.id}
                        </span>
                        <span className='text-muted-foreground tabular-nums'>
                          {t('{{count}} models', {
                            count: channel.model_count,
                          })}
                        </span>
                      </label>
                    )
                  })}
                </div>
              )}

              {saved && (
                <div className='flex flex-wrap items-center gap-2'>
                  <Badge variant={saved.healthy ? 'secondary' : 'destructive'}>
                    {saved.healthy ? t('Healthy') : t('Binding issue')}
                  </Badge>
                  {saved.mixed_channel_count > 0 && (
                    <Badge variant='outline'>
                      {t('{{count}} other channels in this group', {
                        count: saved.mixed_channel_count,
                      })}
                    </Badge>
                  )}
                  {(saved.issues ?? []).map((issue) => (
                    <span key={issue} className='text-destructive text-sm'>
                      {issue}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </div>

      {(data.candidates ?? []).length > 0 && (
        <section className='space-y-2 border-t pt-4'>
          <h3 className='text-sm font-medium'>{t('Binding candidates')}</h3>
          <div className='divide-y rounded-md border'>
            {(data.candidates ?? []).map((candidate) => (
              <div
                key={`${candidate.source_group}-${candidate.local_group}`}
                className='flex flex-wrap items-center justify-between gap-3 px-3 py-2'
              >
                <div className='min-w-0'>
                  <div className='font-medium break-all'>
                    {candidate.source_group}
                  </div>
                  <div className='text-muted-foreground text-sm break-all'>
                    {candidate.local_group} ·{' '}
                    {(candidate.channels ?? [])
                      .map((channel) => channel.name)
                      .join(', ')}
                  </div>
                </div>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={saving || usedSources.has(candidate.source_group)}
                  onClick={() => adoptCandidate(candidate)}
                >
                  <Plus />
                  {t('Use candidate')}
                </Button>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
