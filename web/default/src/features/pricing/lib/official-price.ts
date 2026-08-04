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
import type { OfficialPriceUnit, PricingModel } from '../types'

export type OfficialPriceSavings = {
  percent: string
  officialPrice: number
  unit: OfficialPriceUnit
  sourceModel?: string
  sourceUrl?: string
  verifiedAt?: string
}

type PricingTierWithConditions = {
  conditions?: Array<{
    var: string
    op: string
    value: number
  }>
}

export type OfficialTierThreshold = number | null

function getInputTokenUpperBound(
  tier: PricingTierWithConditions
): number | null {
  const conditions = Array.isArray(tier.conditions) ? tier.conditions : []
  if (conditions.length !== 1) return null
  const condition = conditions[0]
  if (
    condition.var !== 'len' ||
    (condition.op !== '<' && condition.op !== '<=') ||
    !Number.isFinite(condition.value) ||
    condition.value <= 0
  ) {
    return null
  }
  return condition.value
}

export function sortPricingTiersByInputThreshold<
  T extends PricingTierWithConditions,
>(tiers: readonly T[]): T[] {
  return tiers
    .map((tier, index) => ({
      tier,
      index,
      upperBound: getInputTokenUpperBound(tier),
    }))
    .sort((left, right) => {
      const leftBound = left.upperBound ?? Number.POSITIVE_INFINITY
      const rightBound = right.upperBound ?? Number.POSITIVE_INFINITY
      return leftBound - rightBound || left.index - right.index
    })
    .map(({ tier }) => tier)
}

export function getInputTokenTierThresholds(
  tiers: readonly PricingTierWithConditions[]
): OfficialTierThreshold[] | null {
  if (tiers.length === 0) return null
  const thresholds: OfficialTierThreshold[] = []
  for (let index = 0; index < tiers.length; index += 1) {
    const tier = tiers[index]
    const conditions = Array.isArray(tier.conditions) ? tier.conditions : []
    if (index === tiers.length - 1) {
      if (conditions.length !== 0) return null
      thresholds.push(null)
      continue
    }
    const upperBound = getInputTokenUpperBound(tier)
    if (upperBound === null) return null
    thresholds.push(upperBound)
  }
  return thresholds
}

function formatPercent(value: number): string {
  return Number(value.toFixed(1)).toString()
}

export function getOfficialPriceSavings(
  model: PricingModel,
  actualPrice: number,
  unit: OfficialPriceUnit,
  tierIndex: number,
  tierThresholds: readonly OfficialTierThreshold[] | null
): OfficialPriceSavings | null {
  const reference = model.official_price
  if (
    !reference ||
    reference.unit !== unit ||
    !Number.isFinite(actualPrice) ||
    actualPrice < 0 ||
    tierThresholds === null ||
    reference.tiers.length !== tierThresholds.length ||
    !reference.tiers.every(
      (tier, index) =>
        (tier.up_to_input_tokens ?? null) === tierThresholds[index]
    )
  ) {
    return null
  }

  const officialPrice = Number(reference.tiers[tierIndex]?.price)
  if (!Number.isFinite(officialPrice) || officialPrice <= 0) return null

  const deltaPercent = ((officialPrice - actualPrice) / officialPrice) * 100
  const roundedSavings = Math.round(deltaPercent * 10) / 10
  if (roundedSavings <= 0) return null

  return {
    percent: formatPercent(roundedSavings),
    officialPrice,
    unit,
    sourceModel: reference.source_model,
    sourceUrl: reference.source_url,
    verifiedAt: reference.verified_at,
  }
}

export function formatOfficialPrice(
  price: number,
  unit: OfficialPriceUnit
): string {
  const formatted = price.toLocaleString('en-US', {
    maximumFractionDigits: 8,
  })
  return unit === 'usd_per_request'
    ? `$${formatted} / request`
    : `$${formatted} / 1M input tokens`
}
