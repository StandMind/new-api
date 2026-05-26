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
import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { BarChart3, Edit, Plus, Search, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from 'recharts'
import {
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
  type InterfaceLanguageCode,
} from '@/i18n/languages'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { TagInput } from '@/components/tag-input'
import {
  createAdminBlogPost,
  deleteAdminBlogPost,
  getAdminBlogPostStats,
  getAdminBlogPosts,
  updateAdminBlogPost,
} from '@/features/blog/api'
import type {
  BlogPostAdmin,
  BlogPostPayload,
  BlogPostStats,
  BlogPostStatus,
  BlogPostTranslation,
} from '@/features/blog/types'
import { SettingsSection } from '../components/settings-section'
import { toast } from 'sonner'

const BLOG_PAGE_SIZE = 20
const DEFAULT_BLOG_STATS_DAYS = 30
const DAY_MS = 24 * 60 * 60 * 1000

const emptyTranslation: BlogPostTranslation = {
  title: '',
  summary: '',
  content: '',
}

function createEmptyPost(): BlogPostPayload {
  return {
    id: 0,
    slug: '',
    status: 'draft',
    tags: [],
    cover_image: '',
    author: '',
    published_time: 0,
    translations: {
      en: { ...emptyTranslation },
    },
  }
}

function clonePost(post?: BlogPostAdmin): BlogPostPayload {
  if (!post) return createEmptyPost()
  return {
    id: post.id,
    slug: post.slug,
    status: post.status,
    tags: [...(post.tags ?? [])],
    cover_image: post.cover_image ?? '',
    author: post.author ?? '',
    published_time: post.published_time ?? 0,
    translations: Object.fromEntries(
      Object.entries(post.translations ?? {}).map(([locale, value]) => [
        locale,
        {
          title: value?.title ?? '',
          summary: value?.summary ?? '',
          content: value?.content ?? '',
        },
      ])
    ) as BlogPostPayload['translations'],
  }
}

function timestampToInput(value: number) {
  if (!value) return ''
  const date = new Date(value * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 16)
}

function inputToTimestamp(value: string) {
  if (!value) return 0
  return Math.floor(new Date(value).getTime() / 1000)
}

function dateToInputDate(value: Date) {
  return value.toISOString().slice(0, 10)
}

function timestampToDateInput(value: number) {
  if (!value) return ''
  return dateToInputDate(new Date(value * 1000))
}

function getDefaultStatsRange(post?: BlogPostAdmin) {
  const endDate = dateToInputDate(new Date())
  const defaultStart = dateToInputDate(
    new Date(new Date(`${endDate}T00:00:00Z`).getTime() - (DEFAULT_BLOG_STATS_DAYS - 1) * DAY_MS)
  )
  const publishedDate = timestampToDateInput(post?.published_time ?? 0)
  let startDate = defaultStart
  if (publishedDate && publishedDate > defaultStart) {
    startDate = publishedDate > endDate ? endDate : publishedDate
  }
  return { startDate, endDate }
}

function normalizeStatsRange(range: { startDate: string; endDate: string }) {
  if (!range.startDate || !range.endDate || range.startDate <= range.endDate) {
    return range
  }
  return { startDate: range.endDate, endDate: range.startDate }
}

function formatDateTime(value: number, locale: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value * 1000))
}

function formatBlogStatsDate(value: string, locale: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat(locale, {
    month: 'short',
    day: 'numeric',
  }).format(new Date(`${value}T12:00:00Z`))
}

function formatNumber(value: number | undefined, locale: string) {
  return new Intl.NumberFormat(locale).format(value ?? 0)
}

function statusLabel(status: BlogPostStatus, t: (key: string) => string) {
  return status === 'published' ? t('Published') : t('Draft')
}

function getLocalizedBlogTranslation(post: BlogPostAdmin, locale: string) {
  return post.translations[locale as InterfaceLanguageCode] ??
    post.translations.en ??
    post.translations.zh
}

