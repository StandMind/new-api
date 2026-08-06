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
import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'

import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { getSelf } from '@/lib/api'

import {
  getInvitationInfo,
  getInvitationRewards,
  transferAffiliateQuota,
} from '../api'
import { generateAffiliateLink } from '../lib'
import type { InvitationInfo, InvitationReward } from '../types'

// ============================================================================
// Affiliate Hook
// ============================================================================

export function useAffiliate() {
  const [info, setInfo] = useState<InvitationInfo | null>(null)
  const [rewards, setRewards] = useState<InvitationReward[]>([])
  const [rewardPage, setRewardPage] = useState(1)
  const [rewardTotal, setRewardTotal] = useState(0)
  const [affiliateLink, setAffiliateLink] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [rewardsLoading, setRewardsLoading] = useState(true)
  const [transferring, setTransferring] = useState(false)
  const { copyToClipboard } = useCopyToClipboard()
  const rewardPageSize = 5

  const fetchInvitationInfo = useCallback(async () => {
    const response = await getInvitationInfo()
    if (response.success && response.data) {
      setInfo(response.data)
      setAffiliateLink(generateAffiliateLink(response.data.code))
    }
  }, [])

  const fetchRewards = useCallback(async (page: number) => {
    try {
      setRewardsLoading(true)
      const response = await getInvitationRewards(page, rewardPageSize)
      if (response.success && response.data) {
        setRewards(response.data.items ?? [])
        setRewardTotal(response.data.total ?? 0)
        setRewardPage(response.data.page ?? page)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch referral rewards:', error)
    } finally {
      setRewardsLoading(false)
    }
  }, [])

  const fetchAffiliateData = useCallback(async () => {
    try {
      setLoading(true)
      await Promise.all([fetchInvitationInfo(), fetchRewards(1)])
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch referral program:', error)
    } finally {
      setLoading(false)
    }
  }, [fetchInvitationInfo, fetchRewards])

  // Copy affiliate link
  const copyAffiliateLink = useCallback(() => {
    copyToClipboard(affiliateLink)
  }, [affiliateLink, copyToClipboard])

  // Transfer affiliate quota to balance
  const transferQuota = useCallback(
    async (quota: number): Promise<boolean> => {
      try {
        setTransferring(true)
        const response = await transferAffiliateQuota({ quota })

        if (response.success) {
          toast.success(response.message || i18next.t('Transfer successful'))
          await Promise.all([getSelf(), fetchInvitationInfo()])
          return true
        }

        toast.error(response.message || i18next.t('Transfer failed'))
        return false
      } catch {
        toast.error(i18next.t('Transfer failed'))
        return false
      } finally {
        setTransferring(false)
      }
    },
    [fetchInvitationInfo]
  )

  useEffect(() => {
    fetchAffiliateData()
  }, [fetchAffiliateData])

  return {
    affiliateCode: info?.code ?? '',
    affiliateLink,
    info,
    rewards,
    rewardPage,
    rewardPageSize,
    rewardTotal,
    loading,
    rewardsLoading,
    transferring,
    copyAffiliateLink,
    transferQuota,
    setRewardPage: fetchRewards,
    refetch: fetchAffiliateData,
  }
}
