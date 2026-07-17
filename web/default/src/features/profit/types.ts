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
export type ProfitAggregate = {
  user_id?: number
  username?: string
  model_name?: string
  channel_id?: number
  channel_name?: string
  group?: string
  revenue_micros: number
  known_revenue_micros: number
  gross_consumption_micros?: number
  nominal_consumption_micros?: number
  recognized_revenue_micros?: number
  promo_consumption_micros?: number
  promo_cost_micros?: number
  admin_consumption_micros?: number
  admin_cost_micros?: number
  promo_unpriced_record_count?: number
  admin_unpriced_record_count?: number
  legacy_consumption_micros?: number
  legacy_unknown_consumption_micros?: number
  cost_micros: number
  profit_micros: number
  record_count: number
  unpriced_record_count: number
  profit_margin: number | null
  cost_coverage: number
}

export type ProfitOverview = {
  summary: ProfitAggregate
  by_user: ProfitAggregate[]
  by_model: ProfitAggregate[]
  by_group: ProfitAggregate[]
  by_channel: ProfitAggregate[]
}

export type ModelCostRule = {
  id: number
  model_name: string
  purchase_price_cny: number
  version: number
  enabled: boolean
  effective_from: number
  effective_to: number
}

export type ProfitCostModelGroup = {
  group: string
  models: string[]
}

export type ProfitQuery = {
  start_timestamp?: number
  end_timestamp?: number
  user_id?: number
  model?: string
  group?: string
  channel?: number
}

export type SaveCostRuleInput = Pick<
  ModelCostRule,
  'model_name' | 'purchase_price_cny'
>
