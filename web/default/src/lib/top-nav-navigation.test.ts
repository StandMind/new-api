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

import { getDocumentNavigation } from './top-nav-navigation'

describe('top navigation document handoff', () => {
  test('reloads local Docs and Blog routes through the reverse proxy', () => {
    assert.deepEqual(getDocumentNavigation('/docs'), {
      external: false,
      reloadDocument: true,
    })
    assert.deepEqual(getDocumentNavigation('/blog'), {
      external: false,
      reloadDocument: true,
    })
  })

  test('uses a same-tab anchor for a fully configured Docs URL', () => {
    assert.deepEqual(getDocumentNavigation('https://docs.example.com'), {
      external: true,
      reloadDocument: true,
    })
  })
})
