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

import type { TFunction } from 'i18next'

import type { ApiKey } from '../types'
import {
  getApiKeyFormDefaultValues,
  getApiKeyFormSchema,
  transformApiKeyToFormDefaults,
  transformFormDataToPayload,
} from './api-key-form'

const translate = ((key: string) => key) as TFunction

describe('API key smart routing form', () => {
  test('defaults to the configured smart routing priority', () => {
    const values = getApiKeyFormDefaultValues('success_rate')

    assert.equal(values.smart_routing, true)
    assert.equal(values.routing_priority, 'success_rate')
  })

  test('keeps the compatibility chain while publishing the selected mode', () => {
    const values = getApiKeyFormDefaultValues('speed')
    values.name = 'smart key'
    values.group = 'group-a'
    values.group_chain = ['group-a', 'group-b']

    assert.deepEqual(transformFormDataToPayload(values), {
      name: 'smart key',
      remain_quota: 0,
      expired_time: -1,
      unlimited_quota: true,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
      group: 'group-a',
      group_chain: ['group-a', 'group-b'],
      routing_priority: 'speed',
    })
  })

  test('requires a visible group chain only in manual mode', () => {
    const schema = getApiKeyFormSchema(translate)
    const smart = getApiKeyFormDefaultValues('price')
    smart.name = 'smart key'
    smart.group_chain = []
    assert.equal(schema.safeParse(smart).success, true)

    smart.smart_routing = false
    const manualResult = schema.safeParse(smart)
    assert.equal(manualResult.success, false)
    if (!manualResult.success) {
      assert.equal(manualResult.error.issues[0]?.path[0], 'group_chain')
    }
  })

  test('restores an existing smart mode when editing', () => {
    const key: ApiKey = {
      id: 1,
      name: 'existing',
      key: 'masked',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 1,
      group: 'group-a',
      group_chain: ['group-a', 'group-b'],
      routing_priority: 'auto',
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    }

    const values = transformApiKeyToFormDefaults(key)
    assert.equal(values.smart_routing, true)
    assert.equal(values.routing_priority, 'auto')
    assert.deepEqual(values.group_chain, ['group-a', 'group-b'])
  })
})
