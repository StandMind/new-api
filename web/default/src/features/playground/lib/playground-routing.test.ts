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

import { DEFAULT_CONFIG, DEFAULT_PARAMETER_ENABLED } from '../constants'
import type { Message, PlaygroundConfig } from '../types'
import { PLAYGROUND_INPUT_GROUP_CLASS_NAME } from './input/input-control-utils'
import { getPlaygroundModelQuery } from './options/playground-option-utils'
import { migratePlaygroundConfig } from './state/playground-state-utils'
import { buildChatCompletionPayload } from './streaming/payload-builder'

const messages: Message[] = [
  {
    key: 'message-1',
    from: 'user',
    versions: [{ id: 'version-1', content: 'hello' }],
  },
]

function config(
  routingPriority: PlaygroundConfig['routing_priority']
): PlaygroundConfig {
  return {
    ...DEFAULT_CONFIG,
    model: 'test-model',
    group: 'manual-group',
    routing_priority: routingPriority,
  }
}

describe('Playground smart routing', () => {
  test('defaults new configurations to price routing', () => {
    assert.equal(DEFAULT_CONFIG.routing_priority, 'price')
  })

  test('preserves a stored manual group from the legacy config shape', () => {
    assert.deepEqual(migratePlaygroundConfig({ group: 'manual-group' }), {
      group: 'manual-group',
      routing_priority: '',
    })
    assert.deepEqual(
      migratePlaygroundConfig({
        group: 'manual-group',
        routing_priority: 'speed',
      }),
      {
        group: 'manual-group',
        routing_priority: 'speed',
      }
    )
  })

  test('publishes mutually exclusive manual and smart routing fields', () => {
    const smartPayload = buildChatCompletionPayload(
      messages,
      config('auto'),
      DEFAULT_PARAMETER_ENABLED
    )
    assert.equal(smartPayload.routing_priority, 'auto')
    assert.equal('route_group' in smartPayload, false)

    const manualPayload = buildChatCompletionPayload(
      messages,
      config(''),
      DEFAULT_PARAMETER_ENABLED
    )
    assert.equal(manualPayload.route_group, 'manual-group')
    assert.equal('routing_priority' in manualPayload, false)
  })

  test('shares one model query across smart modes and scopes manual queries', () => {
    const price = getPlaygroundModelQuery('manual-group', 'price')
    const speed = getPlaygroundModelQuery('other-group', 'speed')
    assert.deepEqual(price.queryKey, speed.queryKey)
    assert.equal(price.routeGroup, undefined)

    const manual = getPlaygroundModelQuery('manual-group', '')
    assert.deepEqual(manual.queryKey, [
      'playground-models',
      'manual',
      'manual-group',
    ])
    assert.equal(manual.routeGroup, 'manual-group')
  })

  test('keeps the composer opaque when its send button is disabled', () => {
    assert.match(PLAYGROUND_INPUT_GROUP_CLASS_NAME, /has-disabled:opacity-100/)
    assert.match(
      PLAYGROUND_INPUT_GROUP_CLASS_NAME,
      /has-disabled:bg-background\/95/
    )
  })
})
