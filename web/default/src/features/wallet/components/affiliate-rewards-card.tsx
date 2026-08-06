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
  BadgePercent,
  ChevronLeft,
  ChevronRight,
  Gift,
  History,
  PauseCircle,
  Share2,
  Users,
  WalletCards,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

import type { InvitationInfo, InvitationReward, UserWalletData } from '../types'

interface AffiliateRewardsCardProps {
  user: UserWalletData | null
  info: InvitationInfo | null
  rewards: InvitationReward[]
  rewardPage: number
  rewardPageSize: number
  rewardTotal: number
  affiliateLink: string
  onRewardPageChange: (page: number) => void
  onTransfer: () => void
  complianceConfirmed?: boolean
  loading?: boolean
  rewardsLoading?: boolean
}

export function AffiliateRewardsCard({
  user,
  info,
  rewards,
  rewardPage,
  rewardPageSize,
  rewardTotal,
  affiliateLink,
  onRewardPageChange,
  onTransfer,
  complianceConfirmed = true,
  loading,
  rewardsLoading,
}: AffiliateRewardsCardProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card data-card-hover='false' className='h-full'>
        <CardContent className='space-y-5 p-5'>
          <Skeleton className='h-7 w-44' />
          <Skeleton className='h-20 w-full rounded-lg' />
          <Skeleton className='h-16 w-full rounded-lg' />
          <Skeleton className='h-40 w-full rounded-lg' />
        </CardContent>
      </Card>
    )
  }

  const mode = info?.mode ?? 'disabled'
  const pendingQuota = info?.pending_reward_quota ?? user?.aff_quota ?? 0
  const totalQuota = info?.total_reward_quota ?? user?.aff_history_quota ?? 0
  const inviteCount = info?.invite_count ?? user?.aff_count ?? 0
  const hasPendingRewards = pendingQuota > 0
  const totalPages = Math.max(1, Math.ceil(rewardTotal / rewardPageSize))
  const rebateRate = (info?.rebate_bps ?? 0) / 100

  let modeSummary = t('New referral rewards are currently disabled.')
  let modeLabel = t('Disabled')
  let modeDescription = t('Existing rewards and history remain available.')
  let ModeIcon = PauseCircle
  if (mode === 'rebate') {
    modeSummary = t('{{rate}}% back on the first {{count}} wallet top-ups', {
      rate: rebateRate,
      count: info?.rebate_topup_count ?? 0,
    })
    modeLabel = t('Top-up rebate')
    modeDescription = t('Rewards are added to your balance automatically.')
    ModeIcon = BadgePercent
  } else if (mode === 'fixed') {
    modeSummary = t(
      'Earn {{inviterReward}} when a referred user registers; they receive {{inviteeReward}}.',
      {
        inviterReward: formatQuota(info?.fixed_inviter_quota ?? 0),
        inviteeReward: formatQuota(info?.fixed_invitee_quota ?? 0),
      }
    )
    modeLabel = t('Registration reward')
    modeDescription = t(
      'Registration rewards can be transferred to your balance.'
    )
    ModeIcon = Gift
  }

  let rewardsContent = (
    <div className='text-muted-foreground py-6 text-center text-xs'>
      {t('No top-up rewards yet')}
    </div>
  )
  if (rewardsLoading) {
    rewardsContent = (
      <div className='space-y-2'>
        <Skeleton className='h-12 w-full rounded-lg' />
        <Skeleton className='h-12 w-full rounded-lg' />
      </div>
    )
  } else if (rewards.length > 0) {
    rewardsContent = (
      <div className='divide-y'>
        {rewards.map((reward) => (
          <div
            key={reward.id}
            className='grid grid-cols-[minmax(0,1fr)_auto] gap-3 py-2.5 text-xs'
          >
            <div className='min-w-0'>
              <div className='truncate font-medium'>
                {reward.invitee} ·{' '}
                {t('Top-up #{{count}}', {
                  count: reward.topup_ordinal,
                })}
              </div>
              <div className='text-muted-foreground mt-1 truncate'>
                {t('Credited {{quota}}', {
                  quota: formatQuota(reward.credited_quota),
                })}{' '}
                · {formatTimestampToDate(reward.created_at)}
              </div>
            </div>
            <div className='text-right'>
              <div className='text-primary font-semibold'>
                +{formatQuota(reward.reward_quota)}
              </div>
              <div className='text-muted-foreground mt-1'>
                {(reward.rebate_bps / 100).toFixed(2).replace(/\.00$/, '')}%
              </div>
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <Card data-card-hover='false' className='h-full overflow-hidden'>
      <CardHeader className='gap-3 border-b px-4 py-4 sm:px-5'>
        <div className='flex items-center justify-between gap-3'>
          <CardTitle className='flex min-w-0 items-center gap-2 text-base'>
            <Share2 className='text-primary size-5 shrink-0' />
            <span className='truncate'>{t('Referral Program')}</span>
          </CardTitle>
          <Badge variant={mode === 'disabled' ? 'secondary' : 'default'}>
            {modeLabel}
          </Badge>
        </div>

        <div className='bg-muted/55 flex gap-3 rounded-lg p-3'>
          <ModeIcon
            className={
              mode === 'disabled'
                ? 'text-muted-foreground mt-0.5 size-5 shrink-0'
                : 'text-primary mt-0.5 size-5 shrink-0'
            }
          />
          <div className='min-w-0'>
            <p className='text-sm font-medium'>{modeSummary}</p>
            <p className='text-muted-foreground mt-1 text-xs'>
              {modeDescription}
            </p>
          </div>
        </div>
      </CardHeader>

      <CardContent className='space-y-5 p-4 sm:p-5'>
        <div className='grid grid-cols-3 gap-2 text-center'>
          {[
            {
              icon: WalletCards,
              label: t('Pending'),
              value: formatQuota(pendingQuota),
            },
            {
              icon: Gift,
              label: t('Total Earned'),
              value: formatQuota(totalQuota),
            },
            { icon: Users, label: t('Invites'), value: String(inviteCount) },
          ].map(({ icon: Icon, label, value }) => (
            <div key={label} className='min-w-0'>
              <Icon className='text-muted-foreground mx-auto size-4' />
              <div className='mt-1 truncate text-sm font-semibold tabular-nums'>
                {value}
              </div>
              <div className='text-muted-foreground mt-0.5 truncate text-[11px]'>
                {label}
              </div>
            </div>
          ))}
        </div>

        {(mode !== 'disabled' || hasPendingRewards) && <Separator />}

        {mode !== 'disabled' ? (
          <div className='space-y-2'>
            <div className='text-sm font-medium'>{t('Referral link')}</div>
            <div className='flex items-center gap-2'>
              <Input
                value={affiliateLink}
                readOnly
                className='h-9 min-w-0 flex-1 font-mono text-xs'
              />
              <CopyButton
                value={affiliateLink}
                variant='outline'
                className='size-9 shrink-0'
                iconClassName='size-4'
                tooltip={t('Copy referral link')}
                aria-label={t('Copy referral link')}
              />
            </div>
          </div>
        ) : null}

        {hasPendingRewards ? (
          <div className='space-y-2'>
            <Button
              onClick={onTransfer}
              disabled={!complianceConfirmed}
              className='w-full'
              size='sm'
            >
              <WalletCards data-icon='inline-start' />
              {t('Transfer to Balance')}
            </Button>
            {!complianceConfirmed ? (
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Referral reward transfer is disabled until the administrator confirms compliance terms.'
                )}
              </p>
            ) : null}
          </div>
        ) : null}

        <Separator />

        <div className='space-y-3'>
          <div className='flex items-center justify-between gap-3'>
            <div className='flex items-center gap-2 text-sm font-medium'>
              <History className='size-4' />
              {t('Recent rewards')}
            </div>
            {rewardTotal > 0 ? (
              <span className='text-muted-foreground text-xs tabular-nums'>
                {t('Page {{page}} of {{pages}}', {
                  page: rewardPage,
                  pages: totalPages,
                })}
              </span>
            ) : null}
          </div>

          {rewardsContent}

          {totalPages > 1 ? (
            <div className='flex justify-end gap-1'>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Previous')}
                title={t('Previous')}
                disabled={rewardPage <= 1 || rewardsLoading}
                onClick={() => onRewardPageChange(rewardPage - 1)}
              >
                <ChevronLeft />
              </Button>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Next')}
                title={t('Next')}
                disabled={rewardPage >= totalPages || rewardsLoading}
                onClick={() => onRewardPageChange(rewardPage + 1)}
              >
                <ChevronRight />
              </Button>
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}
