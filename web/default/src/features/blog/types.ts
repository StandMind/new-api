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
import type { InterfaceLanguageCode } from '@/i18n/languages'

export type BlogPostStatus = 'draft' | 'published'

export type BlogPost = {
  id: number
  slug: string
  title: string
  summary: string
  content?: string
  tags: string[]
  cover_image?: string
  author?: string
  published_time: number
  updated_time: number
  locale: InterfaceLanguageCode
  reading_minutes: number
}

export type BlogPostTranslation = {
  title: string
  summary: string
  content: string
}

export type BlogPostAdmin = {
  id: number
  slug: string
  status: BlogPostStatus
  tags: string[]
  cover_image: string
  author: string
  published_time: number
  created_time: number
  updated_time: number
  translations: Partial<Record<InterfaceLanguageCode, BlogPostTranslation>>
}

export type BlogPostPayload = Omit<
  BlogPostAdmin,
  'created_time' | 'updated_time'
>

export type BlogPage<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type BlogResponse<T> = {
  success: boolean
  message: string
  data: T
}
