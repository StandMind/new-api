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
import { formatLocalCurrencyAmount } from '@/lib/currency'
import { formatNumber } from '@/lib/format'
import type { CurrencyConfig } from '@/stores/system-config-store'

// ============================================================================
// Wallet-specific Formatting Functions
// ============================================================================

/**
 * Format Creem price with currency symbol (USD/EUR)
 */
export function formatCreemPrice(
  price: number,
  currency: 'USD' | 'EUR'
): string {
  const symbol = currency === 'EUR' ? '€' : '$'
  return `${symbol}${price.toFixed(2)}`
}

/**
 * Format large quota numbers with K/M suffix
 */
export function formatQuotaShort(quota: number): string {
  if (quota >= 1000000) {
    return `${(quota / 1000000).toFixed(1)}M`
  }
  if (quota >= 1000) {
    return `${(quota / 1000).toFixed(1)}K`
  }
  return quota.toString()
}

export function formatTopupCreditAmount(
  amount: number,
  currency: CurrencyConfig | undefined
): string {
  if (currency?.quotaDisplayType === 'TOKENS') {
    return formatNumber(amount)
  }

  let exchangeRate = 1
  if (currency?.quotaDisplayType === 'CNY') {
    exchangeRate = currency.usdExchangeRate || 1
  } else if (currency?.quotaDisplayType === 'CUSTOM') {
    exchangeRate = currency.customCurrencyExchangeRate || 1
  }

  return formatLocalCurrencyAmount(amount * exchangeRate, {
    digitsLarge: 2,
    digitsSmall: 2,
    abbreviate: false,
  })
}
