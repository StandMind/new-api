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
  BlogPage,
  BlogPost,
  BlogPostAdmin,
  BlogPostPayload,
  BlogPostStats,
  BlogResponse,
  BlogPostStatus,
} from './types'

export type BlogListParams = {
  lang?: string
  page?: number
  pageSize?: number
  keyword?: string
  tag?: string
}

function publicBlogRequestConfig(params: BlogListParams = {}) {
  return {
    params: {
      lang: normalizeInterfaceLanguage(params.lang),
      p: params.page ?? 1,
      page_size: params.pageSize ?? 12,
      keyword: params.keyword || undefined,
      tag: params.tag || undefined,
    },
    skipBusinessError: true,
  } as Record<string, unknown>
}

export async function getBlogPosts(params: BlogListParams = {}) {
  const res = await api.get<BlogResponse<BlogPage<BlogPost>>>(
    '/api/blog/posts',
    publicBlogRequestConfig(params)
  )
  return res.data
}

export async function getBlogPost(slug: string, lang?: string) {
  const res = await api.get<BlogResponse<BlogPost>>(
    `/api/blog/posts/${encodeURIComponent(slug)}`,
    {
      params: { lang: normalizeInterfaceLanguage(lang) },
      skipBusinessError: true,
    } as Record<string, unknown>
  )
  return res.data
}

export async function recordBlogPostView(slug: string) {
  const res = await api.post<BlogResponse<null>>(
    `/api/blog/posts/${encodeURIComponent(slug)}/view`,
    {},
    { skipBusinessError: true, skipErrorHandler: true } as Record<string, unknown>
  )
  return res.data
}

export async function getAdminBlogPosts(params: {
  page?: number
  pageSize?: number
  keyword?: string
  status?: BlogPostStatus | ''
}) {
  const res = await api.get<BlogResponse<BlogPage<BlogPostAdmin>>>(
    '/api/blog/admin/posts',
    {
      params: {
        p: params.page ?? 1,
        page_size: params.pageSize ?? 20,
        keyword: params.keyword || undefined,
        status: params.status || undefined,
      },
    }
  )
  return res.data
}

export async function getAdminBlogPostStats(
  id: number,
  params: {
    days?: number
    startDate?: string
    endDate?: string
  } = {}
) {
  const res = await api.get<BlogResponse<BlogPostStats>>(
    `/api/blog/admin/posts/${id}/stats`,
    {
      params: {
        days: params.days ?? 30,
        start_date: params.startDate || undefined,
        end_date: params.endDate || undefined,
      },
    }
  )
  return res.data
}

export async function createAdminBlogPost(payload: BlogPostPayload) {
  const res = await api.post<BlogResponse<BlogPostAdmin>>(
    '/api/blog/admin/posts',
    payload
  )
  return res.data
}

export async function updateAdminBlogPost(payload: BlogPostPayload) {
  const res = await api.put<BlogResponse<BlogPostAdmin>>(
    `/api/blog/admin/posts/${payload.id}`,
    payload
  )
  return res.data
}

export async function deleteAdminBlogPost(id: number) {
  const res = await api.delete<BlogResponse<null>>(
    `/api/blog/admin/posts/${id}`
  )
  return res.data
}
