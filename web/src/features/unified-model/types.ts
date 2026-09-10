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
export interface UnifiedModelChannelMember {
  channel_id: number
  model_name: string
  weight: number
  min_score: number
  enabled: boolean
}

export interface UnifiedModel {
  id: string
  enabled: boolean
  channels: UnifiedModelChannelMember[]
}

export interface UnifiedModelHealth {
  channel_id: number
  model_name: string
  enabled: boolean
  group: string
  request_count: number
  success_rate: number
  avg_latency_ms: number
  max_latency_ms: number
  avg_tps: number
  score: number
  weighted_score: number
}

export interface UnifiedModelWithHealth extends UnifiedModel {
  health: UnifiedModelHealth[]
  member_count: number
}

export interface UnifiedModelSettings {
  models: UnifiedModelWithHealth[]
  score_window_hours: number
  score_cache_seconds: number
}

export interface UnifiedModelChannelOption {
  id: number
  name: string
  type: number
  status: number
  models: string[]
  groups: string[]
}

export interface UnifiedModelSettingsPayload {
  models: UnifiedModel[]
  score_window_hours: number
  score_cache_seconds: number
}
