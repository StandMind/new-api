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

import type { PricingModel } from '../types'
import { getOfficialPriceSavings } from './official-price'

function pricingModel(officialPrice: number): PricingModel {
  return {
    id: 1,
    model_name: 'test-model',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['test-group'],
    official_price: {
      unit: 'usd_per_million_input_tokens',
      tiers: [{ price: officialPrice }],
    },
  }
}

describe('official price savings', () => {
  test('returns a badge comparison only when the local price is cheaper', () => {
    const model = pricingModel(10)

    assert.equal(
      getOfficialPriceSavings(model, 8, 'usd_per_million_input_tokens', 0, [
        null,
      ])?.percent,
      '20'
    )
    assert.equal(
      getOfficialPriceSavings(model, 10, 'usd_per_million_input_tokens', 0, [
        null,
      ]),
      null
    )
    assert.equal(
      getOfficialPriceSavings(model, 12, 'usd_per_million_input_tokens', 0, [
        null,
      ]),
      null
    )
  })
})
