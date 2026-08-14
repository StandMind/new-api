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
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'
import { formatLocalCurrencyAmount } from '@/lib/currency'
import { formatPercent } from '@/lib/format'

import { DEFAULT_DISCOUNT_RATE } from '../../constants'
import {
  formatCreemPrice,
  formatTopupCreditAmount,
  getPaymentIcon,
  getTopupDiscountBreakdown,
} from '../../lib'
import type { UnifiedPaymentOption } from '../../types'

interface PaymentConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  topupAmount: number
  paymentAmount: number
  paymentOption: UnifiedPaymentOption | undefined
  calculating: boolean
  processing: boolean
  discountRate?: number
}

export function PaymentConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  topupAmount,
  paymentAmount,
  paymentOption,
  calculating,
  processing,
  discountRate = DEFAULT_DISCOUNT_RATE,
}: PaymentConfirmDialogProps) {
  const { t } = useTranslation()
  const { currency } = useSystemConfig()
  const creemProduct =
    paymentOption?.kind === 'creem' ? paymentOption.product : undefined
  const discountBreakdown = getTopupDiscountBreakdown(
    paymentAmount,
    discountRate
  )
  let formattedPaymentAmount = formatLocalCurrencyAmount(paymentAmount)
  let formattedOriginalAmount = discountBreakdown
    ? formatLocalCurrencyAmount(discountBreakdown.originalAmount)
    : ''
  let formattedSavingsAmount = discountBreakdown
    ? formatLocalCurrencyAmount(discountBreakdown.savingsAmount)
    : ''

  if (creemProduct) {
    formattedPaymentAmount = formatCreemPrice(
      paymentAmount,
      creemProduct.currency
    )
    if (discountBreakdown) {
      formattedOriginalAmount = formatCreemPrice(
        discountBreakdown.originalAmount,
        creemProduct.currency
      )
      formattedSavingsAmount = formatCreemPrice(
        discountBreakdown.savingsAmount,
        creemProduct.currency
      )
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'>
        <AlertDialogHeader>
          <AlertDialogTitle className='text-xl font-semibold'>
            {t('Confirm Payment')}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t('Review your payment details')}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <dl className='space-y-3 py-3 sm:space-y-4 sm:py-4'>
          <div className='flex items-center justify-between gap-4'>
            <dt className='text-muted-foreground text-sm'>
              {t('Topup Amount')}
            </dt>
            <dd className='text-lg font-semibold'>
              {formatTopupCreditAmount(topupAmount, currency)}
            </dd>
          </div>

          {discountBreakdown && (
            <div className='flex items-center justify-between gap-4'>
              <dt className='text-muted-foreground text-sm'>{t('Discount')}</dt>
              <dd className='font-semibold text-emerald-600 dark:text-emerald-400'>
                {formatPercent(discountBreakdown.savingsPercent)}
              </dd>
            </div>
          )}

          {discountBreakdown && (
            <div className='flex items-center justify-between gap-4'>
              <dt className='text-muted-foreground text-sm'>
                {t('Original Price')}
              </dt>
              <dd className='text-muted-foreground font-medium line-through'>
                {formattedOriginalAmount}
              </dd>
            </div>
          )}

          <div className='flex items-center justify-between gap-4'>
            <dt className='text-muted-foreground text-sm'>{t('You Pay')}</dt>
            {calculating ? (
              <dd>
                <Skeleton className='h-6 w-24' />
              </dd>
            ) : (
              <dd className='text-2xl font-semibold'>
                {formattedPaymentAmount}
              </dd>
            )}
          </div>

          {discountBreakdown && !calculating && (
            <div className='flex items-center justify-between gap-4'>
              <dt className='text-muted-foreground text-sm'>{t('You save')}</dt>
              <dd className='font-semibold text-emerald-600 dark:text-emerald-400'>
                {formattedSavingsAmount}
              </dd>
            </div>
          )}

          <div className='border-t pt-4'>
            <div className='flex items-center justify-between gap-4'>
              <dt className='text-muted-foreground text-sm'>
                {t('Payment Method')}
              </dt>
              <dd className='flex min-w-0 items-center gap-2'>
                {getPaymentIcon(
                  paymentOption?.type,
                  'h-4 w-4',
                  paymentOption?.icon,
                  paymentOption?.name
                )}
                <span className='truncate font-medium'>
                  {paymentOption?.name}
                </span>
              </dd>
            </div>
          </div>
        </dl>

        <AlertDialogFooter className='grid grid-cols-2 gap-2 sm:flex'>
          <AlertDialogCancel disabled={processing}>
            {t('Cancel')}
          </AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={processing}>
            {processing && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Confirm Payment')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
