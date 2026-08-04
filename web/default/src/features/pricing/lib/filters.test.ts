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

import { DEFAULT_SORT_OPTION, SORT_OPTIONS } from '../constants'
import type { PricingModel, PricingVendor } from '../types'
import { sortModels, sortVendors } from './filters'

function pricingModel(name: string, releaseDate?: string): PricingModel {
  return {
    id: 1,
    model_name: name,
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    release_date: releaseDate,
  }
}

function pricingVendor(id: number, name: string): PricingVendor {
  return { id, name }
}

describe('pricing vendor sorting', () => {
  test('puts primary vendors first and sorts the rest by name', () => {
    const vendors = [
      pricingVendor(1, 'Zhipu'),
      pricingVendor(2, 'anthropic'),
      pricingVendor(3, 'Baidu'),
      pricingVendor(4, 'OpenAI'),
      pricingVendor(5, 'azure'),
      pricingVendor(6, 'Google'),
    ]

    const sorted = sortVendors(vendors)

    assert.deepEqual(
      sorted.map((vendor) => vendor.name),
      ['OpenAI', 'Google', 'anthropic', 'azure', 'Baidu', 'Zhipu']
    )
    assert.deepEqual(
      vendors.map((vendor) => vendor.name),
      ['Zhipu', 'anthropic', 'Baidu', 'OpenAI', 'azure', 'Google']
    )
  })
})

describe('pricing model sorting', () => {
  test('uses release date as the default sort option', () => {
    assert.equal(DEFAULT_SORT_OPTION, SORT_OPTIONS.RELEASE_DATE)
  })

  test('sorts newest releases first and missing dates last', () => {
    const sorted = sortModels(
      [
        pricingModel('missing'),
        pricingModel('older', '2025-01-01'),
        pricingModel('newer-b', '2026-07-31'),
        pricingModel('invalid', 'not-a-date'),
        pricingModel('newer-a', '2026-07-31'),
      ],
      SORT_OPTIONS.RELEASE_DATE
    )

    assert.deepEqual(
      sorted.map((model) => model.model_name),
      ['newer-a', 'newer-b', 'older', 'invalid', 'missing']
    )
  })
})
