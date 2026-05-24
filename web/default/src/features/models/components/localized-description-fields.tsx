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
import type {
  Control,
  FieldPath,
  FieldValues,
  PathValue,
} from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import type { SupportedLocale } from '../types'

type LocaleOption = {
  value: SupportedLocale
  labelKey: string
}

const DESCRIPTION_LOCALES: LocaleOption[] = [
  { value: 'en', labelKey: 'English' },
  { value: 'zh', labelKey: 'Chinese' },
  { value: 'es', labelKey: 'Spanish' },
  { value: 'fr', labelKey: 'French' },
  { value: 'ja', labelKey: 'Japanese' },
  { value: 'ru', labelKey: 'Russian' },
  { value: 'vi', labelKey: 'Vietnamese' },
]

type LocalizedDescriptionFieldsProps<TFieldValues extends FieldValues> = {
  control: Control<TFieldValues>
  basePlaceholder: string
  localizedPlaceholder: string
}

export function LocalizedDescriptionFields<TFieldValues extends FieldValues>({
  control,
  basePlaceholder,
  localizedPlaceholder,
}: LocalizedDescriptionFieldsProps<TFieldValues>) {
  const { t } = useTranslation()
  const descriptionName = 'description' as FieldPath<TFieldValues>
  const emptyStringValue = '' as PathValue<
    TFieldValues,
    FieldPath<TFieldValues>
  >

  return (
    <div className='space-y-4'>
      <FormField
        control={control}
        name={descriptionName}
        defaultValue={emptyStringValue}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Default description')}</FormLabel>
            <FormControl>
              <Textarea
                placeholder={t(basePlaceholder)}
                rows={3}
                {...field}
                value={(field.value as string | undefined) ?? ''}
              />
            </FormControl>
            <FormDescription>
              {t(
                'Used when a matching localized description is not configured.'
              )}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <div className='space-y-2'>
        <div>
          <div className='text-sm leading-none font-medium'>
            {t('Localized descriptions')}
          </div>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Fill only the languages that need custom descriptions.')}
          </p>
        </div>
        <Tabs defaultValue='en'>
          <TabsList className='h-auto max-w-full flex-wrap justify-start'>
            {DESCRIPTION_LOCALES.map((locale) => (
              <TabsTrigger key={locale.value} value={locale.value}>
                {t(locale.labelKey)}
              </TabsTrigger>
            ))}
          </TabsList>

          {DESCRIPTION_LOCALES.map((locale) => {
            const label = t(locale.labelKey)
            return (
              <TabsContent
                key={locale.value}
                value={locale.value}
                className='mt-3'
              >
                <FormField
                  control={control}
                  name={
                    `description_i18n.${locale.value}` as FieldPath<TFieldValues>
                  }
                  defaultValue={emptyStringValue}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('{{language}} description', { language: label })}
                      </FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder={t(localizedPlaceholder, {
                            language: label,
                          })}
                          rows={3}
                          {...field}
                          value={(field.value as string | undefined) ?? ''}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </TabsContent>
            )
          })}
        </Tabs>
      </div>
    </div>
  )
}
