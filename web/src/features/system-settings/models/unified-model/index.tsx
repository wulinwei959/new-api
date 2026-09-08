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
import { Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StaticDataTable } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { SettingsSection } from '../../components/settings-section'
import {
  getUnifiedModelChannels,
  getUnifiedModelSettings,
  updateUnifiedModelSettings,
} from './api'
import { UnifiedModelEditorDialog } from './editor-dialog'
import type {
  UnifiedModel,
  UnifiedModelChannelOption,
  UnifiedModelSettings as UnifiedModelSettingsData,
} from './types'

const HEALTH_REFRESH_MS = 30_000

type UnifiedModelRow = UnifiedModelSettingsData['models'][number]

// 池聚合指标取成员按请求量加权的均值,与选择器的实际流量分布近似。
function aggregateHealth(row: UnifiedModelRow) {
  const withData = row.health.filter((item) => item.request_count > 0)
  if (withData.length === 0) {
    return { latency: 0, successRate: 0, tps: 0, hasData: false }
  }
  const totalRequests = withData.reduce(
    (sum, item) => sum + item.request_count,
    0
  )
  const latency = Math.round(
    withData.reduce(
      (sum, item) => sum + item.avg_latency_ms * item.request_count,
      0
    ) / totalRequests
  )
  const successRate = Number(
    (
      withData.reduce(
        (sum, item) => sum + item.success_rate * item.request_count,
        0
      ) / totalRequests
    ).toFixed(2)
  )
  const tps = Number(
    (
      withData.reduce(
        (sum, item) => sum + item.avg_tps * item.request_count,
        0
      ) / totalRequests
    ).toFixed(2)
  )
  return { latency, successRate, tps, hasData: true }
}

function latencyClass(latencyMs: number) {
  if (latencyMs < 1500) return 'text-emerald-600'
  if (latencyMs < 5000) return 'text-amber-600'
  return 'text-red-600'
}

function successClass(rate: number) {
  if (rate >= 95) return 'text-emerald-600'
  if (rate >= 80) return 'text-amber-600'
  return 'text-red-600'
}

