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
import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { getSelf } from '@/lib/api'

import { AffiliateRewardsCard } from './components/affiliate-rewards-card'
import { BillingHistoryDialog } from './components/dialogs/billing-history-dialog'
import { PaymentConfirmDialog } from './components/dialogs/payment-confirm-dialog'
import { TransferDialog } from './components/dialogs/transfer-dialog'
import { RechargeFormCard } from './components/recharge-form-card'
import { SubscriptionPlansCard } from './components/subscription-plans-card'
import { WalletStatsCard } from './components/wallet-stats-card'
import { DEFAULT_DISCOUNT_RATE } from './constants'
import {
  useTopupInfo,
  usePayment,
  useAffiliate,
  useRedemption,
  useCreemPayment,
  useWaffoPayment,
  useWaffoPancakePayment,
} from './hooks'
import {
  buildUnifiedPaymentOptions,
  getPaymentDispatch,
  getPaymentOptionUnavailableReason,
  resolveFixedTopupAmount,
} from './lib'
import type {
  UserWalletData,
  PresetAmount,
  UnifiedPaymentOption,
} from './types'

interface WalletProps {
  initialShowHistory?: boolean
}

export function Wallet(props: WalletProps) {
  const { t } = useTranslation()
  const [user, setUser] = useState<UserWalletData | null>(null)
  const [userLoading, setUserLoading] = useState(true)
  const [topupAmount, setTopupAmount] = useState(0)
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null)
  const [selectedPaymentOption, setSelectedPaymentOption] =
    useState<UnifiedPaymentOption>()
  const [paymentLoading, setPaymentLoading] = useState<string | null>(null)
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false)
  const [transferDialogOpen, setTransferDialogOpen] = useState(false)
  const [billingDialogOpen, setBillingDialogOpen] = useState(false)
  const [redemptionCode, setRedemptionCode] = useState('')
  const [showSubscriptionPanel, setShowSubscriptionPanel] = useState(true)

  const { topupInfo, presetAmounts, loading: topupLoading } = useTopupInfo()
  const {
    amount: paymentAmount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    setAmount: setPaymentAmount,
  } = usePayment()
  const {
    affiliateLink,
    info: affiliateInfo,
    rewards: affiliateRewards,
    rewardPage,
    rewardPageSize,
    rewardTotal,
    loading: affiliateLoading,
    rewardsLoading,
    transferQuota,
    transferring,
    setRewardPage,
  } = useAffiliate()
  const showAffiliate = Boolean(
    !affiliateLoading && affiliateInfo && affiliateInfo.mode !== 'disabled'
  )
  const { redeeming, redeemCode } = useRedemption()
  const { processing: creemProcessing, processCreemPayment } = useCreemPayment()
  const { processing: waffoProcessing, processWaffoPayment } = useWaffoPayment()
  const { processing: pancakeProcessing, processWaffoPancakePayment } =
    useWaffoPancakePayment()

  // Fetch and refresh user data
  const fetchUser = useCallback(async () => {
    try {
      setUserLoading(true)
      const response = await getSelf()
      if (response.success && response.data) {
        setUser(response.data as UserWalletData)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch user data:', error)
    } finally {
      setUserLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  useEffect(() => {
    if (props.initialShowHistory) {
      setBillingDialogOpen(true)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [props.initialShowHistory])

  const paymentOptions = useMemo(
    () => buildUnifiedPaymentOptions(topupInfo, topupAmount),
    [topupInfo, topupAmount]
  )

  // Fixed amounts are the only allowed source of wallet topup quantities.
  useEffect(() => {
    const resolvedAmount = resolveFixedTopupAmount(presetAmounts, topupAmount)
    if (resolvedAmount === null) {
      if (topupAmount !== 0) setTopupAmount(0)
      if (selectedPreset !== null) setSelectedPreset(null)
      setSelectedPaymentOption(undefined)
      setPaymentAmount(0)
      setConfirmDialogOpen(false)
      return
    }

    if (resolvedAmount !== topupAmount) {
      setTopupAmount(resolvedAmount)
      setSelectedPreset(resolvedAmount)
      setSelectedPaymentOption(undefined)
      setPaymentAmount(0)
      setConfirmDialogOpen(false)
      return
    }
    if (selectedPreset !== resolvedAmount) {
      setSelectedPreset(resolvedAmount)
    }
  }, [presetAmounts, selectedPreset, setPaymentAmount, topupAmount])

  // Handle preset selection
  const handleSelectPreset = (preset: PresetAmount) => {
    setTopupAmount(preset.value)
    setSelectedPreset(preset.value)
    setSelectedPaymentOption(undefined)
    setPaymentAmount(0)
    setConfirmDialogOpen(false)
  }

  const handlePaymentOptionSelect = async (option: UnifiedPaymentOption) => {
    const unavailable = getPaymentOptionUnavailableReason(
      option,
      topupAmount,
      presetAmounts.length > 0
    )
    if (unavailable) return

    setSelectedPaymentOption(option)
    setPaymentLoading(option.id)

    try {
      if (option.kind === 'creem') {
        if (!option.product || option.product.price <= 0) {
          setSelectedPaymentOption(undefined)
          toast.error(t('This amount is unavailable'))
          return
        }
        setPaymentAmount(option.product.price)
        setConfirmDialogOpen(true)
        return
      }

      const quotedAmount = await calculatePaymentAmount(
        topupAmount,
        option.type
      )
      if (quotedAmount <= 0) {
        setSelectedPaymentOption(undefined)
        toast.error(t('Unable to calculate payment amount'))
        return
      }
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  // Handle payment confirmation
  const handlePaymentConfirm = async () => {
    if (!selectedPaymentOption) return

    const dispatch = getPaymentDispatch(selectedPaymentOption)
    let success = false
    if (dispatch.kind === 'creem') {
      success = await processCreemPayment(topupAmount)
    } else if (dispatch.kind === 'waffo') {
      success = await processWaffoPayment(topupAmount, dispatch.payMethodIndex)
    } else if (dispatch.kind === 'waffo-pancake') {
      success = await processWaffoPancakePayment(topupAmount)
    } else {
      success = await processPayment(topupAmount, dispatch.paymentType)
    }

    if (success) {
      setConfirmDialogOpen(false)
      await fetchUser()
    }
  }

  // Handle redemption
  const handleRedeem = async () => {
    if (!redemptionCode) return

    const success = await redeemCode(redemptionCode)
    if (success) {
      setRedemptionCode('')
      await fetchUser()
    }
  }

  // Handle transfer
  const handleTransfer = async (amount: number) => {
    const success = await transferQuota(amount)
    if (success) {
      await fetchUser()
    }
    return success
  }

  // Get discount rate for current topup amount
  const getDiscountRate = useCallback(() => {
    return topupInfo?.discount?.[topupAmount] || DEFAULT_DISCOUNT_RATE
  }, [topupInfo, topupAmount])

  const handleSubscriptionAvailabilityChange = useCallback(
    (available: boolean) => {
      setShowSubscriptionPanel(available)
    },
    []
  )

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Wallet')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='mx-auto flex w-full max-w-7xl flex-col gap-4 sm:gap-5'>
            <WalletStatsCard user={user} loading={userLoading} />

            <div
              className={
                showAffiliate
                  ? 'grid gap-4 xl:grid-cols-[minmax(0,1.08fr)_minmax(380px,0.92fr)] xl:items-start'
                  : 'grid gap-4'
              }
            >
              <div className='flex min-w-0 flex-col gap-4'>
                <div id='wallet-add-funds' className='scroll-mt-4'>
                  <RechargeFormCard
                    topupInfo={topupInfo}
                    presetAmounts={presetAmounts}
                    selectedPreset={selectedPreset}
                    onSelectPreset={handleSelectPreset}
                    topupAmount={topupAmount}
                    paymentOptions={paymentOptions}
                    onPaymentOptionSelect={handlePaymentOptionSelect}
                    paymentLoading={paymentLoading}
                    redemptionCode={redemptionCode}
                    onRedemptionCodeChange={setRedemptionCode}
                    onRedeem={handleRedeem}
                    redeeming={redeeming}
                    topupLink={topupInfo?.topup_link}
                    loading={topupLoading}
                    onOpenBilling={() => setBillingDialogOpen(true)}
                  />
                </div>

                <div className={showSubscriptionPanel ? 'block' : 'hidden'}>
                  <SubscriptionPlansCard
                    topupInfo={topupInfo}
                    onAvailabilityChange={handleSubscriptionAvailabilityChange}
                    userQuota={user?.quota}
                    onPurchaseSuccess={fetchUser}
                  />
                </div>
              </div>

              {showAffiliate ? (
                <AffiliateRewardsCard
                  user={user}
                  info={affiliateInfo}
                  rewards={affiliateRewards}
                  rewardPage={rewardPage}
                  rewardPageSize={rewardPageSize}
                  rewardTotal={rewardTotal}
                  affiliateLink={affiliateLink}
                  onRewardPageChange={setRewardPage}
                  onTransfer={() => setTransferDialogOpen(true)}
                  complianceConfirmed={
                    topupInfo?.payment_compliance_confirmed !== false
                  }
                  rewardsLoading={rewardsLoading}
                />
              ) : null}
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <PaymentConfirmDialog
        open={confirmDialogOpen}
        onOpenChange={setConfirmDialogOpen}
        onConfirm={handlePaymentConfirm}
        topupAmount={topupAmount}
        paymentAmount={paymentAmount}
        paymentOption={selectedPaymentOption}
        calculating={calculating}
        processing={
          processing || creemProcessing || waffoProcessing || pancakeProcessing
        }
        discountRate={getDiscountRate()}
      />

      <TransferDialog
        open={transferDialogOpen}
        onOpenChange={setTransferDialogOpen}
        onConfirm={handleTransfer}
        availableQuota={user?.aff_quota ?? 0}
        transferring={transferring}
      />

      <BillingHistoryDialog
        open={billingDialogOpen}
        onOpenChange={setBillingDialogOpen}
      />
    </>
  )
}
