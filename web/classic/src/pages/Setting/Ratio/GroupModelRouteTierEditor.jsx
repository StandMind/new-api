/*
Copyright (C) 2025 QuantumNous

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
import React, { useMemo } from 'react';
import {
  Button,
  InputNumber,
  Select,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

export default function GroupModelRouteTierEditor({
  tiers,
  candidates,
  savedChannelNames,
  onChange,
}) {
  const { t } = useTranslation();
  const candidateMap = useMemo(
    () =>
      new Map(candidates.map((candidate) => [candidate.channel_id, candidate])),
    [candidates],
  );
  const usedChannelIds = useMemo(
    () =>
      new Set(
        tiers.flatMap((tier) =>
          tier.channels.map((channel) => channel.channel_id),
        ),
      ),
    [tiers],
  );

  const updateTier = (tierIndex, updater) => {
    onChange(
      tiers.map((tier, index) => (index === tierIndex ? updater(tier) : tier)),
    );
  };

  const addTier = () => {
    let priority = 100;
    if (tiers.length > 0) {
      priority = Math.min(...tiers.map((tier) => tier.priority)) - 10;
    }
    onChange([
      ...tiers,
      {
        editorId: crypto.randomUUID(),
        priority,
        channels: [],
      },
    ]);
  };

  const addChannel = (tierIndex) => {
    const candidate = candidates.find(
      (item) => !usedChannelIds.has(item.channel_id),
    );
    if (!candidate) return;
    updateTier(tierIndex, (tier) => ({
      ...tier,
      channels: [
        ...tier.channels,
        { channel_id: candidate.channel_id, weight: candidate.weight },
      ],
    }));
  };

  return (
    <div className='space-y-3'>
      {tiers.map((tier, tierIndex) => (
        <section
          key={tier.editorId}
          className='overflow-hidden rounded-lg border border-solid border-gray-200'
        >
          <div className='flex flex-wrap items-end gap-3 border-b border-solid border-gray-200 bg-gray-50 px-3 py-2'>
            <label className='min-w-36 flex-1 space-y-1'>
              <Text size='small' strong>
                {t('优先级')}
              </Text>
              <InputNumber
                value={tier.priority}
                precision={0}
                onChange={(value) =>
                  updateTier(tierIndex, (current) => ({
                    ...current,
                    priority: Number(value),
                  }))
                }
                style={{ width: '100%' }}
              />
            </label>
            <Tooltip content={t('删除优先级层级')}>
              <Button
                icon={<IconDelete />}
                type='danger'
                theme='borderless'
                aria-label={t('删除优先级层级')}
                onClick={() =>
                  onChange(tiers.filter((_, index) => index !== tierIndex))
                }
              />
            </Tooltip>
          </div>

          <div className='divide-y divide-gray-100'>
            {tier.channels.map((channel, channelIndex) => {
              const currentCandidate = candidateMap.get(channel.channel_id);
              const currentName =
                currentCandidate?.channel_name ||
                savedChannelNames.get(channel.channel_id) ||
                t('已删除或不可用的渠道');
              const optionList = candidates.map((candidate) => ({
                value: candidate.channel_id,
                label: `#${candidate.channel_id} ${
                  candidate.channel_name || t('未命名渠道')
                }`,
                disabled:
                  usedChannelIds.has(candidate.channel_id) &&
                  candidate.channel_id !== channel.channel_id,
              }));
              if (!currentCandidate) {
                optionList.unshift({
                  value: channel.channel_id,
                  label: `#${channel.channel_id} ${currentName} (${t('不可用')})`,
                  disabled: false,
                });
              }
              return (
                <div
                  key={channel.channel_id}
                  className='grid gap-2 px-3 py-2 sm:grid-cols-[minmax(0,1fr)_8rem_auto] sm:items-end'
                >
                  <label className='space-y-1'>
                    <span className='flex items-center gap-2'>
                      <Text size='small' strong>
                        {t('渠道')}
                      </Text>
                      {!currentCandidate && (
                        <Tooltip
                          content={t('该渠道当前不可用，请在保存前移除或替换')}
                        >
                          <Tag color='red' size='small'>
                            {t('不可用')}
                          </Tag>
                        </Tooltip>
                      )}
                    </span>
                    <Select
                      value={channel.channel_id}
                      optionList={optionList}
                      onChange={(channelId) =>
                        updateTier(tierIndex, (current) => ({
                          ...current,
                          channels: current.channels.map((item, index) => {
                            if (index !== channelIndex) return item;
                            const candidate = candidateMap.get(
                              Number(channelId),
                            );
                            return {
                              channel_id: Number(channelId),
                              weight: candidate?.weight ?? item.weight,
                            };
                          }),
                        }))
                      }
                      style={{ width: '100%' }}
                    />
                  </label>
                  <label className='space-y-1'>
                    <Text size='small' strong>
                      {t('权重')}
                    </Text>
                    <InputNumber
                      min={0}
                      precision={0}
                      value={channel.weight}
                      onChange={(value) =>
                        updateTier(tierIndex, (current) => ({
                          ...current,
                          channels: current.channels.map((item, index) =>
                            index === channelIndex
                              ? { ...item, weight: Number(value) }
                              : item,
                          ),
                        }))
                      }
                      style={{ width: '100%' }}
                    />
                  </label>
                  <Tooltip content={t('移除渠道')}>
                    <Button
                      icon={<IconDelete />}
                      type='danger'
                      theme='borderless'
                      aria-label={t('移除渠道')}
                      onClick={() =>
                        updateTier(tierIndex, (current) => ({
                          ...current,
                          channels: current.channels.filter(
                            (_, index) => index !== channelIndex,
                          ),
                        }))
                      }
                    />
                  </Tooltip>
                </div>
              );
            })}
          </div>

          <div className='px-3 py-2'>
            <Button
              icon={<IconPlus />}
              theme='outline'
              size='small'
              disabled={usedChannelIds.size >= candidates.length}
              onClick={() => addChannel(tierIndex)}
            >
              {t('添加渠道')}
            </Button>
          </div>
        </section>
      ))}

      <Button icon={<IconPlus />} theme='outline' onClick={addTier}>
        {t('添加优先级层级')}
      </Button>

      <section className='space-y-2 pt-2'>
        <div className='flex items-center justify-between'>
          <Title heading={6}>{t('候选渠道')}</Title>
          <Tag color='grey'>{candidates.length}</Tag>
        </div>
        {candidates.length === 0 ? (
          <Text type='tertiary'>
            {t('没有启用的 Ability 匹配该分组和模型')}
          </Text>
        ) : (
          <div className='overflow-hidden rounded-lg border border-solid border-gray-200'>
            <div className='grid grid-cols-[minmax(0,1fr)_6rem_6rem] gap-2 border-b border-solid border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-500'>
              <span>{t('渠道')}</span>
              <span>{t('优先级')}</span>
              <span>{t('权重')}</span>
            </div>
            {candidates.map((candidate) => (
              <div
                key={candidate.channel_id}
                className='grid grid-cols-[minmax(0,1fr)_6rem_6rem] gap-2 border-b border-solid border-gray-100 px-3 py-2 last:border-b-0'
              >
                <span className='truncate'>
                  #{candidate.channel_id}{' '}
                  {candidate.channel_name || t('未命名渠道')}
                </span>
                <span>{candidate.priority}</span>
                <span>{candidate.weight}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
