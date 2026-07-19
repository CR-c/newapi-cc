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
import type { WalletAllocationPayload } from '../types'

export function attributeLegacyWallet(
  current: WalletAllocationPayload,
  legacyPaidQuota: number
): WalletAllocationPayload | null {
  if (
    !Number.isInteger(legacyPaidQuota) ||
    legacyPaidQuota < 0 ||
    legacyPaidQuota > current.legacy_quota
  ) {
    return null
  }

  return {
    paid_quota: current.paid_quota + legacyPaidQuota,
    promo_quota: current.promo_quota + current.legacy_quota - legacyPaidQuota,
    legacy_quota: 0,
  }
}
