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
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import type {
  UnifiedModel,
  UnifiedModelChannelMember,
  UnifiedModelChannelOption,
} from './types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  initial: UnifiedModel | null
  channels: UnifiedModelChannelOption[]
  onSave: (model: UnifiedModel) => void
}

// _key 是仅用于 React 列表渲染的客户端标识,保存时剥离。
type EditableMember = UnifiedModelChannelMember & { _key: string }

let memberKeyCounter = 0
const nextMemberKey = () => `member-${++memberKeyCounter}`

const emptyMember = (): EditableMember => ({
  channel_id: 0,
  model_name: '',
  weight: 100,
  min_score: 0,
  enabled: true,
  _key: nextMemberKey(),
})

export function UnifiedModelEditorDialog(props: Props) {
  const { t } = useTranslation()
  const [id, setId] = useState(props.initial?.id ?? '')
  const [enabled, setEnabled] = useState(props.initial?.enabled ?? true)
  const [members, setMembers] = useState<EditableMember[]>(
    props.initial?.channels.map((member) => ({
      ...member,
      _key: nextMemberKey(),
    })) ?? []
  )
  const [error, setError] = useState('')

  const updateMember = (key: string, patch: Partial<UnifiedModelChannelMember>) => {
    setMembers((prev) =>
      prev.map((member) => (member._key === key ? { ...member, ...patch } : member))
    )
  }

  const moveMember = (index: number, direction: -1 | 1) => {
    setMembers((prev) => {
      const next = [...prev]
      const target = index + direction
      if (target < 0 || target >= next.length) return prev
      ;[next[index], next[target]] = [next[target], next[index]]
      return next
    })
  }

  const selectedChannel = (member: UnifiedModelChannelMember) =>
    props.channels.find((channel) => channel.id === member.channel_id)

  const handleSave = () => {
    const trimmedId = id.trim()
    if (!trimmedId) {
      setError(t('Unified model ID is required'))
      return
    }
    const enabledMembers = members.filter((member) => member.enabled)
    if (enabled && enabledMembers.length === 0) {
      setError(t('An enabled unified model needs at least one enabled member'))
      return
    }
    for (const member of members) {
      if (!member.channel_id || !member.model_name.trim()) {
        setError(t('Every member needs a channel and an upstream model name'))
        return
      }
    }
    setError('')
    props.onSave({
      id: trimmedId,
      enabled,
      channels: members.map(({ _key: _ignored, ...member }) => ({
        ...member,
        model_name: member.model_name.trim(),
        weight: member.weight > 0 ? member.weight : 100,
        min_score: Math.max(member.min_score, 0),
      })),
    })
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={
        props.initial
          ? t('Edit Unified Model')
          : t('Add Unified Model')
      }
      contentClassName='sm:max-w-2xl'
      contentHeight='auto'
      footer={
        <>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleSave}>{t('Save')}</Button>
        </>
      }
    >
      <div className='flex flex-col gap-4'>
        <div className='grid grid-cols-[1fr_auto] items-end gap-3'>
          <div className='flex flex-col gap-1.5'>
            <Label>{t('Unified Model ID')}</Label>
            <Input
              value={id}
              onChange={(event) => setId(event.target.value)}
              placeholder='auto'
              disabled={!!props.initial}
            />
          </div>
          <div className='flex items-center gap-2 pb-1'>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
            <Label>{t('Enabled')}</Label>
          </div>
        </div>
        {props.initial ? (
          <div className='text-muted-foreground text-xs'>
            {t('The unified model ID cannot be changed after creation')}
          </div>
        ) : null}

        <div className='flex flex-col gap-2'>
          <div className='flex items-center justify-between'>
            <Label>{t('Pool Members')}</Label>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setMembers((prev) => [...prev, emptyMember()])}
            >
              <Plus className='size-4' />
              {t('Add Member')}
            </Button>
          </div>
          {members.length === 0 ? (
            <div className='text-muted-foreground rounded-md border border-dashed p-4 text-center text-sm'>
              {t('No members yet. Add a channel and upstream model pair.')}
            </div>
          ) : (
            <div className='flex flex-col gap-2'>
              {members.map((member, index) => {
                const channel = selectedChannel(member)
                return (
                  <div
                    key={member._key}
                    className='flex flex-col gap-2 rounded-md border p-3'
                  >
                    <div className='grid grid-cols-1 gap-2 sm:grid-cols-2'>
                      <div className='flex flex-col gap-1'>
                        <Label className='text-xs'>{t('Channel')}</Label>
                        <Select
                          value={member.channel_id ? String(member.channel_id) : ''}
                          onValueChange={(value) =>
                            updateMember(member._key, {
                              channel_id: Number(value),
                              model_name: '',
                            })
                          }
                        >
                          <SelectTrigger size='sm'>
                            <SelectValue
                              placeholder={t('Select channel')}
                            />
                          </SelectTrigger>
                          <SelectContent>
                            {props.channels.map((option) => (
                              <SelectItem key={option.id} value={String(option.id)}>
                                {`#${option.id} ${option.name}`}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className='flex flex-col gap-1'>
                        <Label className='text-xs'>{t('Upstream Model')}</Label>
                        {channel && channel.models.length > 0 ? (
                          <Select
                            value={member.model_name}
                            onValueChange={(value) =>
                              updateMember(member._key, { model_name: value ?? '' })
                            }
                          >
                            <SelectTrigger size='sm'>
                              <SelectValue
                                placeholder={t('Select upstream model')}
                              />
                            </SelectTrigger>
                            <SelectContent>
                              {channel.models.map((modelName) => (
                                <SelectItem key={modelName} value={modelName}>
                                  {modelName}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <Input
                            value={member.model_name}
                            onChange={(event) =>
                              updateMember(member._key, { model_name: event.target.value })
                            }
                            placeholder='gpt-4o'
                          />
                        )}
                      </div>
                    </div>
                    <div className='grid grid-cols-2 gap-2'>
                      <div className='flex flex-col gap-1'>
                        <Label className='text-xs'>{t('Weight')}</Label>
                        <Input
                          type='number'
                          min={1}
                          value={member.weight}
                          onChange={(event) =>
                            updateMember(member._key, {
                              weight: Number(event.target.value),
                            })
                          }
                        />
                      </div>
                      <div className='flex flex-col gap-1'>
                        <Label className='text-xs'>
                          {t('Min Score (0-100)')}
                        </Label>
                        <Input
                          type='number'
                          min={0}
                          max={100}
                          value={member.min_score}
                          onChange={(event) =>
                            updateMember(member._key, {
                              min_score: Number(event.target.value),
                            })
                          }
                        />
                      </div>
                    </div>
                    <div className='flex items-center justify-between'>
                      <div className='flex items-center gap-2'>
                        <Switch
                          checked={member.enabled}
                          onCheckedChange={(checked) =>
                            updateMember(member._key, { enabled: checked })
                          }
                        />
                        <Label className='text-xs'>{t('Enabled')}</Label>
                      </div>
                      <div className='flex items-center gap-1'>
                        <Button
                          variant='ghost'
                          size='icon-sm'
                          disabled={index === 0}
                          onClick={() => moveMember(index, -1)}
                        >
                          <ArrowUp className='size-4' />
                        </Button>
                        <Button
                          variant='ghost'
                          size='icon-sm'
                          disabled={index === members.length - 1}
                          onClick={() => moveMember(index, 1)}
                        >
                          <ArrowDown className='size-4' />
                        </Button>
                        <Button
                          variant='ghost'
                          size='icon-sm'
                          onClick={() =>
                            setMembers((prev) =>
                              prev.filter((item) => item._key !== member._key)
                            )
                          }
                        >
                          <Trash2 className='size-4 text-destructive' />
                        </Button>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {error ? <div className='text-destructive text-sm'>{error}</div> : null}
      </div>
    </Dialog>
  )
}
