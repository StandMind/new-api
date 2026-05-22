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

const EMAIL_TEMPLATES_OPTION_KEY = 'EmailTemplates'
const EMAIL_TEMPLATES_FALLBACK = `{
  "default_locale": "en",
  "support_email": "",
  "templates": {}
}`

type EmailTemplatesSectionProps = {
  defaultValue: string
}

type EmailTemplatesFormValues = {
  json: string
}

const supportedLocales = new Set(['en', 'zh', 'es', 'fr', 'ru', 'ja', 'vi'])
const supportedLocaleLabel = 'en, zh, es, fr, ru, ja, vi'

function validateEmailTemplateConfig(value: string) {
  const normalized = normalizeJsonString(value, EMAIL_TEMPLATES_FALLBACK)
  const parsed = JSON.parse(normalized) as unknown

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Email template config must be a JSON object')
  }

  const config = parsed as Record<string, unknown>
  const defaultLocale = config.default_locale
  if (
    typeof defaultLocale !== 'string' ||
    !supportedLocales.has(defaultLocale)
  ) {
    throw new Error(`default_locale must be one of ${supportedLocaleLabel}`)
  }

  if (
    config.support_email !== undefined &&
    typeof config.support_email !== 'string'
  ) {
    throw new Error('support_email must be a string')
  }

  if (
    !config.templates ||
    typeof config.templates !== 'object' ||
    Array.isArray(config.templates)
  ) {
    throw new Error('templates must be a JSON object')
  }

  for (const [templateName, locales] of Object.entries(
    config.templates as Record<string, unknown>
  )) {
    if (!locales || typeof locales !== 'object' || Array.isArray(locales)) {
      throw new Error(`${templateName} must be an object keyed by locale`)
    }

    for (const [locale, template] of Object.entries(
      locales as Record<string, unknown>
    )) {
      if (!supportedLocales.has(locale)) {
        throw new Error(`${templateName}.${locale} uses an unsupported locale`)
      }
      if (
        !template ||
        typeof template !== 'object' ||
        Array.isArray(template)
      ) {
        throw new Error(`${templateName}.${locale} must be a JSON object`)
      }

      const templateObject = template as Record<string, unknown>
      if (
        typeof templateObject.subject !== 'string' ||
        typeof templateObject.html !== 'string'
      ) {
        throw new Error(`${templateName}.${locale} requires subject and html`)
      }
    }
  }
}

function createSchema(t: (key: string) => string) {
  return z.object({
    json: z.string().superRefine((value, ctx) => {
      try {
        validateEmailTemplateConfig(value)
      } catch (error) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message:
            error instanceof Error
              ? error.message
              : t('Email template JSON is invalid'),
        })
      }
    }),
  })
}

export function EmailTemplatesSection({
  defaultValue,
}: EmailTemplatesSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialNormalizedRef = useRef(
    normalizeJsonString(defaultValue, EMAIL_TEMPLATES_FALLBACK)
  )

  const formattedDefault = useMemo(
    () => formatJsonForEditor(defaultValue, EMAIL_TEMPLATES_FALLBACK),
    [defaultValue]
  )

  const form = useForm<EmailTemplatesFormValues>({
    mode: 'onChange',
    resolver: zodResolver(createSchema(t)),
    defaultValues: {
      json: formattedDefault,
    },
  })

  useEffect(() => {
    initialNormalizedRef.current = normalizeJsonString(
      defaultValue,
      EMAIL_TEMPLATES_FALLBACK
    )
    form.reset({
      json: formatJsonForEditor(defaultValue, EMAIL_TEMPLATES_FALLBACK),
    })
  }, [defaultValue, form])

  const onSubmit = useCallback(
    async (values: EmailTemplatesFormValues) => {
      const normalized = normalizeJsonString(
        values.json,
        EMAIL_TEMPLATES_FALLBACK
      )
      if (normalized === initialNormalizedRef.current) return

      await updateOption.mutateAsync({
        key: EMAIL_TEMPLATES_OPTION_KEY,
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
      title={t('Email Templates')}
      description={t(
        'Configure localized verification and password reset email templates'
      )}
    >
      <Form {...form}>
        <form onSubmit={handleFormSubmit} className='space-y-6'>
          <FormField
            control={form.control}
            name='json'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Template JSON')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={24}
                    spellCheck={false}
                    className='font-mono text-xs'
                    placeholder={EMAIL_TEMPLATES_FALLBACK}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t('Available placeholders:')}{' '}
                  {[
                    '{{.SystemName}}',
                    '{{.Email}}',
                    '{{.Code}}',
                    '{{.ResetLink}}',
                    '{{.SupportEmail}}',
                    '{{.ValidMinutes}}',
                  ].map((placeholder, index, placeholders) => (
                    <span key={placeholder}>
                      <code>{placeholder}</code>
                      {index < placeholders.length - 1 ? ', ' : ''}
                    </span>
                  ))}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button type='submit' disabled={updateOption.isPending}>
            {updateOption.isPending
              ? t('Saving...')
              : t('Save email templates')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
