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
export function hasDuplicateCreemQuota(products: unknown[]): boolean {
  const quotas = new Set<number>()
  for (const product of products) {
    if (!product || typeof product !== 'object' || !('quota' in product)) {
      continue
    }
    const quota = Number(product.quota)
    if (!Number.isFinite(quota)) {
      continue
    }
    if (quotas.has(quota)) {
      return true
    }
    quotas.add(quota)
  }
  return false
}
