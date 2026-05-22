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
export type DocumentationNavItem = {
  slug?: string
  title: string
  description?: string
  children?: DocumentationNavItem[]
}

export type DocumentationConfig = {
  enabled: boolean
  locale: string
  default_locale: string
  default_slug: string
  supported_locales: string[]
  nav: DocumentationNavItem[]
  pages?: DocumentationPageItem[]
}

export type DocumentationPageItem = {
  path: string
  slug: string
  title: string
  description?: string
}

export type DocumentationDebugExample = {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  path: string
  path_template?: string
  auth?: 'bearer' | 'anthropic' | string
  model?: string
  headers?: Record<string, string>
  body?: unknown
}

export type DocumentationPageData = {
  enabled: boolean
  slug: string
  title: string
  content: string
  locale: string
  debug?: DocumentationDebugExample
}

export type DocumentationResponse<T> = {
  success: boolean
  message: string
  data?: T
}
