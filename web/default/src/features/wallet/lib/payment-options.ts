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
import { PAYMENT_TYPES } from '../constants'
import type {
  PresetAmount,
  TopupInfo,
  UnifiedPaymentOption,
  WaffoPayMethod,
} from '../types'

export type PaymentOptionUnavailableReason =
  | { kind: 'amounts-unconfigured' }
  | { kind: 'minimum'; amount: number }
  | { kind: 'amount-unavailable' }

export type PaymentDispatch =
  | { kind: 'generic'; paymentType: string }
  | { kind: 'waffo'; payMethodIndex: number }
  | { kind: 'waffo-pancake' }
  | { kind: 'creem' }

export function normalizeTopupAmounts(values: unknown[]): number[] {
  const seen = new Set<number>()
  const normalized: number[] = []
  for (const value of values) {
    const amount = Number(value)
    if (!Number.isFinite(amount) || amount <= 0 || seen.has(amount)) {
      continue
    }
    seen.add(amount)
    normalized.push(amount)
  }
  return normalized
}

export function resolveFixedTopupAmount(
  presetAmounts: PresetAmount[],
  currentAmount: number
): number | null {
  if (presetAmounts.length === 0) {
    return null
  }
  if (presetAmounts.some((preset) => preset.value === currentAmount)) {
    return currentAmount
  }
  return presetAmounts[0].value
}

function appendWaffoOptions(
  options: UnifiedPaymentOption[],
  methods: WaffoPayMethod[],
  minTopup: number
): void {
  methods.forEach((method, index) => {
    options.push({
      id: `waffo:${index}`,
      kind: 'waffo',
      name: method.name,
      type: PAYMENT_TYPES.WAFFO,
      icon: method.icon,
      minTopup,
      payMethodIndex: index,
    })
  })
}

export function buildUnifiedPaymentOptions(
  topupInfo: TopupInfo | null,
  topupAmount: number
): UnifiedPaymentOption[] {
  if (!topupInfo) {
    return []
  }

  const options: UnifiedPaymentOption[] = []
  const waffoMethods = topupInfo.enable_waffo_topup
    ? topupInfo.waffo_pay_methods || []
    : []
  let waffoExpanded = false

  topupInfo.pay_methods.forEach((method, index) => {
    if (method.type === PAYMENT_TYPES.WAFFO) {
      if (!waffoExpanded) {
        appendWaffoOptions(
          options,
          waffoMethods,
          topupInfo.waffo_min_topup || 0
        )
        waffoExpanded = true
      }
      return
    }

    if (method.type === PAYMENT_TYPES.CREEM) {
      return
    }

    if (
      method.type === PAYMENT_TYPES.STRIPE &&
      !topupInfo.enable_stripe_topup
    ) {
      return
    }

    if (
      method.type === PAYMENT_TYPES.WAFFO_PANCAKE &&
      !topupInfo.enable_waffo_pancake_topup
    ) {
      return
    }

    const isDedicatedProvider =
      method.type === PAYMENT_TYPES.STRIPE ||
      method.type === PAYMENT_TYPES.WAFFO_PANCAKE
    if (!isDedicatedProvider && !topupInfo.enable_online_topup) {
      return
    }

    let minTopup = Math.max(method.min_topup || 0, topupInfo.min_topup || 0)
    if (method.type === PAYMENT_TYPES.STRIPE) {
      minTopup = Math.max(
        method.min_topup || 0,
        topupInfo.stripe_min_topup || 0
      )
    } else if (method.type === PAYMENT_TYPES.WAFFO_PANCAKE) {
      minTopup = Math.max(
        method.min_topup || 0,
        topupInfo.waffo_pancake_min_topup || 0
      )
    }

    options.push({
      id: `standard:${method.type}:${index}`,
      kind: 'standard',
      name: method.name,
      type: method.type,
      icon: method.icon,
      minTopup,
    })
  })

  if (!waffoExpanded && waffoMethods.length > 0) {
    appendWaffoOptions(options, waffoMethods, topupInfo.waffo_min_topup || 0)
  }

  if (topupInfo.enable_creem_topup) {
    const matchingProducts = (topupInfo.creem_products || []).filter(
      (product) => product.topupAmount === topupAmount
    )
    options.push({
      id: 'creem',
      kind: 'creem',
      name: 'Creem',
      type: PAYMENT_TYPES.CREEM,
      minTopup: 0,
      product: matchingProducts.length === 1 ? matchingProducts[0] : undefined,
      testMode: topupInfo.creem_test_mode === true,
    })
  }

  return options
}

export function getPaymentOptionUnavailableReason(
  option: UnifiedPaymentOption,
  topupAmount: number,
  hasConfiguredAmounts: boolean
): PaymentOptionUnavailableReason | null {
  if (!hasConfiguredAmounts) {
    return { kind: 'amounts-unconfigured' }
  }
  if (option.minTopup > topupAmount) {
    return { kind: 'minimum', amount: option.minTopup }
  }
  if (option.kind === 'creem' && !option.product) {
    return { kind: 'amount-unavailable' }
  }
  return null
}

export function getPaymentDispatch(
  option: UnifiedPaymentOption
): PaymentDispatch {
  if (option.kind === 'creem') {
    return { kind: 'creem' }
  }
  if (option.kind === 'waffo') {
    return { kind: 'waffo', payMethodIndex: option.payMethodIndex }
  }
  if (option.type === PAYMENT_TYPES.WAFFO_PANCAKE) {
    return { kind: 'waffo-pancake' }
  }
  return { kind: 'generic', paymentType: option.type }
}