type BlogStatsDialogProps = {
  open: boolean
  post?: BlogPostAdmin
  stats?: BlogPostStats
  loading: boolean
  error: boolean
  rangeStart: string
  rangeEnd: string
  locale: string
  onRangeChange: (range: { startDate: string; endDate: string }) => void
  onResetRange: () => void
  onOpenChange: (open: boolean) => void
}

function BlogStatsDialog({
  open,
  post,
  stats,
  loading,
  error,
  rangeStart,
  rangeEnd,
  locale,
  onRangeChange,
  onResetRange,
  onOpenChange,
}: BlogStatsDialogProps) {
  const { t } = useTranslation()
  const translation = post ? getLocalizedBlogTranslation(post, locale) : undefined
  const daily = stats?.daily ?? []
  const hasChartRange = daily.length > 0
  const chartData = daily.map((point) => ({
    ...point,
    label: formatBlogStatsDate(point.date, locale),
  }))
  const chartConfig = {
    views: {
      label: t('Views'),
      color: 'var(--primary)',
    },
  } satisfies ChartConfig

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[92svh] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{t('Blog post stats')}</DialogTitle>
          <DialogDescription>
            {translation?.title || post?.slug || t('Blog')}
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className='space-y-3'>
            <Skeleton className='h-20 w-full' />
            <Skeleton className='h-48 w-full' />
          </div>
        ) : error ? (
          <div className='rounded-lg border border-dashed px-4 py-10 text-center text-sm'>
            {t('Failed to load blog stats')}
          </div>
        ) : stats ? (
          <div className='space-y-5'>
            <div className='grid gap-3 sm:grid-cols-3'>
              <div className='rounded-lg border p-4'>
                <p className='text-muted-foreground text-sm'>
                  {t('All-time views')}
                </p>
                <p className='mt-2 text-2xl font-semibold tabular-nums'>
                  {formatNumber(stats.lifetime_views, locale)}
                </p>
              </div>
              <div className='rounded-lg border p-4'>
                <p className='text-muted-foreground text-sm'>
                  {t('Selected range')}
                </p>
                <p className='mt-2 text-2xl font-semibold tabular-nums'>
                  {formatNumber(stats.total_views, locale)}
                </p>
              </div>
              <div className='rounded-lg border p-4'>
                <p className='text-muted-foreground text-sm'>{t('Today')}</p>
                <p className='mt-2 text-2xl font-semibold tabular-nums'>
                  {formatNumber(stats.views_today, locale)}
                </p>
              </div>
            </div>

            <div className='space-y-2'>
              <div className='flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between'>
                <h4 className='text-sm font-semibold'>{t('Daily views')}</h4>
                <div className='grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]'>
                  <div className='space-y-1'>
                    <Label htmlFor='blog-stats-start' className='text-xs'>
                      {t('Start date')}
                    </Label>
                    <Input
                      id='blog-stats-start'
                      type='date'
                      value={rangeStart}
                      max={rangeEnd}
                      onChange={(event) =>
                        onRangeChange({
                          startDate: event.target.value,
                          endDate: rangeEnd,
                        })
                      }
                    />
                  </div>
                  <div className='space-y-1'>
                    <Label htmlFor='blog-stats-end' className='text-xs'>
                      {t('End date')}
                    </Label>
                    <Input
                      id='blog-stats-end'
                      type='date'
                      value={rangeEnd}
                      min={rangeStart}
                      max={dateToInputDate(new Date())}
                      onChange={(event) =>
                        onRangeChange({
                          startDate: rangeStart,
                          endDate: event.target.value,
                        })
                      }
                    />
                  </div>
                  <Button
                    type='button'
                    variant='outline'
                    className='self-end'
                    onClick={onResetRange}
                  >
                    {t('Reset')}
                  </Button>
                </div>
              </div>
              <div className='h-48 overflow-x-auto rounded-lg border p-3'>
                {hasChartRange ? (
                  <ChartContainer
                    config={chartConfig}
                    className='h-full min-w-[34rem] aspect-auto'
                    initialDimension={{ width: 544, height: 192 }}
                  >
                    <LineChart
                      accessibilityLayer
                      data={chartData}
                      margin={{ top: 8, right: 12, bottom: 0, left: 0 }}
                    >
                      <CartesianGrid vertical={false} />
                      <XAxis
                        dataKey='date'
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        minTickGap={24}
                        tickFormatter={(value) =>
                          formatBlogStatsDate(String(value), locale)
                        }
                      />
                      <YAxis
                        width={48}
                        tickLine={false}
                        axisLine={false}
                        tickMargin={8}
                        allowDecimals={false}
                        tickFormatter={(value) =>
                          formatNumber(Number(value), locale)
                        }
                      />
                      <ChartTooltip
                        cursor={{
                          stroke: 'var(--border)',
                          strokeDasharray: '4 4',
                        }}
                        content={
                          <ChartTooltipContent
                            labelFormatter={(value) =>
                              formatBlogStatsDate(String(value), locale)
                            }
                          />
                        }
                      />
                      <Line
                        dataKey='views'
                        type='monotone'
                        stroke='var(--color-views)'
                        strokeWidth={2}
                        dot={false}
                        activeDot={{ r: 4 }}
                      />
                    </LineChart>
                  </ChartContainer>
                ) : (
                  <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
                    {t('No views in this period')}
                  </div>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

type BlogEditorDialogProps = {
  open: boolean
  post?: BlogPostAdmin
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: BlogPostPayload) => void
}

function BlogEditorDialog({
  open,
  post,
  saving,
  onOpenChange,
  onSubmit,
}: BlogEditorDialogProps) {
  const { t, i18n } = useTranslation()
  const [draft, setDraft] = useState<BlogPostPayload>(() => clonePost(post))
  const [locale, setLocale] = useState<InterfaceLanguageCode>(() =>
    normalizeInterfaceLanguage(i18n.language) as InterfaceLanguageCode
  )

  useEffect(() => {
    if (!open) return
    setDraft(clonePost(post))
    setLocale(normalizeInterfaceLanguage(i18n.language) as InterfaceLanguageCode)
  }, [i18n.language, open, post])

  const activeTranslation =
    draft.translations[locale] ?? ({ ...emptyTranslation } as BlogPostTranslation)

  const updateDraft = <TKey extends keyof BlogPostPayload>(
    key: TKey,
    value: BlogPostPayload[TKey]
  ) => {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  const updateTranslation = (
    key: keyof BlogPostTranslation,
    value: string
  ) => {
    setDraft((current) => ({
      ...current,
      translations: {
        ...current.translations,
        [locale]: {
          ...(current.translations[locale] ?? emptyTranslation),
          [key]: value,
        },
      },
    }))
  }

  const canSave = useMemo(() => {
    const hasContent = Object.values(draft.translations).some(
      (item) => item && (item.title.trim() || item.content.trim())
    )
    return draft.slug.trim() !== '' && hasContent
  }, [draft.slug, draft.translations])

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!canSave) return
    onSubmit(draft)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[92svh] overflow-y-auto sm:max-w-4xl'>
        <DialogHeader>
          <DialogTitle>
            {post ? t('Edit blog post') : t('New blog post')}
          </DialogTitle>
          <DialogDescription>
            {t('Write localized Markdown content for the public blog.')}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className='space-y-5'>
          <div className='grid gap-4 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='blog-slug'>{t('Slug')}</Label>
              <Input
                id='blog-slug'
                value={draft.slug}
                onChange={(event) => updateDraft('slug', event.target.value)}
                placeholder='model-pricing-guide'
              />
              <p className='text-muted-foreground text-xs'>
                {t('Used in the public URL, for example /blog/model-pricing-guide.')}
              </p>
            </div>

            <div className='space-y-2'>
              <Label htmlFor='blog-status'>{t('Status')}</Label>
              <NativeSelect
                id='blog-status'
                className='w-full'
                value={draft.status}
                onChange={(event) =>
                  updateDraft('status', event.target.value as BlogPostStatus)
                }
              >
                <NativeSelectOption value='draft'>{t('Draft')}</NativeSelectOption>
                <NativeSelectOption value='published'>
                  {t('Published')}
                </NativeSelectOption>
              </NativeSelect>
            </div>
          </div>

          <div className='grid gap-4 md:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='blog-author'>{t('Author')}</Label>
              <Input
                id='blog-author'
                value={draft.author}
                onChange={(event) => updateDraft('author', event.target.value)}
                placeholder={t('Optional')}
              />
            </div>

            <div className='space-y-2'>
              <Label htmlFor='blog-published-time'>{t('Published at')}</Label>
              <Input
                id='blog-published-time'
                type='datetime-local'
                value={timestampToInput(draft.published_time)}
                onChange={(event) =>
                  updateDraft(
                    'published_time',
                    inputToTimestamp(event.target.value)
                  )
                }
              />
            </div>
          </div>

          <div className='space-y-2'>
            <Label htmlFor='blog-cover'>{t('Cover image URL')}</Label>
            <Input
              id='blog-cover'
              value={draft.cover_image}
              onChange={(event) => updateDraft('cover_image', event.target.value)}
              placeholder='https://example.com/image.png'
            />
          </div>

          <div className='space-y-2'>
            <Label>{t('Tags')}</Label>
            <TagInput
              value={draft.tags}
              onChange={(tags) => updateDraft('tags', tags)}
              placeholder={t('Add tags...')}
            />
          </div>

          <div className='rounded-lg border p-4'>
            <div className='mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
              <div>
                <h4 className='text-sm font-semibold'>
                  {t('Localized content')}
                </h4>
                <p className='text-muted-foreground text-xs'>
                  {t('Empty locales fall back to English, then Chinese.')}
                </p>
              </div>
              <NativeSelect
                value={locale}
                onChange={(event) =>
                  setLocale(event.target.value as InterfaceLanguageCode)
                }
              >
                {INTERFACE_LANGUAGE_OPTIONS.map((option) => (
                  <NativeSelectOption key={option.code} value={option.code}>
                    {option.label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>

            <div className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='blog-title'>{t('Title')}</Label>
                <Input
                  id='blog-title'
                  value={activeTranslation.title}
                  onChange={(event) =>
                    updateTranslation('title', event.target.value)
                  }
                />
              </div>

              <div className='space-y-2'>
                <Label htmlFor='blog-summary'>{t('Summary')}</Label>
                <Textarea
                  id='blog-summary'
                  rows={3}
                  value={activeTranslation.summary}
                  onChange={(event) =>
                    updateTranslation('summary', event.target.value)
                  }
                />
              </div>

              <div className='space-y-2'>
                <Label htmlFor='blog-content'>{t('Markdown content')}</Label>
                <Textarea
                  id='blog-content'
                  rows={16}
                  className='font-mono text-xs'
                  value={activeTranslation.content}
                  onChange={(event) =>
                    updateTranslation('content', event.target.value)
                  }
                />
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' disabled={!canSave || saving}>
              {saving ? t('Saving...') : t('Save blog post')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function BlogSection() {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const locale = normalizeInterfaceLanguage(i18n.language)
  const [page, setPage] = useState(1)
  const [keywordDraft, setKeywordDraft] = useState('')
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<BlogPostStatus | ''>('')
  const [editingPost, setEditingPost] = useState<BlogPostAdmin | undefined>()
  const [statsPost, setStatsPost] = useState<BlogPostAdmin | undefined>()
  const [statsRange, setStatsRange] = useState(() => getDefaultStatsRange())
  const [editorOpen, setEditorOpen] = useState(false)

  const postsQuery = useQuery({
    queryKey: ['admin-blog-posts', page, keyword, status],
    queryFn: () =>
      getAdminBlogPosts({
        page,
        pageSize: BLOG_PAGE_SIZE,
        keyword,
        status,
      }),
  })

  const statsQuery = useQuery({
    queryKey: [
      'admin-blog-post-stats',
      statsPost?.id,
      statsRange.startDate,
      statsRange.endDate,
    ],
    queryFn: () =>
      getAdminBlogPostStats(statsPost!.id, {
        startDate: statsRange.startDate,
        endDate: statsRange.endDate,
      }),
    enabled: Boolean(statsPost?.id && statsRange.startDate && statsRange.endDate),
  })

  const invalidatePosts = () =>
    queryClient.invalidateQueries({ queryKey: ['admin-blog-posts'] })

  const saveMutation = useMutation({
    mutationFn: (payload: BlogPostPayload) =>
      payload.id ? updateAdminBlogPost(payload) : createAdminBlogPost(payload),
    onSuccess: (data) => {
      if (!data.success) {
        toast.error(data.message || t('Failed to save blog post'))
        return
      }
      toast.success(t('Blog post saved'))
      setEditorOpen(false)
      setEditingPost(undefined)
      void invalidatePosts()
      void queryClient.invalidateQueries({ queryKey: ['blog-posts'] })
      void queryClient.invalidateQueries({ queryKey: ['blog-post'] })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to save blog post'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteAdminBlogPost(id),
    onSuccess: (data) => {
      if (!data.success) {
        toast.error(data.message || t('Failed to delete blog post'))
        return
      }
      toast.success(t('Blog post deleted'))
      void invalidatePosts()
      void queryClient.invalidateQueries({ queryKey: ['blog-posts'] })
      void queryClient.invalidateQueries({ queryKey: ['blog-post'] })
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to delete blog post'))
    },
  })

  const pageData = postsQuery.data?.data
  const posts = pageData?.items ?? []
  const totalPages = pageData
    ? Math.max(1, Math.ceil(pageData.total / pageData.page_size))
    : 1

  const openCreate = () => {
    setEditingPost(undefined)
    setEditorOpen(true)
  }

  const openEdit = (post: BlogPostAdmin) => {
    setEditingPost(post)
    setEditorOpen(true)
  }

  const openStats = (post: BlogPostAdmin) => {
    setStatsRange(getDefaultStatsRange(post))
    setStatsPost(post)
  }

  const handleDelete = (post: BlogPostAdmin) => {
    if (!window.confirm(t('Delete this blog post?'))) return
    deleteMutation.mutate(post.id)
  }

  const onSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPage(1)
    setKeyword(keywordDraft.trim())
  }

  return (
    <SettingsSection
      title={t('Blog')}
      description={t('Manage multilingual Markdown blog posts.')}
    >
      <div className='space-y-4'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <form onSubmit={onSearch} className='flex flex-col gap-2 sm:flex-row'>
            <div className='relative min-w-0 sm:w-80'>
              <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
              <Input
                value={keywordDraft}
                onChange={(event) => setKeywordDraft(event.target.value)}
                className='pl-8'
                placeholder={t('Search blog posts')}
              />
            </div>
            <NativeSelect
              className='w-full sm:w-36'
              value={status}
              onChange={(event) => {
                setPage(1)
                setStatus(event.target.value as BlogPostStatus | '')
              }}
            >
              <NativeSelectOption value=''>{t('All statuses')}</NativeSelectOption>
              <NativeSelectOption value='draft'>{t('Draft')}</NativeSelectOption>
              <NativeSelectOption value='published'>
                {t('Published')}
              </NativeSelectOption>
            </NativeSelect>
            <Button type='submit'>{t('Search')}</Button>
          </form>

          <Button onClick={openCreate}>
            <Plus className='size-4' />
            {t('New blog post')}
          </Button>
        </div>

        {postsQuery.isLoading ? (
          <div className='space-y-2 rounded-lg border p-4'>
            <Skeleton className='h-8 w-full' />
            <Skeleton className='h-8 w-5/6' />
            <Skeleton className='h-8 w-2/3' />
          </div>
        ) : posts.length > 0 ? (
          <div className='rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Title')}</TableHead>
                  <TableHead>{t('Slug')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Published at')}</TableHead>
                  <TableHead className='text-right'>{t('30-day views')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {posts.map((post) => {
                  const translation = getLocalizedBlogTranslation(post, locale)
                  return (
                    <TableRow key={post.id}>
                      <TableCell className='min-w-64 whitespace-normal'>
                        <div className='font-medium'>
                          {translation?.title || post.slug}
                        </div>
                        {translation?.summary ? (
                          <div className='text-muted-foreground line-clamp-1 text-xs'>
                            {translation.summary}
                          </div>
                        ) : null}
                      </TableCell>
                      <TableCell>{post.slug}</TableCell>
                      <TableCell>{statusLabel(post.status, t)}</TableCell>
                      <TableCell>
                        {formatDateTime(post.published_time, locale)}
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='font-medium tabular-nums'>
                          {formatNumber(post.stats?.total_views, locale)}
                        </div>
                        <div className='text-muted-foreground text-xs'>
                          {t('Today {{count}}', {
                            count: formatNumber(post.stats?.views_today, locale),
                          })}
                        </div>
                        <div className='text-muted-foreground text-xs'>
                          {t('All-time {{count}}', {
                            count: formatNumber(post.stats?.lifetime_views, locale),
                          })}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className='flex justify-end gap-2'>
                          <Button
                            size='icon-sm'
                            variant='outline'
                            aria-label={t('View blog stats')}
                            onClick={() => openStats(post)}
                          >
                            <BarChart3 className='size-4' />
                          </Button>
                          <Button
                            size='icon-sm'
                            variant='outline'
                            aria-label={t('Edit blog post')}
                            onClick={() => openEdit(post)}
                          >
                            <Edit className='size-4' />
                          </Button>
                          <Button
                            size='icon-sm'
                            variant='destructive'
                            aria-label={t('Delete blog post')}
                            onClick={() => handleDelete(post)}
                            disabled={deleteMutation.isPending}
                          >
                            <Trash2 className='size-4' />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className='rounded-lg border border-dashed px-4 py-10 text-center'>
            <p className='text-sm font-medium'>{t('No blog posts yet')}</p>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t('Create a draft and publish it when ready.')}
            </p>
          </div>
        )}

        {totalPages > 1 ? (
          <div className='flex flex-col items-center justify-between gap-3 sm:flex-row'>
            <p className='text-muted-foreground text-sm'>
              {t('Page {{page}} of {{total}}', { page, total: totalPages })}
            </p>
            <div className='flex gap-2'>
              <Button
                variant='outline'
                disabled={page <= 1}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                {t('Previous')}
              </Button>
              <Button
                variant='outline'
                disabled={page >= totalPages}
                onClick={() =>
                  setPage((value) => Math.min(totalPages, value + 1))
                }
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        ) : null}
      </div>

      <BlogEditorDialog
        open={editorOpen}
        post={editingPost}
        saving={saveMutation.isPending}
        onOpenChange={(open) => {
          setEditorOpen(open)
          if (!open) setEditingPost(undefined)
        }}
        onSubmit={(payload) => saveMutation.mutate(payload)}
      />
      <BlogStatsDialog
        open={Boolean(statsPost)}
        post={statsPost}
        stats={statsQuery.data?.success ? statsQuery.data.data : undefined}
        loading={statsQuery.isLoading}
        error={statsQuery.isError || statsQuery.data?.success === false}
        rangeStart={statsRange.startDate}
        rangeEnd={statsRange.endDate}
        locale={locale}
        onRangeChange={(range) =>
          setStatsRange((current) =>
            normalizeStatsRange({
              startDate: range.startDate || current.startDate,
              endDate: range.endDate || current.endDate,
            })
          )
        }
        onResetRange={() => setStatsRange(getDefaultStatsRange(statsPost))}
        onOpenChange={(open) => {
          if (!open) setStatsPost(undefined)
        }}
      />
    </SettingsSection>
  )
}
