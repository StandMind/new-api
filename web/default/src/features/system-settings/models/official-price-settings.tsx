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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { StaticDataTable } from '@/components/data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
import { getModels } from '@/features/models/api'
import type { OfficialPrice, OfficialPriceUnit } from '@/features/pricing/types'

import { useUpdateOption } from '../hooks/use-update-option'
import { safeJsonParse } from '../utils/json-parser'

const TOKEN_UNIT = 'usd_per_million_input_tokens'
const REQUEST_UNIT = 'usd_per_request'

const tierSchema = z.object({
  up_to_input_tokens: z.string(),
  price: z.string().refine(
    (value) => {
      const price = Number(value)
      return Number.isFinite(price) && price > 0
    },
    { message: 'Price must be greater than zero' }
  ),
})

const createOfficialPriceSchema = (t: (key: string) => string) =>
  z
    .object({
      model: z.string().trim().min(1, t('Select a model')),
      unit: z.enum([TOKEN_UNIT, REQUEST_UNIT]),
      source_model: z.string(),
      source_url: z.string().refine(
        (value) =>
          value.trim() === '' ||
          (() => {
            try {
              const url = new URL(value)
              return url.protocol === 'http:' || url.protocol === 'https:'
            } catch {
              return false
            }
          })(),
        t('Enter a valid HTTP or HTTPS URL')
      ),
      verified_at: z
        .string()
        .refine(
          (value) => value === '' || /^\d{4}-\d{2}-\d{2}$/.test(value),
          t('Enter a valid verification date')
        ),
      notes: z.string(),
      tiers: z.array(tierSchema).min(1),
    })
    .superRefine((values, context) => {
      if (values.unit === REQUEST_UNIT && values.tiers.length !== 1) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['tiers'],
          message: t('Per-request references support exactly one tier'),
        })
        return
      }

      let previousLimit = 0
      values.tiers.forEach((tier, index) => {
        const isLast = index === values.tiers.length - 1
        const rawLimit = tier.up_to_input_tokens.trim()
        if (isLast) {
          if (rawLimit !== '') {
            context.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['tiers', index, 'up_to_input_tokens'],
              message: t('The final tier must not have an upper limit'),
            })
          }
          return
        }

        const limit = Number(rawLimit)
        if (!Number.isInteger(limit) || limit <= previousLimit) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['tiers', index, 'up_to_input_tokens'],
            message: t('Tier limits must be positive and strictly increasing'),
          })
          return
        }
        previousLimit = limit
      })
    })

type OfficialPriceFormValues = z.infer<
  ReturnType<typeof createOfficialPriceSchema>
>

type OfficialPriceSettingsProps = {
  defaultValue: string
}

type EditorTarget = {
  model: string
  existing: boolean
} | null

function parseOfficialPrices(value: string): Record<string, OfficialPrice> {
  return safeJsonParse<Record<string, OfficialPrice>>(value, {
    fallback: {},
    silent: true,
  })
}

function toFormValues(
  model: string,
  price?: OfficialPrice
): OfficialPriceFormValues {
  return {
    model,
    unit: price?.unit ?? TOKEN_UNIT,
    source_model: price?.source_model ?? model,
    source_url: price?.source_url ?? '',
    verified_at: price?.verified_at ?? '',
    notes: price?.notes ?? '',
    tiers: (price?.tiers ?? [{ price: 0 }]).map((tier) => ({
      up_to_input_tokens:
        tier.up_to_input_tokens === undefined
          ? ''
          : String(tier.up_to_input_tokens),
      price: tier.price > 0 ? String(tier.price) : '',
    })),
  }
}

