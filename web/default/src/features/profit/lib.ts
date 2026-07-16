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
const MICROS_PER_UNIT = 1_000_000

export function formatProfitMoney(micros: number): string {
  const amount = micros / MICROS_PER_UNIT
  const sign = amount < 0 ? '-' : ''
  const formatted = new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(Math.abs(amount))
  return `${sign}¥ ${formatted}`
}

export function formatProfitPercent(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '--'
  return new Intl.NumberFormat(undefined, {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 2,
  }).format(value)
}

export function getProfitTone(micros: number): string {
  if (micros > 0) return 'text-emerald-600 dark:text-emerald-400'
  if (micros < 0) return 'text-rose-600 dark:text-rose-400'
  return 'text-muted-foreground'
}
