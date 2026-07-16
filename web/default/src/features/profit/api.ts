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
  ModelCostRule,
  ProfitOverview,
  ProfitQuery,
  SaveCostRuleInput,
} from './types'

type ApiEnvelope<T> = {
  success: boolean
  message?: string
  data: T
}

function unwrap<T>(envelope: ApiEnvelope<T>): T {
  if (!envelope.success) {
    throw new Error(envelope.message || 'Request failed')
  }
  return envelope.data
}

export async function getProfitOverview(
  query: ProfitQuery
): Promise<ProfitOverview> {
  const response = await api.get<ApiEnvelope<ProfitOverview>>(
    '/api/profit/overview',
    { params: query }
  )
  return unwrap(response.data)
}

export async function getModelCostRules(): Promise<ModelCostRule[]> {
  const response = await api.get<ApiEnvelope<ModelCostRule[]>>(
    '/api/profit/cost-rules'
  )
  return unwrap(response.data)
}

export async function getProfitCostModels(): Promise<string[]> {
  const response = await api.get<ApiEnvelope<string[]>>(
    '/api/profit/cost-models'
  )
  return unwrap(response.data)
}

export async function saveModelCostRule(
  input: SaveCostRuleInput
): Promise<ModelCostRule> {
  const response = await api.post<ApiEnvelope<ModelCostRule>>(
    '/api/profit/cost-rules',
    input
  )
  return unwrap(response.data)
}

export async function backfillProfitRecords(): Promise<void> {
  const response = await api.post<ApiEnvelope<null>>('/api/profit/backfill')
  unwrap(response.data)
}

export async function resetProfitAnalysisData(): Promise<void> {
  const response =
    await api.post<ApiEnvelope<{ reset_at: number }>>('/api/profit/reset')
  unwrap(response.data)
}
