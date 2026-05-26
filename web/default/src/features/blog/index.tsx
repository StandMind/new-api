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
import { useMemo, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowRight, CalendarDays, Search, SearchX } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { normalizeInterfaceLanguage } from '@/i18n/languages'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { PublicLayout } from '@/components/layout'
import { getBlogPosts } from './api'
import type { BlogPost } from './types'

function formatBlogDate(timestamp: number, locale: string) {
  if (!timestamp) return ''
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(timestamp * 1000))
}

function BlogCard({ post, locale }: { post: BlogPost; locale: string }) {
  const { t } = useTranslation()
  const publishedAt = formatBlogDate(post.published_time, locale)

  return (
    <article className='group border-border bg-card text-card-foreground overflow-hidden rounded-lg border transition-colors hover:border-primary/40'>
      {post.cover_image ? (
        <Link
          to='/blog/$slug'
          params={{ slug: post.slug }}
          className='bg-muted block aspect-[16/9] overflow-hidden'
        >
          <img
            src={post.cover_image}
            alt=''
            loading='lazy'
            className='h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]'
          />
        </Link>
      ) : null}
      <div className='space-y-4 p-4 sm:p-5'>
        <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
          {publishedAt ? (
            <span className='inline-flex items-center gap-1'>
              <CalendarDays className='size-3.5' />
              {publishedAt}
            </span>
          ) : null}
          {post.reading_minutes > 0 ? (
            <span>
              {t('{{count}} min read', { count: post.reading_minutes })}
            </span>
          ) : null}
        </div>

        <div className='space-y-2'>
          <h2 className='line-clamp-2 text-lg font-semibold tracking-tight sm:text-xl'>
            <Link
              to='/blog/$slug'
              params={{ slug: post.slug }}
              className='hover:text-primary transition-colors'
            >
              {post.title || post.slug}
            </Link>
          </h2>
          {post.summary ? (
            <p className='text-muted-foreground line-clamp-3 text-sm leading-6'>
              {post.summary}
            </p>
          ) : null}
        </div>

        {post.tags.length > 0 ? (
          <div className='flex flex-wrap gap-2'>
            {post.tags.slice(0, 4).map((tag) => (
              <Badge key={tag} variant='outline'>
                {tag}
              </Badge>
            ))}
          </div>
        ) : null}

        <Button
          variant='ghost'
          size='sm'
          className='px-0 hover:bg-transparent'
          render={<Link to='/blog/$slug' params={{ slug: post.slug }} />}
        >
          {t('Read article')}
          <ArrowRight className='size-4' />
        </Button>
      </div>
    </article>
  )
}

function BlogListSkeleton() {
  return (
    <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
      {Array.from({ length: 6 }).map((_, index) => (
        <div key={index} className='rounded-lg border p-5'>
          <Skeleton className='mb-4 h-4 w-32' />
          <Skeleton className='mb-3 h-6 w-4/5' />
          <Skeleton className='mb-2 h-4 w-full' />
          <Skeleton className='mb-2 h-4 w-5/6' />
          <Skeleton className='h-4 w-2/3' />
        </div>
      ))}
    </div>
  )
}

function EmptyBlogState({ searching }: { searching: boolean }) {
  const { t } = useTranslation()
  return (
    <div className='flex min-h-[40vh] items-center justify-center rounded-lg border border-dashed px-4 py-12'>
      <div className='max-w-md space-y-3 text-center'>
        <div className='bg-muted mx-auto flex size-12 items-center justify-center rounded-lg'>
          <SearchX className='text-muted-foreground size-5' />
        </div>
        <h2 className='text-xl font-semibold'>
          {searching ? t('No matching articles') : t('No blog posts yet')}
        </h2>
        <p className='text-muted-foreground text-sm leading-6'>
          {searching
            ? t('Try another keyword or clear the search.')
            : t('Published articles will appear here.')}
        </p>
      </div>
    </div>
  )
}

export function Blog() {
  const { t, i18n } = useTranslation()
  const locale = normalizeInterfaceLanguage(i18n.language)
  const [page, setPage] = useState(1)
  const [searchDraft, setSearchDraft] = useState('')
  const [keyword, setKeyword] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['blog-posts', locale, page, keyword],
    queryFn: () =>
      getBlogPosts({
        lang: locale,
        page,
        pageSize: 12,
        keyword,
      }),
  })

  const pageData = data?.data
  const posts = pageData?.items ?? []
  const totalPages = useMemo(() => {
    if (!pageData || pageData.page_size <= 0) return 1
    return Math.max(1, Math.ceil(pageData.total / pageData.page_size))
  }, [pageData])

  const onSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPage(1)
    setKeyword(searchDraft.trim())
  }

  return (
    <PublicLayout>
      <div className='mx-auto max-w-7xl space-y-8 px-0 py-4 sm:px-2 lg:px-4'>
        <header className='flex flex-col gap-5 border-b pb-6 lg:flex-row lg:items-end lg:justify-between'>
          <div className='max-w-2xl space-y-2'>
            <h1 className='text-3xl font-semibold tracking-tight sm:text-4xl'>
              {t('Blog')}
            </h1>
            <p className='text-muted-foreground text-sm leading-6 sm:text-base'>
              {t('AI updates, model guides, and practical API notes.')}
            </p>
          </div>

          <form
            onSubmit={onSearch}
            className='flex w-full flex-col gap-2 sm:w-auto sm:min-w-80 sm:flex-row'
          >
            <div className='relative flex-1'>
              <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
              <Input
                value={searchDraft}
                onChange={(event) => setSearchDraft(event.target.value)}
                className='pl-8'
                placeholder={t('Search articles')}
              />
            </div>
            <Button type='submit'>{t('Search')}</Button>
          </form>
        </header>

        {isLoading ? (
          <BlogListSkeleton />
        ) : posts.length > 0 ? (
          <>
            <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
              {posts.map((post) => (
                <BlogCard key={post.id} post={post} locale={locale} />
              ))}
            </div>

            {totalPages > 1 ? (
              <div className='flex flex-col items-center justify-between gap-3 border-t pt-4 sm:flex-row'>
                <p className='text-muted-foreground text-sm'>
                  {t('Page {{page}} of {{total}}', {
                    page,
                    total: totalPages,
                  })}
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
          </>
        ) : (
          <EmptyBlogState searching={keyword !== ''} />
        )}
      </div>
    </PublicLayout>
  )
}
