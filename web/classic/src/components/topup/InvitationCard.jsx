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

import React from 'react';
import {
  Avatar,
  Button,
  Card,
  Divider,
  Input,
  Pagination,
  Skeleton,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  BadgePercent,
  Copy,
  Gift,
  History,
  PauseCircle,
  Users,
  WalletCards,
  Zap,
} from 'lucide-react';
import { timestamp2string } from '../../helpers';

const { Text } = Typography;

const InvitationCard = ({
  t,
  userState,
  renderQuota,
  setOpenTransfer,
  affLink,
  handleAffLinkClick,
  invitationInfo,
  rewards,
  rewardPage,
  rewardPageSize,
  rewardTotal,
  onRewardPageChange,
  loading,
  complianceConfirmed = true,
}) => {
  if (loading) {
    return (
      <Card className='!rounded-xl shadow-sm border-0 h-full'>
        <Skeleton placeholder={<Skeleton.Paragraph rows={8} />} loading />
      </Card>
    );
  }

  const mode = invitationInfo?.mode || 'disabled';
  const pendingQuota =
    invitationInfo?.pending_reward_quota || userState?.user?.aff_quota || 0;
  const totalQuota =
    invitationInfo?.total_reward_quota ||
    userState?.user?.aff_history_quota ||
    0;
  const inviteCount =
    invitationInfo?.invite_count ?? userState?.user?.aff_count ?? 0;
  const rebateRate = (invitationInfo?.rebate_bps || 0) / 100;
  const rebateTopupCount = invitationInfo?.rebate_topup_count || 0;
  const rebateDetails = [
    {
      markerClassName: 'bg-blue-500',
      text: t('邀请好友注册，好友充值后您可获得相应奖励'),
    },
    {
      markerClassName: 'bg-green-500',
      text: t('返利会自动进入余额，无需划转。'),
    },
    {
      markerClassName: 'bg-gray-300',
      text: t('邀请的好友越多，获得的奖励越多'),
    },
  ];
  const modeTitle =
    mode === 'rebate'
      ? t(
          '邀请好友注册，好友前 {{count}} 次充值每次均可为您带来 {{rate}}% 返利。',
          {
            count: rebateTopupCount,
            rate: rebateRate,
          },
        )
      : mode === 'fixed'
        ? t('固定注册奖励')
        : t('邀请奖励已关闭');
  const ModeIcon =
    mode === 'rebate' ? BadgePercent : mode === 'fixed' ? Gift : PauseCircle;

  return (
    <Card className='!rounded-xl shadow-sm border-0 h-full'>
      <div className='flex items-start justify-between gap-3 mb-4'>
        <div className='flex min-w-0 items-center'>
          <Avatar size='small' color='green' className='mr-3 shrink-0'>
            <Gift size={16} />
          </Avatar>
          <div className='min-w-0'>
            <Typography.Text className='text-lg font-medium'>
              {t('邀请奖励')}
            </Typography.Text>
            <div className='text-xs text-gray-500'>{modeTitle}</div>
          </div>
        </div>
        <Tag
          color={mode === 'disabled' ? 'grey' : 'green'}
          className='shrink-0'
        >
          {mode === 'rebate'
            ? t('充值返利')
            : mode === 'fixed'
              ? t('注册奖励')
              : t('已关闭')}
        </Tag>
      </div>

      <div className='rounded-lg bg-semi-color-fill-0 p-3 flex gap-3'>
        <ModeIcon
          size={19}
          className='text-semi-color-primary shrink-0 mt-0.5'
        />
        <div className='min-w-0'>
          <Text strong>{modeTitle}</Text>
          <div className='text-xs text-gray-500 mt-1'>
            {mode === 'rebate'
              ? t('返利会自动进入余额，无需划转。')
              : mode === 'fixed'
                ? t('注册奖励会进入待划转额度。')
                : t('历史统计、流水和待划转额度仍然保留。')}
          </div>
        </div>
      </div>

      <div className='grid grid-cols-3 gap-2 py-5 text-center'>
        {[
          [WalletCards, t('待使用收益'), renderQuota(pendingQuota)],
          [Gift, t('总收益'), renderQuota(totalQuota)],
          [Users, t('邀请人数'), String(inviteCount)],
        ].map(([Icon, label, value]) => (
          <div key={label} className='min-w-0'>
            <Icon size={16} className='mx-auto text-gray-500' />
            <div className='font-semibold mt-1 break-words'>{value}</div>
            <div className='text-xs text-gray-500 leading-tight break-words'>
              {label}
            </div>
          </div>
        ))}
      </div>

      {mode !== 'disabled' && (
        <div className='mb-3 space-y-1.5'>
          <Text className='text-sm font-medium'>{t('邀请链接')}</Text>
          <div className='flex items-center gap-2'>
            <Input
              value={affLink}
              readonly
              className='!rounded-lg min-w-0 flex-1'
            />
            <Button
              type='primary'
              theme='borderless'
              className='shrink-0'
              onClick={handleAffLinkClick}
              icon={<Copy size={14} />}
              aria-label={t('复制')}
            />
          </div>
        </div>
      )}

      {mode === 'rebate' && (
        <>
          <Divider margin='16px' />
          <div className='space-y-3'>
            <div className='flex items-center gap-2 font-medium'>
              <BadgePercent size={16} className='text-semi-color-primary' />
              {t('奖励说明')}
            </div>
            <ul className='space-y-3'>
              {rebateDetails.map((detail) => (
                <li key={detail.text} className='flex items-start gap-3'>
                  <span
                    aria-hidden='true'
                    className={`${detail.markerClassName} mt-1.5 h-2 w-2 shrink-0 rounded-full`}
                  />
                  <Text type='tertiary' size='small' className='leading-5'>
                    {detail.text}
                  </Text>
                </li>
              ))}
            </ul>
          </div>
        </>
      )}

      {pendingQuota > 0 && (
        <div className='mb-3'>
          <Button
            block
            type='primary'
            icon={<Zap size={14} />}
            disabled={!complianceConfirmed}
            onClick={() => setOpenTransfer(true)}
          >
            {t('划转到余额')}
          </Button>
          {!complianceConfirmed && (
            <div className='text-xs text-gray-500 mt-1'>
              {t('邀请奖励划转已禁用，管理员需先确认合规声明。')}
            </div>
          )}
        </div>
      )}

      <Divider margin='16px' />

      <div className='flex items-center justify-between gap-2 mb-2'>
        <div className='flex items-center gap-2 font-medium'>
          <History size={16} />
          {t('最近返利流水')}
        </div>
        {rewardTotal > 0 && (
          <Text type='tertiary' size='small'>
            {t('共 {{count}} 条', { count: rewardTotal })}
          </Text>
        )}
      </div>

      {rewards?.length ? (
        <div className='divide-y divide-semi-color-border'>
          {rewards.map((reward) => (
            <div
              key={reward.id}
              className='grid grid-cols-[minmax(0,1fr)_auto] gap-3 py-3 text-xs'
            >
              <div className='min-w-0'>
                <div className='font-medium truncate'>
                  {reward.invitee} ·{' '}
                  {t('第 {{count}} 次充值', {
                    count: reward.topup_ordinal,
                  })}
                </div>
                <div className='text-gray-500 truncate mt-1'>
                  {t('到账 {{quota}}', {
                    quota: renderQuota(reward.credited_quota),
                  })}{' '}
                  · {timestamp2string(reward.created_at)}
                </div>
              </div>
              <div className='text-right'>
                <div className='font-semibold text-semi-color-primary'>
                  +{renderQuota(reward.reward_quota)}
                </div>
                <div className='text-gray-500 mt-1'>
                  {(reward.rebate_bps / 100).toFixed(2).replace(/\.00$/, '')}%
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className='text-center text-xs text-gray-500 py-8'>
          {t('暂无充值返利流水')}
        </div>
      )}

      {rewardTotal > rewardPageSize && (
        <div className='flex justify-center mt-3'>
          <Pagination
            currentPage={rewardPage}
            pageSize={rewardPageSize}
            total={rewardTotal}
            size='small'
            showSizeChanger={false}
            onPageChange={onRewardPageChange}
          />
        </div>
      )}
    </Card>
  );
};

export default InvitationCard;
