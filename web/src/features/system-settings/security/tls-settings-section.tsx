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
import { useEffect, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const numericString = z.string().refine((value) => {
  const trimmed = value.trim()
  if (!trimmed) return true
  return !Number.isNaN(Number(trimmed)) && Number(trimmed) >= 0
}, 'Enter a non-negative number or leave empty')

const tlsSchema = z.object({
  'tls_enabled': z.boolean().default(false),
  'tls_mode': z.enum(['auto', 'manual']).default('auto'),
  'tls_min_version': z.enum(['1.2', '1.3']).default('1.2'),
  'http_to_https_redirect': z.boolean().default(false),
  'http_redirect_port': numericString.default('8080'),
  'tls_cert_file': z.string().default(''),
  'tls_key_file': z.string().default(''),
})

type TLSFormValues = z.output<typeof tlsSchema>
type TLSFormInput = z.input<typeof tlsSchema>

type FlatTLSDefaults = {
  'tls_enabled': boolean
  'tls_mode': 'auto' | 'manual'
  'tls_min_version': '1.2' | '1.3'
  'http_to_https_redirect': boolean
  'http_redirect_port': string
  'tls_cert_file': string
  'tls_key_file': string
}

type TLSSettingsSectionProps = {
  defaultValues: FlatTLSDefaults
}

const buildFormDefaults = (
  defaults: TLSSettingsSectionProps['defaultValues']
): TLSFormInput => ({
  'tls_enabled': defaults['tls_enabled'],
  'tls_mode': defaults['tls_mode'],
  'tls_min_version': defaults['tls_min_version'],
  'http_to_https_redirect': defaults['http_to_https_redirect'],
  'http_redirect_port': defaults['http_redirect_port'],
  'tls_cert_file': defaults['tls_cert_file'],
  'tls_key_file': defaults['tls_key_file'],
})

const normalizeDefaults = (
  defaults: TLSSettingsSectionProps['defaultValues']
): FlatTLSDefaults => ({
  'tls_enabled': defaults['tls_enabled'] ?? false,
  'tls_mode': defaults['tls_mode'] ?? 'auto',
  'tls_min_version': defaults['tls_min_version'] ?? '1.2',
  'http_to_https_redirect': defaults['http_to_https_redirect'] ?? false,
  'http_redirect_port': (defaults['http_redirect_port'] ?? '8080').trim(),
  'tls_cert_file': (defaults['tls_cert_file'] ?? '').trim(),
  'tls_key_file': (defaults['tls_key_file'] ?? '').trim(),
})

const normalizeFormValues = (values: TLSFormValues): FlatTLSDefaults => ({
  'tls_enabled': values['tls_enabled'] ?? false,
  'tls_mode': values['tls_mode'] ?? 'auto',
  'tls_min_version': values['tls_min_version'] ?? '1.2',
  'http_to_https_redirect': values['http_to_https_redirect'] ?? false,
  'http_redirect_port': (values['http_redirect_port'] ?? '').trim(),
  'tls_cert_file': (values['tls_cert_file'] ?? '').trim(),
  'tls_key_file': (values['tls_key_file'] ?? '').trim(),
})

export function TLSSettingsSection({
  defaultValues,
}: TLSSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<FlatTLSDefaults>(
    normalizeDefaults(defaultValues)
  )

  const formDefaults = useMemo(
    () => buildFormDefaults(defaultValues),
    [defaultValues]
  )

  const form = useForm<TLSFormInput, unknown, TLSFormValues>({
    defaultValues: formDefaults,
  })

  useEffect(() => {
    baselineRef.current = normalizeDefaults(defaultValues)
    form.reset(buildFormDefaults(defaultValues))
  }, [defaultValues, form])

  const onSubmit = async (data: TLSFormValues) => {
    const normalized = normalizeFormValues(data)
    const updates = (
      Object.keys(normalized) as Array<keyof FlatTLSDefaults>
    ).filter((key) => normalized[key] !== baselineRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      const value = normalized[key]
      await updateOption.mutateAsync({
        key,
        value,
      })
    }

    baselineRef.current = normalized
  }

  const tlsEnabled = form.watch('tls_enabled')
  const tlsMode = form.watch('tls_mode')
  const httpRedirect = form.watch('http_to_https_redirect')

  return (
    <SettingsSection title={t('TLS Settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel={t('Save TLS settings')}
          />
          <FormField
            control={form.control}
            name="tls_enabled"
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable TLS')}</FormLabel>
                  <FormDescription>
                    {t('启用 TLS/HTTPS 支持')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          {tlsEnabled && (
            <>
              <FormField
                control={form.control}
                name="tls_mode"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('TLS Mode')}</FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      value={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="auto">{t('Auto (Self-signed)')}</SelectItem>
                        <SelectItem value="manual">{t('Manual (Upload)')}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {tlsMode === 'auto'
                        ? t('Self-signed certificates are only valid for local network use. Browsers will show an insecure warning.')
                        : t('For public internet access, upload a trusted certificate via environment variables: TLS_CERT_FILE and TLS_KEY_FILE.')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {tlsMode === 'auto' && (
                <FormItem>
                  <FormLabel>{t('Certificate Path')}</FormLabel>
                  <FormControl>
                    <Input
                      value="/data/tls/cert.pem"
                      disabled
                      readOnly
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Auto-generated certificate will be stored here.')}
                  </FormDescription>
                </FormItem>
              )}

              {tlsMode === 'manual' && (
                <>
                  <FormField
                    control={form.control}
                    name="tls_cert_file"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Certificate File Path')}</FormLabel>
                        <FormControl>
                          <Input
                            placeholder={t('Path to certificate file')}
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Full path to the TLS certificate file (.crt or .pem)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="tls_key_file"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Key File Path')}</FormLabel>
                        <FormControl>
                          <Input
                            type="password"
                            placeholder={t('Path to private key file')}
                            {...field}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Full path to the TLS private key file (.key)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </>
              )}

              <FormField
                control={form.control}
                name="tls_min_version"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum TLS Version')}</FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      value={field.value}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="1.2">TLS 1.2</SelectItem>
                        <SelectItem value="1.3">TLS 1.3</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {t('Minimum TLS protocol version to accept')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="http_to_https_redirect"
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t('HTTP to HTTPS Redirect')}</FormLabel>
                      <FormDescription>
                        {t('Redirect HTTP requests to HTTPS')}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />

              {httpRedirect && (
                <FormField
                  control={form.control}
                  name="http_redirect_port"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Redirect Port')}</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          placeholder="8080"
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('Port to listen for HTTP redirects (must be different from main port)')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </>
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
