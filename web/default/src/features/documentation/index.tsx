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
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { FileText, SearchX } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { PublicLayout } from '@/components/layout'
import { cn } from '@/lib/utils'
import { getDocumentationConfig, getDocumentationPage } from './api'
import { ApiDebugPanel } from './api-debug-panel'
import type {
  DocumentationConfig,
  DocumentationNavItem,
  DocumentationPageData,
} from './types'

type DocumentationProps = {
  slug?: string
}

type EmptyStateProps = {
  title: string
  message: string
}

function normalizeSlug(value?: string) {
  return (value ?? '').replace(/^\/+|\/+$/g, '')
}

function EmptyState({ title, message }: EmptyStateProps) {
  return (
    <div className='flex min-h-[45vh] items-center justify-center px-4 py-12'>
      <div className='max-w-lg space-y-4 text-center'>
        <div className='bg-muted mx-auto flex size-12 items-center justify-center rounded-lg'>
          <SearchX className='text-muted-foreground size-5' />
        </div>
        <div className='space-y-2'>
          <h1 className='text-2xl font-semibold tracking-tight'>{title}</h1>
          <p className='text-muted-foreground text-sm leading-relaxed'>
            {message}
          </p>
        </div>
      </div>
    </div>
  )
}

function LoadingDocumentation() {
  return (
    <PublicLayout>
      <div className='mx-auto grid max-w-7xl gap-8 py-8 lg:grid-cols-[16rem_minmax(0,1fr)]'>
        <aside className='space-y-3'>
          <Skeleton className='h-6 w-28' />
          <Skeleton className='h-9 w-full' />
          <Skeleton className='h-9 w-[85%]' />
          <Skeleton className='h-9 w-[75%]' />
        </aside>
        <main className='space-y-4'>
          <Skeleton className='h-9 w-[45%]' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-[90%]' />
          <Skeleton className='h-4 w-[70%]' />
        </main>
      </div>
    </PublicLayout>
  )
}

function NavTree({
  items,
  activeSlug,
  depth = 0,
}: {
  items: DocumentationNavItem[]
  activeSlug: string
  depth?: number
}) {
  if (items.length === 0) return null

  return (
    <ul className={cn(depth > 0 && 'mt-1 space-y-1 border-l pl-3')}>
      {items.map((item, index) => {
        const slug = normalizeSlug(item.slug)
        const isActive = slug !== '' && slug === activeSlug
        const title = item.title || slug
        const children = item.children ?? []

        return (
          <li key={`${slug || title}-${index}`} className='space-y-1'>
            {slug ? (
              <Link
                to='/docs/$'
                params={{ _splat: slug }}
                className={cn(
                  'block rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                )}
              >
                <span className='block truncate'>{title}</span>
                {item.description ? (
                  <span
                    className={cn(
                      'mt-0.5 block truncate text-xs font-normal',
                      isActive
                        ? 'text-primary-foreground/80'
                        : 'text-muted-foreground'
                    )}
                  >
                    {item.description}
                  </span>
                ) : null}
              </Link>
            ) : (
              <div className='text-foreground px-3 py-2 text-sm font-semibold'>
                {title}
              </div>
            )}

            <NavTree
              items={children}
              activeSlug={activeSlug}
              depth={depth + 1}
            />
          </li>
        )
      })}
    </ul>
  )
}

function DocumentationShell({
  config,
  page,
  activeSlug,
  children,
}: {
  config?: DocumentationConfig
  page?: DocumentationPageData
  activeSlug: string
  children: ReactNode
}) {
  const { t } = useTranslation()
  const navItems = config?.nav ?? []

  return (
    <PublicLayout>
      <div className='mx-auto grid max-w-7xl gap-8 py-8 lg:grid-cols-[16rem_minmax(0,1fr)]'>
        <aside className='lg:sticky lg:top-20 lg:max-h-[calc(100svh-6rem)] lg:overflow-y-auto'>
          <div className='mb-4 flex items-center gap-2'>
            <FileText className='text-muted-foreground size-5' />
            <h1 className='text-lg font-semibold'>{t('Docs')}</h1>
          </div>

          {navItems.length > 0 ? (
            <nav aria-label={t('Documentation navigation')}>
              <NavTree items={navItems} activeSlug={activeSlug} />
            </nav>
          ) : (
            <p className='text-muted-foreground text-sm'>
              {t('No documentation navigation configured.')}
            </p>
          )}
        </aside>

        <main className='min-w-0'>
          {page ? (
            <article className='space-y-6'>
              <header className='border-b pb-5'>
                <h2 className='text-3xl font-semibold tracking-tight'>
                  {page.title}
                </h2>
                <p className='text-muted-foreground mt-2 text-sm'>
                  {t('Language')}: {page.locale}
                </p>
              </header>
              <Markdown className='prose-neutral dark:prose-invert'>
                {page.content}
              </Markdown>
              <ApiDebugPanel debug={page.debug} />
            </article>
          ) : (
            children
          )}
        </main>
      </div>
    </PublicLayout>
  )
}

export function Documentation({ slug }: DocumentationProps) {
  const { t } = useTranslation()
  const configQuery = useQuery({
    queryKey: ['documentation-config'],
    queryFn: getDocumentationConfig,
    staleTime: 5 * 60 * 1000,
  })

  const config = configQuery.data?.data
  const requestedSlug = normalizeSlug(slug)
  const activeSlug = requestedSlug || normalizeSlug(config?.default_slug)

  const pageQuery = useQuery({
    queryKey: ['documentation-page', activeSlug],
    queryFn: () => getDocumentationPage(activeSlug),
    enabled: Boolean(config?.enabled && activeSlug),
    staleTime: 5 * 60 * 1000,
  })

  if (configQuery.isLoading) {
    return <LoadingDocumentation />
  }

  if (!configQuery.data?.success || !config) {
    return (
      <PublicLayout>
        <EmptyState
          title={t('Documentation unavailable')}
          message={
            configQuery.data?.message ||
            t('Documentation settings could not be loaded.')
          }
        />
      </PublicLayout>
    )
  }

  if (!config.enabled) {
    return (
      <PublicLayout>
        <EmptyState
          title={t('Documentation is disabled')}
          message={t('The administrator has not enabled documentation pages.')}
        />
      </PublicLayout>
    )
  }

  if (!activeSlug) {
    return (
      <DocumentationShell config={config} activeSlug=''>
        <EmptyState
          title={t('No documentation pages configured')}
          message={t('Add a default slug or navigation item in system settings.')}
        />
      </DocumentationShell>
    )
  }

  if (pageQuery.isLoading) {
    return <LoadingDocumentation />
  }

  const page = pageQuery.data?.success ? pageQuery.data.data : undefined

  return (
    <DocumentationShell config={config} page={page} activeSlug={activeSlug}>
      <EmptyState
        title={t('Documentation page not found')}
        message={
          pageQuery.data?.message ||
          t('The requested documentation page does not exist.')
        }
      />
    </DocumentationShell>
  )
}
