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
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, CalendarDays, SearchX } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { normalizeInterfaceLanguage } from '@/i18n/languages'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Markdown,
  extractMarkdownHeadings,
  type MarkdownHeading,
} from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { PublicLayout } from '@/components/layout'
import { getBlogPost } from './api'

type BlogPostPageProps = {
  slug: string
}

function formatBlogDate(timestamp: number, locale: string) {
  if (!timestamp) return ''
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(new Date(timestamp * 1000))
}

function BlogPostLoading() {
  return (
    <PublicLayout>
      <div className='mx-auto max-w-6xl px-0 py-4 sm:px-2 lg:px-4'>
        <Skeleton className='mb-6 h-8 w-28' />
        <div className='grid gap-8 lg:grid-cols-[minmax(0,1fr)_14rem]'>
          <main className='space-y-5'>
            <Skeleton className='h-10 w-4/5' />
            <Skeleton className='h-5 w-2/3' />
            <Skeleton className='h-64 w-full rounded-lg' />
            <Skeleton className='h-4 w-full' />
            <Skeleton className='h-4 w-11/12' />
            <Skeleton className='h-4 w-10/12' />
          </main>
          <aside className='hidden space-y-3 lg:block'>
            <Skeleton className='h-4 w-24' />
            <Skeleton className='h-4 w-full' />
            <Skeleton className='h-4 w-2/3' />
          </aside>
        </div>
      </div>
    </PublicLayout>
  )
}

function EmptyPost() {
  const { t } = useTranslation()
  return (
    <PublicLayout>
      <div className='flex min-h-[50vh] items-center justify-center px-4 py-12'>
        <div className='max-w-md space-y-4 text-center'>
          <div className='bg-muted mx-auto flex size-12 items-center justify-center rounded-lg'>
            <SearchX className='text-muted-foreground size-5' />
          </div>
          <div className='space-y-2'>
            <h1 className='text-2xl font-semibold'>
              {t('Article not found')}
            </h1>
            <p className='text-muted-foreground text-sm leading-6'>
              {t('The article may be unpublished or the link may be invalid.')}
            </p>
          </div>
          <Button variant='outline' render={<Link to='/blog' />}>
            <ArrowLeft className='size-4' />
            {t('Back to blog')}
          </Button>
        </div>
      </div>
    </PublicLayout>
  )
}

function TableOfContents({ headings }: { headings: MarkdownHeading[] }) {
  const { t } = useTranslation()
  if (headings.length === 0) return null

  return (
    <aside className='hidden lg:block'>
      <div className='sticky top-20 max-h-[calc(100svh-6rem)] overflow-y-auto'>
        <p className='mb-3 text-sm font-semibold'>{t('On this page')}</p>
        <nav aria-label={t('On this page')}>
          <ol className='space-y-2 border-l pl-4'>
            {headings.map((heading) => (
              <li
                key={heading.id}
                className={cn(
                  'leading-tight',
                  heading.depth === 3 && 'pl-3',
                  heading.depth >= 4 && 'pl-6'
                )}
              >
                <a
                  href={`#${heading.id}`}
                  className='text-muted-foreground hover:text-foreground block text-xs transition-colors'
                >
                  {heading.title}
                </a>
              </li>
            ))}
          </ol>
        </nav>
      </div>
    </aside>
  )
}

export function BlogPostPage({ slug }: BlogPostPageProps) {
  const { t, i18n } = useTranslation()
  const locale = normalizeInterfaceLanguage(i18n.language)
  const { data, isLoading } = useQuery({
    queryKey: ['blog-post', slug, locale],
    queryFn: () => getBlogPost(slug, locale),
  })

  const post = data?.success ? data.data : undefined
  const headings = useMemo(
    () => extractMarkdownHeadings(post?.content ?? ''),
    [post?.content]
  )

  if (isLoading) return <BlogPostLoading />
  if (!post) return <EmptyPost />

  const publishedAt = formatBlogDate(post.published_time, locale)

  return (
    <PublicLayout>
      <div className='mx-auto max-w-6xl space-y-6 px-0 py-4 sm:px-2 lg:px-4'>
        <Button variant='ghost' className='px-0' render={<Link to='/blog' />}>
          <ArrowLeft className='size-4' />
          {t('Back to blog')}
        </Button>

        <div className='grid gap-8 lg:grid-cols-[minmax(0,1fr)_14rem]'>
          <article className='min-w-0 space-y-8'>
            <header className='space-y-5 border-b pb-6'>
              <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-2 text-sm'>
                {publishedAt ? (
                  <span className='inline-flex items-center gap-1'>
                    <CalendarDays className='size-4' />
                    {publishedAt}
                  </span>
                ) : null}
                {post.author ? <span>{post.author}</span> : null}
                {post.reading_minutes > 0 ? (
                  <span>
                    {t('{{count}} min read', {
                      count: post.reading_minutes,
                    })}
                  </span>
                ) : null}
              </div>

              <div className='space-y-3'>
                <h1 className='text-3xl font-semibold tracking-tight sm:text-4xl lg:text-5xl'>
                  {post.title || post.slug}
                </h1>
                {post.summary ? (
                  <p className='text-muted-foreground max-w-3xl text-base leading-7 sm:text-lg'>
                    {post.summary}
                  </p>
                ) : null}
              </div>

              {post.tags.length > 0 ? (
                <div className='flex flex-wrap gap-2'>
                  {post.tags.map((tag) => (
                    <Badge key={tag} variant='outline'>
                      {tag}
                    </Badge>
                  ))}
                </div>
              ) : null}
            </header>

            {post.cover_image ? (
              <div className='bg-muted overflow-hidden rounded-lg border'>
                <img
                  src={post.cover_image}
                  alt=''
                  className='h-auto w-full object-cover'
                />
              </div>
            ) : null}

            <Markdown className='prose-neutral dark:prose-invert'>
              {post.content ?? ''}
            </Markdown>
          </article>

          <TableOfContents headings={headings} />
        </div>
      </div>
    </PublicLayout>
  )
}
