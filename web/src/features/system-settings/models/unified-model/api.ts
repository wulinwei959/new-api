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
import { api } from '@/lib/api'

import type {
  UnifiedModelChannelOption,
  UnifiedModelSettings,
  UnifiedModelSettingsPayload,
} from './types'

interface ApiEnvelope<T> {
  success: boolean
  message: string
  data: T
}

export async function getUnifiedModelSettings() {
  const res = await api.get<ApiEnvelope<UnifiedModelSettings>>(
    '/api/unified_model/'
  )
  return res.data
}

export async function updateUnifiedModelSettings(
  payload: UnifiedModelSettingsPayload
) {
  const res = await api.put<ApiEnvelope<null>>('/api/unified_model/', payload)
  return res.data
}

export async function getUnifiedModelChannels() {
  const res = await api.get<ApiEnvelope<UnifiedModelChannelOption[]>>(
    '/api/unified_model/channels'
  )
  return res.data
}
