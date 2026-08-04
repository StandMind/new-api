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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  formatOfficialPrice,
  getOfficialPriceSavings,
  type OfficialTierThreshold,
} from '../lib/official-price'
import type { OfficialPriceUnit, PricingModel } from '../types'

type OfficialPriceBadgeProps = {
  model: PricingModel
  actualPrice: number
  unit: OfficialPriceUnit
  tierIndex?: number
  tierThresholds?: readonly OfficialTierThreshold[] | null
}

export function OfficialPriceBadge(props: OfficialPriceBadgeProps) {
  const { t } = useTranslation()
  const comparison = getOfficialPriceSavings(
    props.model,
    props.actualPrice,
    props.unit,
    props.tierIndex ?? 0,
    props.tierThresholds === undefined ? [null] : props.tierThresholds
  )
  if (!comparison) return null

  const label = t('Cheaper than official by {{percent}}%', {
    percent: comparison.percent,
  })

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Badge
            variant='outline'
            className='max-w-full shrink-0 border-emerald-200 bg-emerald-50 text-[10px] font-normal whitespace-nowrap text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-300'
          />
        }
      >
        {label}
      </TooltipTrigger>
      <TooltipContent className='max-w-80 space-y-1'>
        <div>
          {t('Official reference: {{price}}', {
            price: formatOfficialPrice(
              comparison.officialPrice,
              comparison.unit
            ),
          })}
        </div>
        {comparison.sourceModel && (
          <div>
            {t('Source model: {{model}}', {
              model: comparison.sourceModel,
            })}
          </div>
        )}
        {comparison.verifiedAt && (
          <div>
            {t('Verified on: {{date}}', { date: comparison.verifiedAt })}
          </div>
        )}
      </TooltipContent>
    </Tooltip>
  )
}
