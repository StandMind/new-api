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

import type { PricingModel, RequestPriceTier } from '../types'
import { formatRequestPrice } from './price'

function requestPricedModel(tiers?: RequestPriceTier[]): PricingModel {
  return {
    id: 1,
    model_name: 'resolution-priced-model',
    quota_type: 1,
    model_ratio: 0,
    completion_ratio: 0,
    model_price: 0.33,
    enable_groups: ['default'],
    group_ratio: { default: 1 },
    request_price_policy: tiers
      ? {
          dimension: 'image_resolution',
          default_value: '1K',
          tiers,
        }
      : undefined,
  }
}

describe('request price ranges', () => {
  test('uses backend policy multipliers for Pro and Flash ranges', () => {
    assert.equal(
      formatRequestPrice(
        requestPricedModel([
          { value: '1K', multiplier: 1 },
          { value: '2K', multiplier: 1 },
          { value: '4K', multiplier: 1.79 },
        ]),
        false,
        1,
        1,
        'default'
      ),
      '$0.33 - $0.5907'
    )
    assert.equal(
      formatRequestPrice(
        requestPricedModel([
          { value: '512', multiplier: 0.66696 },
          { value: '1K', multiplier: 1 },
          { value: '2K', multiplier: 1 },
          { value: '4K', multiplier: 1.78571 },
        ]),
        false,
        1,
        1,
        'default'
      ),
      '$0.2201 - $0.5893'
    )
  })

  test('keeps a single price when no request policy is present', () => {
    assert.equal(
      formatRequestPrice(requestPricedModel(), false, 1, 1, 'default'),
      '$0.33'
    )
  })
})
