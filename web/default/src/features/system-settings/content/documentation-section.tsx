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
import { useCallback, useEffect, useMemo, useRef, type FormEvent } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Textarea } from '@/components/ui/textarea'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { formatJsonForEditor, normalizeJsonString } from './utils'

const DOCUMENTATION_OPTION_KEY = 'DocumentationSettings'
const DOCUMENTATION_FALLBACK = `{
  "enabled": false,
  "content_dir": "data/docs",
  "default_locale": "en",
  "default_slug": "",
  "nav": [],
  "pages": [],
  "debug_examples": {}
}`

type DocumentationSectionProps = {
  defaultValue: string
}

type DocumentationFormValues = {
  json: string
}

const supportedLocales = new Set(['en', 'zh', 'es', 'fr', 'ru', 'ja', 'vi'])
const supportedLocaleLabel = 'en, zh, es, fr, ru, ja, vi'
const slugPattern = /^[A-Za-z0-9._~/-]+$/

function validateLocalizedText(value: unknown, path: string, required = false) {
  if (value === undefined || value === null) {
    if (required) throw new Error(`${path} is required`)
    return
  }
  if (typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${path} must be an object keyed by locale`)
  }

  for (const [locale, text] of Object.entries(value)) {
    if (!supportedLocales.has(locale)) {
      throw new Error(`${path}.${locale} uses an unsupported locale`)
    }
    if (typeof text !== 'string') {
      throw new Error(`${path}.${locale} must be a string`)
    }
  }
}

function validateSlug(value: unknown, path: string, required = false) {
  if (value === undefined || value === null || value === '') {
    if (required) throw new Error(`${path} is required`)
    return
  }
  if (typeof value !== 'string') {
    throw new Error(`${path} must be a string`)
  }
  const slug = value.replace(/^\/+/, '').trim()
  if (
    !slugPattern.test(slug) ||
    slug === '.' ||
    slug === '..' ||
    slug.startsWith('../') ||
    slug.includes('/../') ||
    slug.includes('\\')
  ) {
    throw new Error(`${path} is invalid`)
  }
}

function validatePagePath(value: unknown, path: string) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`${path} must be a non-empty string`)
  }
  const pagePath = value.startsWith('/') ? value : `/${value}`
  if (
    pagePath.includes('\\') ||
    pagePath.includes('\0') ||
    pagePath === '/' ||
    pagePath.startsWith('/api/')
  ) {
    throw new Error(`${path} is invalid`)
  }
}

function validateNavItems(value: unknown, path: string) {
  if (!Array.isArray(value)) {
    throw new Error(`${path} must be an array`)
  }

  value.forEach((item, index) => {
    const itemPath = `${path}.${index}`
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      throw new Error(`${itemPath} must be an object`)
    }

    const navItem = item as Record<string, unknown>
    validateSlug(navItem.slug, `${itemPath}.slug`)
    validateSlug(navItem.file, `${itemPath}.file`)
    validateLocalizedText(navItem.title, `${itemPath}.title`)
    validateLocalizedText(navItem.description, `${itemPath}.description`)
    if (!navItem.slug && !navItem.title) {
      throw new Error(`${itemPath} requires either slug or title`)
    }
    if (navItem.children !== undefined) {
      validateNavItems(navItem.children, `${itemPath}.children`)
    }
  })
}

function validatePages(value: unknown, path: string) {
  if (value === undefined) return
  if (!Array.isArray(value)) {
    throw new Error(`${path} must be an array`)
  }

  const seen = new Set<string>()
  value.forEach((item, index) => {
    const itemPath = `${path}.${index}`
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      throw new Error(`${itemPath} must be an object`)
    }
    const page = item as Record<string, unknown>
    validatePagePath(page.path, `${itemPath}.path`)
    validateSlug(page.slug, `${itemPath}.slug`, true)
    validateSlug(page.file, `${itemPath}.file`)
    validateLocalizedText(page.title, `${itemPath}.title`)
    validateLocalizedText(page.description, `${itemPath}.description`)
    const normalizedPath = String(page.path).startsWith('/')
      ? String(page.path)
      : `/${String(page.path)}`
    if (seen.has(normalizedPath)) {
      throw new Error(`${itemPath}.path is duplicated`)
    }
    seen.add(normalizedPath)
  })
}

function validateDebugExamples(value: unknown, path: string) {
  if (value === undefined) return
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${path} must be a JSON object`)
  }

  for (const [slug, rawExample] of Object.entries(value)) {
    validateSlug(slug, `${path}.${slug}`, true)
    if (
      !rawExample ||
      typeof rawExample !== 'object' ||
      Array.isArray(rawExample)
    ) {
      throw new Error(`${path}.${slug} must be an object`)
    }

    const example = rawExample as Record<string, unknown>
    if (
      typeof example.method !== 'string' ||
      !['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].includes(
        example.method.toUpperCase()
      )
    ) {
      throw new Error(`${path}.${slug}.method is invalid`)
    }
    if (
      typeof example.path !== 'string' &&
      typeof example.path_template !== 'string'
    ) {
      throw new Error(`${path}.${slug} requires path or path_template`)
    }
    if (example.body !== undefined) {
      if (
        !example.body ||
        typeof example.body !== 'object' ||
        Array.isArray(example.body)
      ) {
        throw new Error(
          `${path}.${slug}.body must be an object keyed by locale`
        )
      }
      for (const locale of Object.keys(example.body)) {
        if (!supportedLocales.has(locale)) {
          throw new Error(
            `${path}.${slug}.body.${locale} uses an unsupported locale`
          )
        }
      }
    }
  }
}

