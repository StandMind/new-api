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
import type { OpenLuxPreview } from './types'

export function resolveOpenLuxSelection(
  preview: OpenLuxPreview,
  roots: Set<string>
) {
  const changeById = new Map(
    preview.changes.map((change) => [change.id, change] as const)
  )
  const selected = new Set<string>()
  const visit = (id: string) => {
    if (selected.has(id)) return
    const change = changeById.get(id)
    if (!change?.actionable) return
    selected.add(id)
    for (const dependency of change.requires ?? []) visit(dependency)
  }
  for (const id of roots) visit(id)
  return selected
}