export function UnifiedModelSection() {
  const { t } = useTranslation()
  const [settings, setSettings] = useState<UnifiedModelSettingsData | null>(
    null
  )
  const [channels, setChannels] = useState<UnifiedModelChannelOption[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [editorOpen, setEditorOpen] = useState(false)
  const [editingModel, setEditingModel] = useState<UnifiedModel | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      const [settingsRes, channelsRes] = await Promise.all([
        getUnifiedModelSettings(),
        getUnifiedModelChannels(),
      ])
      setSettings(settingsRes.data)
      setChannels(channelsRes.data ?? [])
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
    const timer = setInterval(() => void load(), HEALTH_REFRESH_MS)
    return () => clearInterval(timer)
  }, [load])

  const persist = useCallback(
    async (models: UnifiedModel[]) => {
      if (!settings) return
      setSaving(true)
      try {
        await updateUnifiedModelSettings({
          models,
          score_window_hours: settings.score_window_hours,
          score_cache_seconds: settings.score_cache_seconds,
        })
        toast.success(t('Unified model settings saved'))
        await load()
      } catch (err) {
        toast.error(err instanceof Error ? err.message : String(err))
      } finally {
        setSaving(false)
      }
    },
    [load, settings, t]
  )

  const handleToggle = (row: UnifiedModelRow, checked: boolean) => {
    if (!settings) return
    const models = settings.models.map((model) =>
      model.id === row.id ? { ...model, enabled: checked } : model
    )
    void persist(models)
  }

  const handleDelete = () => {
    if (!settings || !deletingId) return
    const models = settings.models
      .filter((model) => model.id !== deletingId)
      .map(({ health: _health, member_count: _count, ...rest }) => rest)
    setDeletingId(null)
    void persist(models)
  }

  const columns = [
    {
      id: 'id',
      header: t('Unified Model ID'),
      cell: (row: UnifiedModelRow) => (
        <div className='flex items-center gap-2'>
          <span className='font-medium'>{row.id}</span>
          {!row.enabled ? (
            <Badge variant='secondary'>{t('Disabled')}</Badge>
          ) : null}
        </div>
      ),
    },
    {
      id: 'members',
      header: t('Members'),
      cell: (row: UnifiedModelRow) => (
        <Badge variant='outline'>{row.member_count}</Badge>
      ),
    },
    {
      id: 'latency',
      header: t('Avg Latency'),
      cell: (row: UnifiedModelRow) => {
        const { latency, hasData } = aggregateHealth(row)
        if (!hasData) return <span className='text-muted-foreground'>—</span>
        return (
          <span className={latencyClass(latency)}>{latency} ms</span>
        )
      },
    },
    {
      id: 'success',
      header: t('Success Rate'),
      cell: (row: UnifiedModelRow) => {
        const { successRate, hasData } = aggregateHealth(row)
        if (!hasData) return <span className='text-muted-foreground'>—</span>
        return (
          <span className={successClass(successRate)}>{successRate}%</span>
        )
      },
    },
    {
      id: 'tps',
      header: t('Avg TPS'),
      cell: (row: UnifiedModelRow) => {
        const { tps, hasData } = aggregateHealth(row)
        if (!hasData) return <span className='text-muted-foreground'>—</span>
        return <span>{tps}</span>
      },
    },
    {
      id: 'actions',
      header: t('Actions'),
      cell: (row: UnifiedModelRow) => (
        <div className='flex items-center gap-1'>
          <Switch
            checked={row.enabled}
            onCheckedChange={(checked) => handleToggle(row, checked)}
            disabled={saving}
          />
          <Button
            variant='ghost'
            size='icon-sm'
            onClick={() => {
              setEditingModel({
                id: row.id,
                enabled: row.enabled,
                channels: row.channels,
              })
              setEditorOpen(true)
            }}
          >
            <Pencil className='size-4' />
          </Button>
          <Button
            variant='ghost'
            size='icon-sm'
            onClick={() => setDeletingId(row.id)}
          >
            <Trash2 className='size-4 text-destructive' />
          </Button>
        </div>
      ),
    },
  ]

  return (
    <SettingsSection title={t('Unified Models')}>
      <div className='flex flex-col gap-4'>
        {error ? (
          <Alert variant='destructive'>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
        <div className='flex items-center justify-between'>
          <div className='flex items-center gap-2'>
            <RefreshCw className='text-muted-foreground size-4' />
            <Label className='text-muted-foreground text-xs'>
              {t('Health metrics refresh every 30 seconds')}
            </Label>
          </div>
          <Button
            size='sm'
            onClick={() => {
              setEditingModel(null)
              setEditorOpen(true)
            }}
          >
            <Plus className='size-4' />
            {t('Add Unified Model')}
          </Button>
        </div>

        {loading ? (
          <div className='text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm'>
            {t('Loading...')}
          </div>
        ) : (
          <StaticDataTable
            columns={columns}
            data={settings?.models ?? []}
            getRowKey={(row) => row.id}
            emptyContent={
              <div className='text-muted-foreground p-6 text-center text-sm'>
                {t(
                  'No unified models configured. Create one and point clients at its ID, e.g. "auto".'
                )}
              </div>
            }
          />
        )}
      </div>

      <UnifiedModelEditorDialog
        open={editorOpen}
        onOpenChange={setEditorOpen}
        initial={editingModel}
        channels={channels}
        onSave={async (model) => {
          if (!settings) return
          setEditorOpen(false)
          const exists = settings.models.some((item) => item.id === model.id)
          const models = exists
            ? settings.models.map((item) =>
                item.id === model.id ? model : item
              )
            : [...settings.models, model]
          await persist(models)
        }}
      />

      {deletingId ? (
        <Dialog
          open
          onOpenChange={(open) => !open && setDeletingId(null)}
          title={t('Delete Unified Model')}
          contentClassName='sm:max-w-md'
          contentHeight='auto'
          footer={
            <>
              <Button variant='outline' onClick={() => setDeletingId(null)}>
                {t('Cancel')}
              </Button>
              <Button variant='destructive' onClick={handleDelete}>
                {t('Delete')}
              </Button>
            </>
          }
        >
          <div className='text-muted-foreground text-sm'>
            {t(
              'Clients calling this unified model id will stop being routed. This cannot be undone.'
            )}
          </div>
        </Dialog>
      ) : null}
    </SettingsSection>
  )
}
