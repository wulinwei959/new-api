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
import { Plus, Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Dialog } from '@/components/dialog'
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
import { Input } from '@/components/ui/input'

import {
  getOptionValue,
  useSystemOptions,
} from '@/features/system-settings/hooks/use-system-options'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import { isObjectRecord } from '@/features/system-settings/utils/json-validators'
import { safeJsonParseWithValidation } from '@/features/system-settings/utils/json-parser'

// Option key that stores the per-model RPM/TPM limits as a JSON object:
// { "<model>": { "rpm": N, "tpm": M } } where 0 means unlimited for that
// dimension and an absent model means no limit.
const MODEL_RATE_LIMIT_OPTION = 'ModelRateLimitByModel'

type ModelRateLimitData = {
  model: string
  rpm: number
  tpm: number
}

type ModelRateLimitEntry = ModelRateLimitData

const modelRateLimitSchema = z.object({
  model: z.string().min(1, 'Model name is required'),
  rpm: z
    .number()
    .min(0, 'Must be ≥ 0')
    .max(2147483647, 'Must be ≤ 2,147,483,647'),
  tpm: z
    .number()
    .min(0, 'Must be ≥ 0')
    .max(9007199254740991, 'Must be a valid number'),
})

type ModelRateLimitFormValues = z.infer<typeof modelRateLimitSchema>

const DIALOG_FORM_ID = 'model-rate-limit-form'

function readModelRateLimits(
  raw: string,
): Record<string, { rpm: number; tpm: number }> {
  const parsed = safeJsonParseWithValidation<Record<string, unknown>>(raw, {
    fallback: {},
    validator: isObjectRecord,
    silent: true,
  })
  const result: Record<string, { rpm: number; tpm: number }> = {}
  for (const [model, value] of Object.entries(parsed)) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) continue
    const rec = value as Record<string, unknown>
    result[model] = {
      rpm: typeof rec.rpm === 'number' ? rec.rpm : 0,
      tpm: typeof rec.tpm === 'number' ? rec.tpm : 0,
    }
  }
  return result
}

