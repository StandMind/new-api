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

import {
  buildGeminiImageRequestBody,
  buildOpenAIImageRequestBody,
} from './image-api-samples'

const MODEL_RESOLUTIONS: Record<string, string[]> = {
  'gemini-3-pro-image': ['1K', '2K', '4K'],
  'gemini-3-pro-image-preview': ['1K', '2K', '4K'],
  'gemini-3.1-flash-image': ['512', '1K', '2K', '4K'],
  'gemini-3.1-flash-image-preview': ['512', '1K', '2K', '4K'],
}

describe('resolution-priced image API samples', () => {
  for (const [model, resolutions] of Object.entries(MODEL_RESOLUTIONS)) {
    for (const resolution of resolutions) {
      test(`${model} ${resolution} generates valid protocol payloads`, () => {
        const gemini = JSON.parse(buildGeminiImageRequestBody(resolution))
        assert.deepEqual(gemini.generationConfig.responseModalities, [
          'TEXT',
          'IMAGE',
        ])
        assert.equal(gemini.generationConfig.imageConfig.aspectRatio, '1:1')
        assert.equal(gemini.generationConfig.imageConfig.imageSize, resolution)

        const openai = JSON.parse(
          buildOpenAIImageRequestBody(model, resolution)
        )
        assert.equal(openai.model, model)
        assert.equal(openai.extra_body.google.image_config.aspect_ratio, '1:1')
        assert.equal(
          openai.extra_body.google.image_config.image_size,
          resolution
        )
      })
    }
  }
})
