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
import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatQuota } from '@/lib/format'

import { getInvitationSetting, updateInvitationSetting } from '../api'
import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'
import type { InvitationMode, InvitationSetting } from '../types'

const quotaSchema = z.object({
  QuotaForNewUser: z.coerce.number().min(0),
  PreConsumedQuota: z.coerce.number().min(0),
  TopUpLink: z.string(),
  general_setting: z.object({
    docs_link: z.string(),
  }),
  quota_setting: z.object({
    enable_free_model_pre_consume: z.boolean(),
  }),
})

type QuotaFormValues = z.infer<typeof quotaSchema>
type QuotaInputValue = number | ''

function formatQuotaInputValue(value: QuotaInputValue): string {
  return formatQuota(value === '' ? 0 : value)
}

type QuotaSettingsSectionProps = {
  defaultValues: QuotaFormValues
  complianceConfirmed?: boolean
}

function InvitationSettingsEditor({
  draft,
  loading,
  blockedByCompliance,
  onChange,
}: {
  draft: InvitationSetting | null
  loading: boolean
  blockedByCompliance: boolean
  onChange: (setting: InvitationSetting) => void
}) {
  const { t } = useTranslation()

  const updateDraft = <K extends keyof InvitationSetting>(
    key: K,
    value: InvitationSetting[K]
  ) => {
    if (draft) onChange({ ...draft, [key]: value })
  }

  if (loading || !draft) {
    return (
      <div className='space-y-4'>
        <Skeleton className='h-9 w-full rounded-lg' />
        <Skeleton className='h-24 w-full rounded-lg' />
      </div>
    )
  }

  return (
    <div className='space-y-5'>
      <div>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Choose how referral rewards are issued. Both parameter sets are kept when switching modes.'
          )}
        </p>
      </div>

      <Tabs
        value={draft.mode}
        onValueChange={(value) => updateDraft('mode', value as InvitationMode)}
      >
        <TabsList className='grid h-auto w-full grid-cols-3'>
          <TabsTrigger value='disabled' className='min-h-8 whitespace-normal'>
            {t('Disabled')}
          </TabsTrigger>
          <TabsTrigger value='fixed' className='min-h-8 whitespace-normal'>
            {t('Fixed registration reward')}
          </TabsTrigger>
          <TabsTrigger value='rebate' className='min-h-8 whitespace-normal'>
            {t('Top-up rebate')}
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {draft.mode === 'fixed' ? (
        <div className='grid gap-5 sm:grid-cols-2'>
          <div className='space-y-2'>
            <label
              className='text-sm font-medium'
              htmlFor='fixed-inviter-quota'
            >
              {t('Inviter quota')}
            </label>
            <Input
              id='fixed-inviter-quota'
              type='number'
              min={0}
              value={draft.fixed_inviter_quota}
              onChange={(event) =>
                updateDraft(
                  'fixed_inviter_quota',
                  Math.max(0, event.currentTarget.valueAsNumber || 0)
                )
              }
            />
            <p className='text-muted-foreground text-xs'>
              {formatQuota(draft.fixed_inviter_quota)}
            </p>
          </div>
          <div className='space-y-2'>
            <label
              className='text-sm font-medium'
              htmlFor='fixed-invitee-quota'
            >
              {t('Invitee quota')}
            </label>
            <Input
              id='fixed-invitee-quota'
              type='number'
              min={0}
              value={draft.fixed_invitee_quota}
              onChange={(event) =>
                updateDraft(
                  'fixed_invitee_quota',
                  Math.max(0, event.currentTarget.valueAsNumber || 0)
                )
              }
            />
            <p className='text-muted-foreground text-xs'>
              {formatQuota(draft.fixed_invitee_quota)}
            </p>
          </div>
        </div>
      ) : null}

      {draft.mode === 'rebate' ? (
        <div className='space-y-4'>
          <div className='grid gap-5 sm:grid-cols-2'>
            <div className='space-y-2'>
              <label
                className='text-sm font-medium'
                htmlFor='invitation-rebate-rate'
              >
                {t('Rebate rate (%)')}
              </label>
              <Input
                id='invitation-rebate-rate'
                type='number'
                min={0.01}
                max={100}
                step={0.01}
                value={draft.rebate_bps / 100}
                onChange={(event) =>
                  updateDraft(
                    'rebate_bps',
                    Math.min(
                      10000,
                      Math.max(
                        0,
                        Math.round(
                          (event.currentTarget.valueAsNumber || 0) * 100
                        )
                      )
                    )
                  )
                }
              />
            </div>
            <div className='space-y-2'>
              <label
                className='text-sm font-medium'
                htmlFor='invitation-topup-count'
              >
                {t('Eligible top-ups')}
              </label>
              <Input
                id='invitation-topup-count'
                type='number'
                min={1}
                max={100}
                step={1}
                value={draft.rebate_topup_count}
                onChange={(event) =>
                  updateDraft(
                    'rebate_topup_count',
                    Math.min(
                      100,
                      Math.max(
                        0,
                        Math.trunc(event.currentTarget.valueAsNumber || 0)
                      )
                    )
                  )
                }
              />
            </div>
          </div>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Successful wallet top-ups are counted from the referral date, including top-ups made while another mode is active.'
            )}
          </p>
        </div>
      ) : null}

      {draft.mode === 'disabled' ? (
        <p className='text-muted-foreground text-sm'>
          {t(
            'New referral rewards are disabled. Saved fixed and rebate parameters are retained.'
          )}
        </p>
      ) : null}

      {blockedByCompliance ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {t(
              'Non-zero invitation rewards require compliance confirmation in Payment Gateway settings.'
            )}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  )
}

