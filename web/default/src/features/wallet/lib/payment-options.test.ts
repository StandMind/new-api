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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import type { TopupInfo } from '../types'
import { getTopupDiscountBreakdown, getTopupSavingsPercent } from './payment'
import {
  buildUnifiedPaymentOptions,
  getPaymentDispatch,
  getPaymentOptionUnavailableReason,
  normalizeTopupAmounts,
  resolveFixedTopupAmount,
} from './payment-options'

function topupInfoFixture(): TopupInfo {
  return {
    enable_online_topup: true,
    enable_stripe_topup: true,
    pay_methods: [
      { name: 'Alipay', type: 'alipay', min_topup: 5 },
      { name: 'Waffo', type: 'waffo', min_topup: 10 },
      { name: 'Stripe', type: 'stripe', min_topup: 2 },
      { name: 'Waffo Pancake', type: 'waffo_pancake', min_topup: 3 },
    ],
    min_topup: 1,
    stripe_min_topup: 2,
    amount_options: [10, 20],
    discount: {},
    enable_creem_topup: true,
    creem_test_mode: true,
    creem_products: [
      {
        name: 'Ten',
        productId: 'prod_10',
        price: 9,
        quota: 1000,
        currency: 'USD',
        topupAmount: 10,
      },
    ],
    enable_waffo_topup: true,
    waffo_min_topup: 10,
    enable_waffo_pancake_topup: true,
    waffo_pancake_min_topup: 3,
    waffo_pay_methods: [
      { name: 'Visa', payMethodType: 'card', payMethodName: 'visa' },
      {
        name: 'Mastercard',
        payMethodType: 'card',
        payMethodName: 'mastercard',
      },
    ],
  }
}

describe('wallet payment options', () => {
  test('calculates exact savings labels and confirmation amounts', () => {
    assert.equal(getTopupSavingsPercent(0.99), 1)
    assert.equal(getTopupSavingsPercent(0.985), 1.5)
    assert.equal(getTopupSavingsPercent(0.965), 3.5)
    assert.equal(getTopupSavingsPercent(1), null)
    assert.equal(getTopupSavingsPercent(0), null)

    assert.deepEqual(getTopupDiscountBreakdown(193, 0.965), {
      savingsPercent: 3.5,
      originalAmount: 200,
      savingsAmount: 7,
    })
    assert.equal(getTopupDiscountBreakdown(200, 1), null)
  })

  test('normalizes fixed amounts without changing their order', () => {
    assert.deepEqual(
      normalizeTopupAmounts([0, 20, '10', 20, -1, 'bad', 50]),
      [20, 10, 50]
    )
  })

  test('selects the first configured amount and preserves a valid selection', () => {
    const amounts = [{ value: 25 }, { value: 10 }]

    assert.equal(resolveFixedTopupAmount(amounts, 0), 25)
    assert.equal(resolveFixedTopupAmount(amounts, 10), 10)
    assert.equal(resolveFixedTopupAmount(amounts, 15), 25)
    assert.equal(resolveFixedTopupAmount([], 10), null)
  })

  test('expands legacy Waffo in place and appends Creem', () => {
    const options = buildUnifiedPaymentOptions(topupInfoFixture(), 10)

    assert.deepEqual(
      options.map((option) => option.name),
      ['Alipay', 'Visa', 'Mastercard', 'Stripe', 'Waffo Pancake', 'Creem']
    )
    const creem = options.at(-1)
    assert.equal(creem?.kind, 'creem')
    if (creem?.kind === 'creem') {
      assert.equal(creem.product?.productId, 'prod_10')
      assert.equal(creem.testMode, true)
    }
  })

  test('keeps only enabled providers and applies provider minimums', () => {
    const info = topupInfoFixture()
    info.min_topup = 8
    info.stripe_min_topup = 12
    info.waffo_pancake_min_topup = 15
    info.pay_methods.unshift({ name: 'Creem placeholder', type: 'creem' })

    const options = buildUnifiedPaymentOptions(info, 20)
    assert.deepEqual(
      options.map((option) => [option.name, option.minTopup]),
      [
        ['Alipay', 8],
        ['Visa', 10],
        ['Mastercard', 10],
        ['Stripe', 12],
        ['Waffo Pancake', 15],
        ['Creem', 0],
      ]
    )

    info.enable_online_topup = false
    info.enable_stripe_topup = false
    info.enable_waffo_pancake_topup = false
    assert.deepEqual(
      buildUnifiedPaymentOptions(info, 20).map((option) => option.name),
      ['Visa', 'Mastercard', 'Creem']
    )
  })

  test('marks Creem unavailable when an amount has zero or multiple products', () => {
    const info = topupInfoFixture()
    const missing = buildUnifiedPaymentOptions(info, 20).at(-1)
    assert.ok(missing)
    assert.deepEqual(getPaymentOptionUnavailableReason(missing, 20, true), {
      kind: 'amount-unavailable',
    })

    info.creem_products?.push({
      ...info.creem_products[0],
      productId: 'prod_10_duplicate',
    })
    const duplicate = buildUnifiedPaymentOptions(info, 10).at(-1)
    assert.ok(duplicate)
    assert.deepEqual(getPaymentOptionUnavailableReason(duplicate, 10, true), {
      kind: 'amount-unavailable',
    })
  })

  test('keeps minimum and missing-amount availability rules explicit', () => {
    const option = buildUnifiedPaymentOptions(topupInfoFixture(), 10)[0]
    assert.deepEqual(getPaymentOptionUnavailableReason(option, 0, false), {
      kind: 'amounts-unconfigured',
    })
    assert.deepEqual(getPaymentOptionUnavailableReason(option, 2, true), {
      kind: 'minimum',
      amount: 5,
    })
    assert.equal(getPaymentOptionUnavailableReason(option, 10, true), null)
  })

  test('dispatches each provider through its dedicated payment path', () => {
    const options = buildUnifiedPaymentOptions(topupInfoFixture(), 10)
    assert.deepEqual(getPaymentDispatch(options[0]), {
      kind: 'generic',
      paymentType: 'alipay',
    })
    assert.deepEqual(getPaymentDispatch(options[1]), {
      kind: 'waffo',
      payMethodIndex: 0,
    })
    assert.deepEqual(getPaymentDispatch(options[4]), {
      kind: 'waffo-pancake',
    })
    assert.deepEqual(getPaymentDispatch(options[5]), { kind: 'creem' })
  })
})