function validateDocumentationConfig(value: string) {
  const normalized = normalizeJsonString(value, DOCUMENTATION_FALLBACK)
  const parsed = JSON.parse(normalized) as unknown

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Documentation config must be a JSON object')
  }

  const config = parsed as Record<string, unknown>
  if (typeof config.enabled !== 'boolean') {
    throw new Error('enabled must be a boolean')
  }
  if (typeof config.content_dir !== 'string' || config.content_dir === '') {
    throw new Error('content_dir must be a non-empty string')
  }
  if (
    typeof config.default_locale !== 'string' ||
    !supportedLocales.has(config.default_locale)
  ) {
    throw new Error(`default_locale must be one of ${supportedLocaleLabel}`)
  }
  validateSlug(config.default_slug, 'default_slug')
  validateNavItems(config.nav, 'nav')
  validatePages(config.pages, 'pages')
  validateDebugExamples(config.debug_examples, 'debug_examples')
}

function createSchema(t: (key: string) => string) {
  return z.object({
    json: z.string().superRefine((value, ctx) => {
      try {
        validateDocumentationConfig(value)
      } catch (error) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message:
            error instanceof Error
              ? error.message
              : t('Documentation JSON is invalid'),
        })
      }
    }),
  })
}

export function DocumentationSection({
  defaultValue,
}: DocumentationSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialNormalizedRef = useRef(
    normalizeJsonString(defaultValue, DOCUMENTATION_FALLBACK)
  )

  const formattedDefault = useMemo(
    () => formatJsonForEditor(defaultValue, DOCUMENTATION_FALLBACK),
    [defaultValue]
  )

  const form = useForm<DocumentationFormValues>({
    mode: 'onChange',
    resolver: zodResolver(createSchema(t)),
    defaultValues: {
      json: formattedDefault,
    },
  })

  useEffect(() => {
    initialNormalizedRef.current = normalizeJsonString(
      defaultValue,
      DOCUMENTATION_FALLBACK
    )
    form.reset({
      json: formatJsonForEditor(defaultValue, DOCUMENTATION_FALLBACK),
    })
  }, [defaultValue, form])

  const onSubmit = useCallback(
    async (values: DocumentationFormValues) => {
      const normalized = normalizeJsonString(
        values.json,
        DOCUMENTATION_FALLBACK
      )
      if (normalized === initialNormalizedRef.current) return

      await updateOption.mutateAsync({
        key: DOCUMENTATION_OPTION_KEY,
        value: normalized,
      })
    },
    [updateOption]
  )

  const handleFormSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      void form.handleSubmit(onSubmit)(event)
    },
    [form, onSubmit]
  )

  return (
    <SettingsSection
      title={t('Documentation')}
      description={t('Configure Markdown documentation pages and navigation')}
    >
      <Form {...form}>
        <form onSubmit={handleFormSubmit} className='space-y-6'>
          <FormField
            control={form.control}
            name='json'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Documentation JSON')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={22}
                    spellCheck={false}
                    className='font-mono text-xs'
                    placeholder={DOCUMENTATION_FALLBACK}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Markdown files are resolved from content_dir by locale and slug.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button type='submit' disabled={updateOption.isPending}>
            {updateOption.isPending
              ? t('Saving...')
              : t('Save documentation settings')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