export function QuotaSettingsSection({
  defaultValues,
  complianceConfirmed = true,
}: QuotaSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [savedInvitation, setSavedInvitation] =
    useState<InvitationSetting | null>(null)
  const [invitationDraft, setInvitationDraft] =
    useState<InvitationSetting | null>(null)
  const [invitationLoading, setInvitationLoading] = useState(true)
  const [invitationSaving, setInvitationSaving] = useState(false)

  useEffect(() => {
    let active = true
    void getInvitationSetting()
      .then((response) => {
        if (!active) return
        if (!response.success || !response.data) {
          throw new Error(response.message || 'Missing invitation setting')
        }
        setSavedInvitation(response.data)
        setInvitationDraft(response.data)
      })
      .catch((error: unknown) => {
        if (!active) return
        toast.error(
          error instanceof Error
            ? error.message
            : t('Failed to load invitation settings')
        )
      })
      .finally(() => {
        if (active) setInvitationLoading(false)
      })
    return () => {
      active = false
    }
  }, [t])

  const invitationDirty = useMemo(
    () =>
      Boolean(
        savedInvitation &&
        invitationDraft &&
        JSON.stringify(savedInvitation) !== JSON.stringify(invitationDraft)
      ),
    [invitationDraft, savedInvitation]
  )
  const invitationRewardsEnabled = Boolean(
    invitationDraft &&
    (invitationDraft.mode === 'rebate' ||
      (invitationDraft.mode === 'fixed' &&
        (invitationDraft.fixed_inviter_quota > 0 ||
          invitationDraft.fixed_invitee_quota > 0)))
  )
  const invitationBlockedByCompliance =
    invitationRewardsEnabled && !complianceConfirmed
  const handleNumberChange =
    (onChange: (value: QuotaInputValue) => void) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber
      onChange(Number.isNaN(value) ? '' : value)
    }

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<QuotaFormValues>({
      resolver: zodResolver(quotaSchema) as Resolver<
        QuotaFormValues,
        unknown,
        QuotaFormValues
      >,
      defaultValues,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key,
            value: value as string | number | boolean,
          })
        }
      },
    })

  const handleSave = async () => {
    if (invitationLoading || invitationSaving) return
    if (!isDirty && !invitationDirty) {
      toast.info(t('No changes to save'))
      return
    }
    if (invitationBlockedByCompliance) {
      toast.error(
        t(
          'Non-zero invitation rewards require compliance confirmation in Payment Gateway settings.'
        )
      )
      return
    }

    setInvitationSaving(true)
    try {
      if (invitationDirty && invitationDraft) {
        const response = await updateInvitationSetting(invitationDraft)
        if (!response.success || !response.data) {
          throw new Error(
            response.message || t('Failed to update invitation settings')
          )
        }
        setSavedInvitation(response.data)
        setInvitationDraft(response.data)
      }
      if (isDirty) {
        await handleSubmit()
      }
      if (invitationDirty) {
        toast.success(t('Invitation settings updated'))
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to update invitation settings')
      )
    } finally {
      setInvitationSaving(false)
    }
  }

  return (
    <>
      <FormNavigationGuard when={isDirty || invitationDirty} />
      <SettingsSection title={t('Invitation Program')}>
        <InvitationSettingsEditor
          draft={invitationDraft}
          loading={invitationLoading}
          blockedByCompliance={invitationBlockedByCompliance}
          onChange={setInvitationDraft}
        />
      </SettingsSection>

      <SettingsSection title={t('Quota Settings')}>
        <Form {...form}>
          <SettingsForm
            onSubmit={(event) => {
              event.preventDefault()
              void handleSave()
            }}
          >
            <SettingsPageFormActions
              onSave={handleSave}
              isSaving={
                updateOption.isPending || isSubmitting || invitationSaving
              }
              isSaveDisabled={invitationLoading || !invitationDraft}
            />
            <FormDirtyIndicator isDirty={isDirty || invitationDirty} />
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='QuotaForNewUser'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('New User Quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        value={field.value ?? ''}
                        onChange={handleNumberChange(field.onChange)}
                        name={field.name}
                        onBlur={field.onBlur}
                        ref={field.ref}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Initial quota given to new users ({{formattedQuota}})',
                        {
                          formattedQuota: formatQuotaInputValue(field.value),
                        }
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='PreConsumedQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Pre-Consumed Quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        value={field.value ?? ''}
                        onChange={handleNumberChange(field.onChange)}
                        name={field.name}
                        onBlur={field.onBlur}
                        ref={field.ref}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Quota consumed before charging users')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <SettingsFormGridItem span='full'>
                <FormField
                  control={form.control}
                  name='quota_setting.enable_free_model_pre_consume'
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>
                          {t('Pre-Consume for Free Models')}
                        </FormLabel>
                        <FormDescription>
                          {t(
                            'When enabled, zero-cost models also pre-consume quota before final settlement.'
                          )}
                        </FormDescription>
                      </SettingsSwitchContent>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                          disabled={updateOption.isPending}
                        />
                      </FormControl>
                    </SettingsSwitchItem>
                  )}
                />
              </SettingsFormGridItem>

              <FormField
                control={form.control}
                name='TopUpLink'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Top-Up Link')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('https://example.com/topup')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('External link for users to purchase quota')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='general_setting.docs_link'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Documentation Link')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('https://docs.example.com')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Link to your documentation site')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGrid>
          </SettingsForm>
        </Form>
      </SettingsSection>
    </>
  )
}
