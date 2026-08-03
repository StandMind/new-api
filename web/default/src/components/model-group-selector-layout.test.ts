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

import { scrollSelectedOptionIntoView } from './model-group-selector-layout'

describe('model group selector layout', () => {
  test('resets an auto-scrolled mobile container to the selected smart mode', () => {
    let requestedTop: number | undefined
    const container = {
      clientHeight: 602,
      scrollTop: 1874,
      scrollTo(options: ScrollToOptions) {
        requestedTop = options.top
      },
    }
    const selectedOption = {
      offsetHeight: 40,
      offsetTop: 52,
      scrollIntoView() {},
    }

    scrollSelectedOptionIntoView(selectedOption, container)

    assert.equal(requestedTop, 0)
  })

  test('centers a selected manual group in the mobile container', () => {
    let requestedTop: number | undefined
    const container = {
      clientHeight: 400,
      scrollTop: 0,
      scrollTo(options: ScrollToOptions) {
        requestedTop = options.top
      },
    }
    const selectedOption = {
      offsetHeight: 40,
      offsetTop: 640,
      scrollIntoView() {},
    }

    scrollSelectedOptionIntoView(selectedOption, container)

    assert.equal(requestedTop, 460)
  })
})
