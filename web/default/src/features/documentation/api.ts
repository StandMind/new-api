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
import { normalizeInterfaceLanguage } from '@/i18n/languages'
import { api } from '@/lib/api'
import type {
  DocumentationConfig,
  DocumentationPageData,
  DocumentationResponse,
} from './types'

function documentationRequestConfig(lang?: string) {
  const locale = normalizeInterfaceLanguage(lang)

  return {
    params: { lang: locale },
    skipBusinessError: true,
  } as Record<string, unknown>
}

export async function getDocumentationConfig(lang?: string) {
  const res = await api.get<DocumentationResponse<DocumentationConfig>>(
    '/api/docs/config',
    documentationRequestConfig(lang)
  )
  return res.data
}

export async function getDocumentationPage(slug: string, lang?: string) {
  const normalizedSlug = slug.replace(/^\/+/, '')
  const res = await api.get<DocumentationResponse<DocumentationPageData>>(
    `/api/docs/page/${encodeURI(normalizedSlug)}`,
    documentationRequestConfig(lang)
  )
  return res.data
}