function toOfficialPrice(values: OfficialPriceFormValues): OfficialPrice {
  const price: OfficialPrice = {
    unit: values.unit,
    tiers: values.tiers.map((tier) => ({
      ...(tier.up_to_input_tokens.trim() === ''
        ? {}
        : { up_to_input_tokens: Number(tier.up_to_input_tokens) }),
      price: Number(tier.price),
    })),
  }
  if (values.source_model.trim()) {
    price.source_model = values.source_model.trim()
  }
  if (values.source_url.trim()) {
    price.source_url = values.source_url.trim()
  }
  if (values.verified_at) {
    price.verified_at = values.verified_at
  }
  if (values.notes.trim()) {
    price.notes = values.notes.trim()
  }
  return price
}

function OfficialPriceEditorSheet(props: {
  target: EditorTarget
  price?: OfficialPrice
  modelOptions: string[]
  onClose: () => void
  onApply: (model: string, price: OfficialPrice, previousModel?: string) => void
}) {
  const { t } = useTranslation()
  const schema = useMemo(() => createOfficialPriceSchema(t), [t])
  const form = useForm<OfficialPriceFormValues>({
    resolver: zodResolver(schema),
    defaultValues: toFormValues('', undefined),
  })
  const tiers = useFieldArray({ control: form.control, name: 'tiers' })
  const unit = form.watch('unit')

  useEffect(() => {
    if (!props.target) return
    form.reset(toFormValues(props.target.model, props.price))
  }, [form, props.price, props.target])

  const requestClose = () => {
    if (
      form.formState.isDirty &&
      !window.confirm(t('Discard unsaved official price changes?'))
    ) {
      return
    }
    props.onClose()
  }

  const handleUnitChange = (value: OfficialPriceUnit) => {
    form.setValue('unit', value, { shouldDirty: true, shouldValidate: true })
    if (value === REQUEST_UNIT && tiers.fields.length !== 1) {
      tiers.replace([{ up_to_input_tokens: '', price: '' }])
    }
  }

  return (
    <Sheet
      open={props.target !== null}
      onOpenChange={(open) => {
        if (!open) requestClose()
      }}
    >
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-xl')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {props.target?.existing
              ? t('Edit official reference price')
              : t('Add official reference price')}
          </SheetTitle>
          <SheetDescription>
            {t('Reference prices are used only for price comparison.')}
          </SheetDescription>
        </SheetHeader>

        <form
          className={sideDrawerFormClassName('gap-5')}
          onSubmit={form.handleSubmit((values) => {
            props.onApply(
              values.model.trim(),
              toOfficialPrice(values),
              props.target?.existing ? props.target.model : undefined
            )
            form.reset(values)
          })}
        >
          <div className='space-y-2'>
            <Label htmlFor='official-model'>{t('Model')}</Label>
            {props.target?.existing ? (
              <Input id='official-model' value={props.target.model} disabled />
            ) : (
              <Combobox
                id='official-model'
                options={props.modelOptions.map((model) => ({
                  value: model,
                  label: model,
                }))}
                value={form.watch('model')}
                onValueChange={(value) =>
                  form.setValue('model', value ?? '', {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
                placeholder={t('Search models...')}
                searchPlaceholder={t('Search models...')}
                emptyText={t('No models found')}
                openOnFocus
              />
            )}
            {form.formState.errors.model && (
              <p className='text-destructive text-xs'>
                {form.formState.errors.model.message}
              </p>
            )}
          </div>

          <div className='space-y-2'>
            <Label>{t('Billing unit')}</Label>
            <Select
              value={unit}
              onValueChange={(value) => {
                if (value) handleUnitChange(value)
              }}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={TOKEN_UNIT}>
                  {t('USD per 1M input tokens')}
                </SelectItem>
                <SelectItem value={REQUEST_UNIT}>
                  {t('USD per request')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className='space-y-3'>
            <div className='flex items-center justify-between gap-3'>
              <Label>{t('Price tiers')}</Label>
              {unit === TOKEN_UNIT && (
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => {
                    const index = Math.max(0, tiers.fields.length - 1)
                    tiers.insert(index, {
                      up_to_input_tokens: '',
                      price: '',
                    })
                  }}
                >
                  <Plus />
                  {t('Add tier')}
                </Button>
              )}
            </div>
            {tiers.fields.map((field, index) => {
              const isLast = index === tiers.fields.length - 1
              return (
                <div
                  key={field.id}
                  className='grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] gap-2 rounded-lg border p-3'
                >
                  <div className='space-y-1.5'>
                    <Label>{t('Up to input tokens')}</Label>
                    <Input
                      type='number'
                      min={1}
                      placeholder={isLast ? t('No limit') : '272000'}
                      disabled={isLast}
                      {...form.register(`tiers.${index}.up_to_input_tokens`)}
                    />
                    {form.formState.errors.tiers?.[index]
                      ?.up_to_input_tokens && (
                      <p className='text-destructive text-xs'>
                        {
                          form.formState.errors.tiers[index]?.up_to_input_tokens
                            ?.message
                        }
                      </p>
                    )}
                  </div>
                  <div className='space-y-1.5'>
                    <Label>{t('Official price (USD)')}</Label>
                    <Input
                      type='number'
                      min='0'
                      step='any'
                      {...form.register(`tiers.${index}.price`)}
                    />
                    {form.formState.errors.tiers?.[index]?.price && (
                      <p className='text-destructive text-xs'>
                        {t(
                          form.formState.errors.tiers[index]?.price?.message ??
                            'Price must be greater than zero'
                        )}
                      </p>
                    )}
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon-sm'
                    className='mt-6'
                    disabled={tiers.fields.length === 1 || isLast}
                    onClick={() => tiers.remove(index)}
                    aria-label={t('Delete tier')}
                  >
                    <Trash2 />
                  </Button>
                </div>
              )
            })}
            {form.formState.errors.tiers?.root?.message && (
              <p className='text-destructive text-xs'>
                {form.formState.errors.tiers.root.message}
              </p>
            )}
          </div>

          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='official-source-model'>{t('Source model')}</Label>
              <Input
                id='official-source-model'
                {...form.register('source_model')}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='official-verified-at'>
                {t('Verification date')}
              </Label>
              <Input
                id='official-verified-at'
                type='date'
                {...form.register('verified_at')}
              />
            </div>
          </div>
          <div className='space-y-2'>
            <Label htmlFor='official-source-url'>{t('Source URL')}</Label>
            <Input
              id='official-source-url'
              type='url'
              {...form.register('source_url')}
            />
            {form.formState.errors.source_url && (
              <p className='text-destructive text-xs'>
                {form.formState.errors.source_url.message}
              </p>
            )}
          </div>
          <div className='space-y-2'>
            <Label htmlFor='official-notes'>{t('Notes')}</Label>
            <Textarea
              id='official-notes'
              rows={4}
              {...form.register('notes')}
            />
          </div>

          <SheetFooter className='mt-auto px-0 pb-0'>
            <Button type='button' variant='outline' onClick={requestClose}>
              {t('Cancel')}
            </Button>
            <Button type='submit'>
              <Save />
              {t('Apply')}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}

export function OfficialPriceSettings(props: OfficialPriceSettingsProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [prices, setPrices] = useState<Record<string, OfficialPrice>>(() =>
    parseOfficialPrices(props.defaultValue)
  )
  const [savedSnapshot, setSavedSnapshot] = useState(() =>
    JSON.stringify(parseOfficialPrices(props.defaultValue))
  )
  const [search, setSearch] = useState('')
  const [target, setTarget] = useState<EditorTarget>(null)

  useEffect(() => {
    const next = parseOfficialPrices(props.defaultValue)
    setPrices(next)
    setSavedSnapshot(JSON.stringify(next))
  }, [props.defaultValue])

  const modelsQuery = useQuery({
    queryKey: ['official-price-model-options'],
    queryFn: () => getModels({ p: 1, page_size: 1000 }),
    staleTime: 60_000,
  })
  const modelOptions = useMemo(
    () =>
      [
        ...new Set([
          ...Object.keys(prices),
          ...(modelsQuery.data?.data?.items ?? []).map(
            (model) => model.model_name
          ),
        ]),
      ].sort(),
    [modelsQuery.data?.data?.items, prices]
  )
  const rows = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    return Object.entries(prices)
      .map(([model, price]) => ({ model, price }))
      .filter(
        (row) =>
          !keyword ||
          row.model.toLowerCase().includes(keyword) ||
          row.price.source_model?.toLowerCase().includes(keyword)
      )
      .sort((left, right) => left.model.localeCompare(right.model))
  }, [prices, search])
  const dirty = JSON.stringify(prices) !== savedSnapshot

  const save = async () => {
    const serialized = JSON.stringify(prices)
    const response = await updateOption.mutateAsync({
      key: 'official_price_setting.model_prices',
      value: serialized,
    })
    if (!response.success) return
    setSavedSnapshot(serialized)
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center gap-2'>
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t('Search models or source models...')}
          className='min-w-56 flex-1'
        />
        <Button
          variant='outline'
          onClick={() => setTarget({ model: '', existing: false })}
        >
          <Plus />
          {t('Add reference price')}
        </Button>
        <Button onClick={save} disabled={!dirty || updateOption.isPending}>
          <Save />
          {updateOption.isPending ? t('Saving...') : t('Save changes')}
        </Button>
      </div>

      <StaticDataTable
        data={rows}
        getRowKey={(row) => row.model}
        emptyContent={t('No official reference prices configured')}
        columns={[
          {
            id: 'model',
            header: t('Model'),
            cell: (row) => (
              <span className='font-mono font-medium'>{row.model}</span>
            ),
          },
          {
            id: 'unit',
            header: t('Billing unit'),
            cell: (row) =>
              row.price.unit === REQUEST_UNIT
                ? t('Per request')
                : t('Per 1M input tokens'),
          },
          {
            id: 'tiers',
            header: t('Price tiers'),
            cell: (row) =>
              row.price.tiers
                .map((tier) =>
                  tier.up_to_input_tokens
                    ? `${tier.up_to_input_tokens}: $${tier.price}`
                    : `$${tier.price}`
                )
                .join(' → '),
          },
          {
            id: 'verified',
            header: t('Verification date'),
            cell: (row) => row.price.verified_at || '-',
          },
          {
            id: 'actions',
            header: <span className='sr-only'>{t('Actions')}</span>,
            className: 'text-right',
            cellClassName: 'text-right',
            cell: (row) => (
              <div className='flex justify-end gap-1'>
                <Button
                  variant='ghost'
                  size='sm'
                  onClick={() =>
                    setTarget({ model: row.model, existing: true })
                  }
                >
                  {t('Edit')}
                </Button>
                <Button
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Delete')}
                  onClick={() => {
                    if (!window.confirm(t('Delete this reference price?'))) {
                      return
                    }
                    setPrices((current) => {
                      const next = { ...current }
                      delete next[row.model]
                      return next
                    })
                  }}
                >
                  <Trash2 />
                </Button>
              </div>
            ),
          },
        ]}
      />

      {dirty && (
        <p className='text-muted-foreground text-xs'>
          {t('Official reference price changes have not been saved yet.')}
        </p>
      )}

      <OfficialPriceEditorSheet
        target={target}
        price={target ? prices[target.model] : undefined}
        modelOptions={modelOptions}
        onClose={() => setTarget(null)}
        onApply={(model, price, previousModel) => {
          setPrices((current) => {
            const next = { ...current }
            if (previousModel && previousModel !== model) {
              delete next[previousModel]
            }
            next[model] = price
            return next
          })
          setTarget(null)
          toast.success(
            t('Reference price applied. Save changes to publish it.')
          )
        }}
      />
    </div>
  )
}
