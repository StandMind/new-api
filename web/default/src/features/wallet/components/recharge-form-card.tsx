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
import { ExternalLink, Gift, Loader2, Receipt, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

import {
  formatTopupCreditAmount,
  getPaymentIcon,
  getPaymentOptionUnavailableReason,
} from '../lib'
import type { PresetAmount, TopupInfo, UnifiedPaymentOption } from '../types'

interface RechargeFormCardProps {
  topupInfo: TopupInfo | null
  presetAmounts: PresetAmount[]
  selectedPreset: number | null
  onSelectPreset: (preset: PresetAmount) => void
  topupAmount: number
  paymentOptions: UnifiedPaymentOption[]
  onPaymentOptionSelect: (option: UnifiedPaymentOption) => void
  paymentLoading: string | null
  redemptionCode: string
  onRedemptionCodeChange: (code: string) => void
  onRedeem: () => void
  redeeming: boolean
  topupLink?: string
  loading?: boolean
  onOpenBilling?: () => void
}

function LoadingCard() {
  return (
    <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
      <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
        <Skeleton className='h-6 w-32' />
        <Skeleton className='mt-2 h-4 w-48' />
      </CardHeader>
      <CardContent className='space-y-4 p-3 sm:space-y-6 sm:p-5'>
        <div className='space-y-3'>
          <Skeleton className='h-3 w-16' />
          <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
            {Array.from({ length: 8 }, (_, index) => `preset-${index}`).map(
              (key) => (
                <Skeleton key={key} className='h-16 rounded-lg' />
              )
            )}
          </div>
        </div>
        <div className='space-y-3'>
          <Skeleton className='h-3 w-32' />
          <div className='grid grid-cols-2 gap-3 lg:grid-cols-3'>
            {['primary', 'secondary', 'tertiary'].map((key) => (
              <Skeleton key={key} className='h-14 rounded-lg' />
            ))}
          </div>
        </div>
        <div className='space-y-3 border-t pt-6'>
          <Skeleton className='h-3 w-24' />
          <div className='flex gap-2'>
            <Skeleton className='h-10 flex-1' />
            <Skeleton className='h-10 w-20' />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function RechargeFormCard(props: RechargeFormCardProps) {
  const { t } = useTranslation()
  const { currency } = useSystemConfig()
  const hasConfiguredAmounts = props.presetAmounts.length > 0
  const hasTopupProvider = Boolean(
    props.topupInfo?.enable_online_topup ||
    props.topupInfo?.enable_stripe_topup ||
    props.topupInfo?.enable_creem_topup ||
    props.topupInfo?.enable_waffo_topup ||
    props.topupInfo?.enable_waffo_pancake_topup
  )
  const redemptionEnabled = props.topupInfo?.enable_redemption !== false

  if (props.loading) {
    return <LoadingCard />
  }

  return (
    <TitledCard
      title={t('Add Funds')}
      description={t('Choose an amount and payment method')}
      icon={<WalletCards className='h-4 w-4' />}
      iconTone='success'
      disableHoverEffect
      action={
        props.onOpenBilling ? (
          <Button
            variant='outline'
            size='sm'
            onClick={props.onOpenBilling}
            className='w-full gap-2 sm:w-auto'
          >
            <Receipt className='h-4 w-4' />
            {t('Order History')}
          </Button>
        ) : null
      }
      contentClassName='space-y-4 sm:space-y-6'
    >
      {hasTopupProvider ? (
        <div className='space-y-4 sm:space-y-6'>
          <div className='space-y-2.5 sm:space-y-3'>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Amount')}
            </Label>
            {hasConfiguredAmounts ? (
              <div className='grid grid-cols-2 gap-1.5 sm:gap-3 md:grid-cols-4'>
                {props.presetAmounts.map((preset) => (
                  <Button
                    key={preset.value}
                    variant='outline'
                    disabled={Boolean(props.paymentLoading)}
                    className={cn(
                      'min-h-14 rounded-lg px-3 py-2.5 text-base font-semibold sm:min-h-16 sm:px-4 sm:text-lg',
                      props.selectedPreset === preset.value
                        ? 'border-foreground bg-foreground/5 dark:border-foreground dark:bg-foreground/10'
                        : 'border-muted'
                    )}
                    onClick={() => props.onSelectPreset(preset)}
                  >
                    {formatTopupCreditAmount(preset.value, currency)}
                  </Button>
                ))}
              </div>
            ) : (
              <Alert>
                <AlertDescription>
                  {t(
                    'No recharge amounts are configured. Please contact administrator.'
                  )}
                </AlertDescription>
              </Alert>
            )}
          </div>

          <div className='space-y-2.5 sm:space-y-3'>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Payment Method')}
            </Label>
            {props.paymentOptions.length > 0 ? (
              <div className='grid grid-cols-2 gap-1.5 sm:gap-3 lg:grid-cols-3'>
                {props.paymentOptions.map((option) => {
                  const unavailable = getPaymentOptionUnavailableReason(
                    option,
                    props.topupAmount,
                    hasConfiguredAmounts
                  )
                  let disabledReason: string | undefined
                  if (unavailable?.kind === 'amounts-unconfigured') {
                    disabledReason = t('Recharge amount is not configured')
                  } else if (unavailable?.kind === 'minimum') {
                    disabledReason = t('Minimum topup amount: {{amount}}', {
                      amount: formatTopupCreditAmount(
                        unavailable.amount,
                        currency
                      ),
                    })
                  } else if (unavailable?.kind === 'amount-unavailable') {
                    disabledReason = t('This amount is unavailable')
                  }

                  const secondaryLabel =
                    disabledReason ||
                    (option.kind === 'creem' && option.testMode
                      ? t('Test Mode')
                      : undefined)
                  const button = (
                    <Button
                      key={option.id}
                      variant='outline'
                      onClick={() => props.onPaymentOptionSelect(option)}
                      disabled={Boolean(unavailable) || !!props.paymentLoading}
                      title={disabledReason}
                      aria-label={
                        disabledReason
                          ? `${option.name}. ${disabledReason}`
                          : option.name
                      }
                      className='min-h-14 min-w-0 justify-start gap-2 rounded-lg px-3 py-2 text-left'
                    >
                      {props.paymentLoading === option.id ? (
                        <Loader2 className='h-4 w-4 shrink-0 animate-spin' />
                      ) : (
                        getPaymentIcon(
                          option.type,
                          'h-4 w-4 shrink-0',
                          option.icon,
                          option.name
                        )
                      )}
                      <span className='flex min-w-0 flex-col items-start gap-0.5'>
                        <span className='max-w-full truncate'>
                          {option.name}
                        </span>
                        {secondaryLabel && (
                          <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                            {secondaryLabel}
                          </span>
                        )}
                      </span>
                    </Button>
                  )

                  return disabledReason ? (
                    <TooltipProvider key={option.id}>
                      <Tooltip>
                        <TooltipTrigger render={button} />
                        <TooltipContent>{disabledReason}</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  ) : (
                    button
                  )
                })}
              </div>
            ) : (
              <Alert>
                <AlertDescription>
                  {t(
                    'No payment methods available. Please contact administrator.'
                  )}
                </AlertDescription>
              </Alert>
            )}
          </div>
        </div>
      ) : (
        <Alert>
          <AlertDescription>
            {t(
              'Online topup is not enabled. Please use redemption code or contact administrator.'
            )}
          </AlertDescription>
        </Alert>
      )}

      {redemptionEnabled ? (
        <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='warning' size='xs'>
              <Gift />
            </IconBadge>
            <Label
              htmlFor='redemption-code'
              className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
            >
              {t('Have a Code?')}
            </Label>
          </div>
          <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'>
            <Input
              id='redemption-code'
              value={props.redemptionCode}
              onChange={(event) =>
                props.onRedemptionCodeChange(event.target.value)
              }
              placeholder={t('Enter your redemption code')}
              className='h-9 min-w-0'
            />
            <Button
              onClick={props.onRedeem}
              disabled={props.redeeming}
              variant='outline'
              className='h-9 px-4'
            >
              {props.redeeming && (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              )}
              {t('Redeem')}
            </Button>
          </div>
          {props.topupLink && (
            <p className='text-muted-foreground text-xs'>
              {t('Need a redemption code?')}{' '}
              <a
                href={props.topupLink}
                target='_blank'
                rel='noopener noreferrer'
                className='inline-flex items-center gap-1 underline-offset-4 hover:underline'
              >
                {t('Get one here')}
                <ExternalLink className='h-3 w-3' />
              </a>
            </p>
          )}
        </div>
      ) : (
        <Alert className='border-t'>
          <AlertDescription>
            {t(
              'Redemption codes are disabled until the administrator confirms compliance terms.'
            )}
          </AlertDescription>
        </Alert>
      )}
    </TitledCard>
  )
}
