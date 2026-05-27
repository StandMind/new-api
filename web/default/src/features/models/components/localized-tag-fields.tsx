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
import { TagInput } from '@/components/tag-input'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { SupportedLocale } from '../types'

type LocaleOption = {
  value: SupportedLocale
  labelKey: string
}

const TAG_LOCALES: LocaleOption[] = [
  { value: 'en', labelKey: 'English' },
  { value: 'zh', labelKey: 'Chinese' },
  { value: 'es', labelKey: 'Spanish' },
  { value: 'fr', labelKey: 'French' },
  { value: 'ja', labelKey: 'Japanese' },
  { value: 'ru', labelKey: 'Russian' },
  { value: 'vi', labelKey: 'Vietnamese' },
]

type LocalizedTagFieldsProps<TFieldValues extends FieldValues> = {
  control: Control<TFieldValues>
  basePlaceholder: string
  localizedPlaceholder: string
}

export function LocalizedTagFields<TFieldValues extends FieldValues>({
  control,
  basePlaceholder,
  localizedPlaceholder,
}: LocalizedTagFieldsProps<TFieldValues>) {
  const { t } = useTranslation()
  const tagsName = 'tags' as FieldPath<TFieldValues>
  const emptyTags = [] as PathValue<TFieldValues, FieldPath<TFieldValues>>

  return (
    <div className='space-y-4'>
      <FormField
        control={control}
        name={tagsName}
        defaultValue={emptyTags}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Default tags')}</FormLabel>
            <FormControl>
              <TagInput
                value={(field.value as string[] | undefined) ?? []}
                onChange={field.onChange}
                placeholder={t(basePlaceholder)}
              />
            </FormControl>
            <FormDescription>
              {t('Used when a matching localized tag set is not configured.')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <div className='space-y-2'>
        <div>
          <div className='text-sm leading-none font-medium'>
            {t('Localized tags')}
          </div>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Fill only the languages that need custom tags.')}
          </p>
        </div>
        <Tabs defaultValue='en'>
          <TabsList className='h-auto max-w-full flex-wrap justify-start'>
            {TAG_LOCALES.map((locale) => (
              <TabsTrigger key={locale.value} value={locale.value}>
                {t(locale.labelKey)}
              </TabsTrigger>
            ))}
          </TabsList>

          {TAG_LOCALES.map((locale) => {
            const label = t(locale.labelKey)
            return (
              <TabsContent
                key={locale.value}
                value={locale.value}
                className='mt-3'
              >
                <FormField
                  control={control}
                  name={`tags_i18n.${locale.value}` as FieldPath<TFieldValues>}
                  defaultValue={emptyTags}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('{{language}} tags', { language: label })}
                      </FormLabel>
                      <FormControl>
                        <TagInput
                          value={(field.value as string[] | undefined) ?? []}
                          onChange={field.onChange}
                          placeholder={t(localizedPlaceholder, {
                            language: label,
                          })}
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