export function ModelRateLimitSection() {
  const { t } = useTranslation()
  const { data, isLoading } = useSystemOptions()
  const updateOption = useUpdateOption()

  const [searchText, setSearchText] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editData, setEditData] = useState<ModelRateLimitEntry | null>(null)

  const raw = getOptionValue(data?.data, { [MODEL_RATE_LIMIT_OPTION]: '' })[
    MODEL_RATE_LIMIT_OPTION
  ]

  const current = useMemo(() => readModelRateLimits(raw), [raw])

  const entries: ModelRateLimitEntry[] = useMemo(
    () =>
      Object.entries(current).map(([model, limits]) => ({
        model,
        rpm: limits.rpm,
        tpm: limits.tpm,
      })),
    [current],
  )

  const filtered = useMemo(() => {
    if (!searchText) return entries
    const lower = searchText.toLowerCase()
    return entries.filter((entry) =>
      entry.model.toLowerCase().includes(lower),
    )
  }, [entries, searchText])

  const persist = async (next: Record<string, { rpm: number; tpm: number }>) => {
    await updateOption.mutateAsync({
      key: MODEL_RATE_LIMIT_OPTION,
      value: JSON.stringify(next, null, 2),
    })
  }

  const handleSave = async (data: ModelRateLimitData) => {
    const next = readModelRateLimits(raw)
    if (editData && editData.model !== data.model) {
      delete next[editData.model]
    }
    next[data.model] = { rpm: data.rpm, tpm: data.tpm }
    await persist(next)
    setDialogOpen(false)
  }

  const handleDelete = async (model: string) => {
    const next = readModelRateLimits(raw)
    delete next[model]
    await persist(next)
  }

  if (isLoading) {
    return (
      <div className='text-muted-foreground flex min-h-40 items-center justify-center text-sm'>
        {t('Loading...')}
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Per-model rate limits. RPM caps requests per minute; TPM caps tokens per minute. A value of 0 leaves that dimension unlimited.',
        )}
      </p>

      <div className='flex items-center gap-4'>
        <div className='relative flex-1'>
          <Search className='text-muted-foreground absolute top-2.5 left-2.5 h-4 w-4' />
          <Input
            placeholder={t('Search model names...')}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            className='pl-9'
          />
        </div>
        <Button
          onClick={() => {
            setEditData(null)
            setDialogOpen(true)
          }}
        >
          <Plus className='mr-2 h-4 w-4' />
          {t('Add model')}
        </Button>
      </div>

      <StaticDataTable
        data={filtered}
        getRowKey={(entry) => entry.model}
        emptyContent={
          searchText
            ? t('No models match your search')
            : t(
                'No per-model rate limits configured. Click "Add model" to get started.',
              )
        }
        columns={[
          {
            id: 'model',
            header: t('Model'),
            cellClassName: 'font-medium',
            cell: (entry) => entry.model,
          },
          {
            id: 'rpm',
            header: t('Requests / minute'),
            className: 'text-right',
            cellClassName: 'text-right',
            cell: (entry) => (
              <span className='font-mono'>
                {entry.rpm === 0 ? t('Unlimited') : entry.rpm.toLocaleString()}
              </span>
            ),
          },
          {
            id: 'tpm',
            header: t('Tokens / minute'),
            className: 'text-right',
            cellClassName: 'text-right',
            cell: (entry) => (
              <span className='font-mono'>
                {entry.tpm === 0
                  ? t('Unlimited')
                  : entry.tpm.toLocaleString()}
              </span>
            ),
          },
          {
            id: 'actions',
            header: t('Actions'),
            className: 'text-right',
            cellClassName: 'text-right',
            cell: (entry) => (
              <StaticRowActions
                editLabel={t('Edit')}
                deleteLabel={t('Delete')}
                menuLabel={t('Open menu')}
                onEdit={() => {
                  setEditData(entry)
                  setDialogOpen(true)
                }}
                onDelete={() => void handleDelete(entry.model)}
              />
            ),
          },
        ]}
      />

      <ModelRateLimitDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        editData={editData}
        saving={updateOption.isPending}
        onSave={handleSave}
      />
    </div>
  )
}

type ModelRateLimitDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  editData: ModelRateLimitEntry | null
  saving: boolean
  onSave: (data: ModelRateLimitData) => void
}

function ModelRateLimitDialog({
  open,
  onOpenChange,
  editData,
  saving,
  onSave,
}: ModelRateLimitDialogProps) {
  const { t } = useTranslation()
  const isEditMode = !!editData

  const form = useForm<ModelRateLimitFormValues>({
    resolver: zodResolver(modelRateLimitSchema),
    defaultValues: { model: '', rpm: 0, tpm: 0 },
  })

  const handleSubmit = (values: ModelRateLimitFormValues) => {
    void onSave(values)
    form.reset()
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        isEditMode ? t('Edit model rate limit') : t('Add model rate limit')
      }
      description={t(
        'Set the request and token rate limits for a specific model.',
      )}
      contentClassName='sm:max-w-[500px]'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button type='submit' form={DIALOG_FORM_ID} disabled={saving}>
            {isEditMode ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={DIALOG_FORM_ID}
          onSubmit={form.handleSubmit(handleSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='model'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Model')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t('e.g., gpt-4o, claude-3-opus')}
                    {...field}
                    disabled={isEditMode}
                  />
                </FormControl>
                <FormDescription>
                  {t('The model name requested by clients.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className='grid gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='rpm'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Requests / minute')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...field}
                      onChange={(e) =>
                        field.onChange(Number.parseInt(e.target.value) || 0)
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Max requests per minute, 0 = unlimited')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='tpm'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Tokens / minute')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...field}
                      onChange={(e) =>
                        field.onChange(Number.parseInt(e.target.value) || 0)
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Max tokens per minute, 0 = unlimited')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </form>
      </Form>
    </Dialog>
  )
}
