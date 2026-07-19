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

export function allocateWalletBalance(
  current: WalletAllocationPayload,
  finalPaidQuota: number
): WalletAllocationPayload | null {
  const totalQuota =
    current.paid_quota + current.promo_quota + current.legacy_quota
  if (
    !Number.isInteger(finalPaidQuota) ||
    !Number.isInteger(totalQuota) ||
    finalPaidQuota < 0 ||
    finalPaidQuota > totalQuota
  ) {
    return null
  }

  return {
    paid_quota: finalPaidQuota,
    promo_quota: totalQuota - finalPaidQuota,
    legacy_quota: 0,
  }
}
