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
import { useMemo, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { FileWarning } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Markdown } from '@/components/ui/markdown'
import { Skeleton } from '@/components/ui/skeleton'
import { PublicLayout } from '@/components/layout'
import { getDocumentationConfig, getDocumentationPage } from './api'

type PublicDocumentationPageProps = {
  path: string
  fallbackTitle: string
  fallback?: ReactNode
}

export function PublicDocumentationPage({
  path,
  fallbackTitle,
  fallback,
}: PublicDocumentationPageProps) {
  const { t } = useTranslation()
  const configQuery = useQuery({
    queryKey: ['documentation-config'],
    queryFn: getDocumentationConfig,
    staleTime: 5 * 60 * 1000,
  })

  const pageConfig = useMemo(() => {
    const normalizedPath = path.startsWith('/') ? path : `/${path}`
    return configQuery.data?.data?.pages?.find(
      (page) => page.path === normalizedPath
    )
  }, [configQuery.data?.data?.pages, path])

  const pageQuery = useQuery({
    queryKey: ['documentation-page', pageConfig?.slug],
    queryFn: () => getDocumentationPage(pageConfig?.slug ?? ''),
    enabled: Boolean(configQuery.data?.data?.enabled && pageConfig?.slug),
    staleTime: 5 * 60 * 1000,
  })

  if (configQuery.isLoading || pageQuery.isLoading) {
    return (
      <PublicLayout>
        <div className='mx-auto flex max-w-4xl flex-col gap-4 py-12'>
          <Skeleton className='h-8 w-[45%]' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-[90%]' />
          <Skeleton className='h-4 w-[80%]' />
        </div>
      </PublicLayout>
    )
  }

  const page = pageQuery.data?.success ? pageQuery.data.data : undefined

  if (!configQuery.data?.data?.enabled || !pageConfig || !page) {
    if (fallback) return fallback

    return (
      <PublicLayout>
        <div className='mx-auto max-w-2xl py-12'>
          <Card className='border-dashed'>
            <CardHeader className='flex flex-row items-center gap-4'>
              <div className='bg-muted rounded-lg p-2'>
                <FileWarning className='text-muted-foreground h-5 w-5' />
              </div>
              <div className='space-y-1'>
                <CardTitle className='text-lg font-semibold'>
                  {fallbackTitle}
                </CardTitle>
                <p className='text-muted-foreground text-sm'>
                  {pageQuery.data?.message ||
                    t('The requested public page does not exist.')}
                </p>
              </div>
            </CardHeader>
          </Card>
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout>
      <article className='mx-auto max-w-4xl space-y-6 py-12'>
        <header className='space-y-2 border-b pb-5'>
          <h1 className='text-3xl font-semibold tracking-tight'>
            {page.title || pageConfig.title || fallbackTitle}
          </h1>
          {pageConfig.description ? (
            <p className='text-muted-foreground'>{pageConfig.description}</p>
          ) : null}
        </header>
        <Markdown className='prose-neutral dark:prose-invert max-w-none'>
          {page.content}
        </Markdown>
      </article>
    </PublicLayout>
  )
}
