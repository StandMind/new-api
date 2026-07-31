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
import { AlertTriangle, Plus, Trash2 } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { GroupModelRouteCandidate } from '../types'
import type { EditableGroupModelRouteTier } from './group-model-route-utils'

type GroupModelRouteTierEditorProps = {
  tiers: EditableGroupModelRouteTier[]
  candidates: GroupModelRouteCandidate[]
  savedChannelNames: Map<number, string>
  onChange: (tiers: EditableGroupModelRouteTier[]) => void
}

export function GroupModelRouteTierEditor(
  props: GroupModelRouteTierEditorProps
) {
  const { t } = useTranslation()
  const candidateMap = useMemo(
    () =>
      new Map(
        props.candidates.map((candidate) => [candidate.channel_id, candidate])
      ),
    [props.candidates]
  )
  const usedChannelIDs = useMemo(
    () =>
      new Set(
        props.tiers.flatMap((tier) =>
          tier.channels.map((channel) => channel.channel_id)
        )
      ),
    [props.tiers]
  )

  const updateTier = (
    tierIndex: number,
    updater: (tier: EditableGroupModelRouteTier) => EditableGroupModelRouteTier
  ) => {
    props.onChange(
      props.tiers.map((tier, index) =>
        index === tierIndex ? updater(tier) : tier
      )
    )
  }

  const addTier = () => {
    let priority = 100
    if (props.tiers.length > 0) {
      priority = Math.min(...props.tiers.map((tier) => tier.priority)) - 10
    }
    props.onChange([
      ...props.tiers,
      { editorId: crypto.randomUUID(), priority, channels: [] },
    ])
  }

  const addChannel = (tierIndex: number) => {
    const candidate = props.candidates.find(
      (item) => !usedChannelIDs.has(item.channel_id)
    )
    if (!candidate) return
    updateTier(tierIndex, (tier) => ({
      ...tier,
      channels: [
        ...tier.channels,
        { channel_id: candidate.channel_id, weight: candidate.weight },
      ],
    }))
  }

  return (
    <div className='space-y-3'>
      {props.tiers.map((tier, tierIndex) => (
        <section
          key={tier.editorId}
          className='overflow-hidden rounded-lg border'
        >
          <div className='bg-muted/30 flex flex-wrap items-end gap-3 border-b px-3 py-2'>
            <label className='min-w-32 flex-1 space-y-1 text-xs font-medium'>
              <span>{t('Priority')}</span>
              <Input
                type='number'
                step={1}
                value={tier.priority}
                onChange={(event) =>
                  updateTier(tierIndex, (current) => ({
                    ...current,
                    priority: Number(event.target.value),
                  }))
                }
              />
            </label>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type='button'
                    variant='destructive'
                    size='icon'
                    aria-label={t('Remove priority tier')}
                    onClick={() =>
                      props.onChange(
                        props.tiers.filter((_, index) => index !== tierIndex)
                      )
                    }
                  />
                }
              >
                <Trash2 />
              </TooltipTrigger>
              <TooltipContent>{t('Remove priority tier')}</TooltipContent>
            </Tooltip>
          </div>

          <div className='divide-y'>
            {tier.channels.map((channel, channelIndex) => {
              const currentCandidate = candidateMap.get(channel.channel_id)
              const currentName =
                currentCandidate?.channel_name ||
                props.savedChannelNames.get(channel.channel_id) ||
                t('Deleted or unavailable channel')
              return (
                <div
                  key={channel.channel_id}
                  className='grid gap-2 px-3 py-2 sm:grid-cols-[minmax(0,1fr)_8rem_auto] sm:items-end'
                >
                  <label className='space-y-1 text-xs font-medium'>
                    <span className='flex items-center gap-1.5'>
                      {t('Channel')}
                      {!currentCandidate && (
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <span
                                className='text-destructive inline-flex'
                                tabIndex={0}
                                aria-label={t('Currently unavailable')}
                              />
                            }
                          >
                            <AlertTriangle className='size-3.5' />
                          </TooltipTrigger>
                          <TooltipContent>
                            {t(
                              'This channel is no longer eligible. Remove or replace it before saving.'
                            )}
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </span>
                    <NativeSelect
                      className='w-full'
                      value={channel.channel_id}
                      onChange={(event) => {
                        const channelID = Number(event.target.value)
                        updateTier(tierIndex, (current) => ({
                          ...current,
                          channels: current.channels.map((item, index) => {
                            if (index !== channelIndex) return item
                            const candidate = candidateMap.get(channelID)
                            return {
                              channel_id: channelID,
                              weight: candidate?.weight ?? item.weight,
                            }
                          }),
                        }))
                      }}
                    >
                      {!currentCandidate && (
                        <NativeSelectOption value={channel.channel_id}>
                          #{channel.channel_id} {currentName} (
                          {t('Unavailable')})
                        </NativeSelectOption>
                      )}
                      {props.candidates.map((candidate) => {
                        const usedElsewhere =
                          usedChannelIDs.has(candidate.channel_id) &&
                          candidate.channel_id !== channel.channel_id
                        return (
                          <NativeSelectOption
                            key={candidate.channel_id}
                            value={candidate.channel_id}
                            disabled={usedElsewhere}
                          >
                            #{candidate.channel_id}{' '}
                            {candidate.channel_name || t('Unnamed channel')}
                          </NativeSelectOption>
                        )
                      })}
                    </NativeSelect>
                  </label>
                  <label className='space-y-1 text-xs font-medium'>
                    <span>{t('Weight')}</span>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      value={channel.weight}
                      onChange={(event) =>
                        updateTier(tierIndex, (current) => ({
                          ...current,
                          channels: current.channels.map((item, index) =>
                            index === channelIndex
                              ? {
                                  ...item,
                                  weight: Number(event.target.value),
                                }
                              : item
                          ),
                        }))
                      }
                    />
                  </label>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          type='button'
                          variant='destructive'
                          size='icon'
                          aria-label={t('Remove channel')}
                          onClick={() =>
                            updateTier(tierIndex, (current) => ({
                              ...current,
                              channels: current.channels.filter(
                                (_, index) => index !== channelIndex
                              ),
                            }))
                          }
                        />
                      }
                    >
                      <Trash2 />
                    </TooltipTrigger>
                    <TooltipContent>{t('Remove channel')}</TooltipContent>
                  </Tooltip>
                </div>
              )
            })}
          </div>

          <div className='px-3 py-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => addChannel(tierIndex)}
              disabled={usedChannelIDs.size >= props.candidates.length}
            >
              <Plus />
              {t('Add channel')}
            </Button>
          </div>
        </section>
      ))}

      <Button type='button' variant='outline' onClick={addTier}>
        <Plus />
        {t('Add priority tier')}
      </Button>

      <section className='space-y-2 pt-2'>
        <div className='flex items-center justify-between'>
          <h4 className='text-sm font-semibold'>{t('Eligible channels')}</h4>
          <Badge variant='secondary'>{props.candidates.length}</Badge>
        </div>
        {props.candidates.length === 0 ? (
          <p className='text-muted-foreground text-sm'>
            {t('No enabled channel ability matches this group and model.')}
          </p>
        ) : (
          <div className='overflow-hidden rounded-lg border text-sm'>
            <div className='text-muted-foreground bg-muted/30 grid grid-cols-[minmax(0,1fr)_6rem_6rem] gap-2 border-b px-3 py-2 text-xs font-medium'>
              <span>{t('Channel')}</span>
              <span>{t('Priority')}</span>
              <span>{t('Weight')}</span>
            </div>
            {props.candidates.map((candidate) => (
              <div
                key={candidate.channel_id}
                className='grid grid-cols-[minmax(0,1fr)_6rem_6rem] gap-2 border-b px-3 py-2 last:border-b-0'
              >
                <span className='truncate'>
                  #{candidate.channel_id}{' '}
                  {candidate.channel_name || t('Unnamed channel')}
                </span>
                <span className='tabular-nums'>{candidate.priority}</span>
                <span className='tabular-nums'>{candidate.weight}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
