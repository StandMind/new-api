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

import { resolveOpenLuxSelection } from './selection'
import type { OpenLuxPreview } from './types'

const preview: OpenLuxPreview = {
  source_hash: 'source',
  local_fingerprint: 'local',
  binding_revision: 1,
  fetched_at: 1,
  changes: [
    {
      id: 'billing',
      kind: 'model_billing_update',
      actionable: true,
      destructive: false,
      requires: null,
      blocked_reasons: null,
    },
    {
      id: 'price',
      kind: 'group_model_price_update',
      actionable: true,
      destructive: false,
      requires: ['billing'],
      blocked_reasons: null,
    },
    {
      id: 'blocked-removal',
      kind: 'group_model_remove',
      actionable: false,
      destructive: true,
      requires: null,
      blocked_reasons: ['route reference'],
    },
  ],
  notices: [],
  summary: { actionable: 2, blocked: 1, notices: 0, price: 1, removal: 1 },
}

describe('OpenLux preview selection', () => {
  test('starts with no selected changes', () => {
    assert.deepEqual([...resolveOpenLuxSelection(preview, new Set())], [])
  })

  test('automatically includes required model billing changes', () => {
    assert.deepEqual(
      [...resolveOpenLuxSelection(preview, new Set(['price']))].sort(),
      ['billing', 'price']
    )
  })

  test('does not allow blocked changes into the resolved selection', () => {
    assert.deepEqual(
      [...resolveOpenLuxSelection(preview, new Set(['blocked-removal']))],
      []
    )
  })
})
