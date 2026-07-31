/*
Copyright (C) 2025 QuantumNous

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

import React from 'react';
import { Tag, Tooltip } from '@douyinfe/semi-ui';

const TOKEN_UNIT = 'usd_per_million_input_tokens';
const REQUEST_UNIT = 'usd_per_request';

function formatPercent(value) {
  return Number(value.toFixed(1)).toString();
}

function formatOfficialPrice(price, unit) {
  const formatted = Number(price).toLocaleString('en-US', {
    maximumFractionDigits: 8,
  });
  return unit === REQUEST_UNIT
    ? `$${formatted} / request`
    : `$${formatted} / 1M input tokens`;
}

function getInputTokenUpperBound(tier) {
  const conditions = Array.isArray(tier?.conditions) ? tier.conditions : [];
  if (conditions.length !== 1) return null;
  const condition = conditions[0];
  if (
    condition.var !== 'len' ||
    !['<', '<='].includes(condition.op) ||
    !Number.isFinite(condition.value) ||
    condition.value <= 0
  ) {
    return null;
  }
  return condition.value;
}

export function sortPricingTiersByInputThreshold(tiers) {
  return tiers
    .map((tier, index) => ({
      tier,
      index,
      upperBound: getInputTokenUpperBound(tier),
    }))
    .sort((left, right) => {
      const leftBound = left.upperBound ?? Number.POSITIVE_INFINITY;
      const rightBound = right.upperBound ?? Number.POSITIVE_INFINITY;
      return leftBound - rightBound || left.index - right.index;
    })
    .map(({ tier }) => tier);
}

export function getInputTokenTierThresholds(tiers) {
  if (!Array.isArray(tiers) || tiers.length === 0) return null;
  const thresholds = [];
  for (let index = 0; index < tiers.length; index += 1) {
    const conditions = Array.isArray(tiers[index]?.conditions)
      ? tiers[index].conditions
      : [];
    if (index === tiers.length - 1) {
      if (conditions.length !== 0) return null;
      thresholds.push(null);
      continue;
    }
    const upperBound = getInputTokenUpperBound(tiers[index]);
    if (upperBound === null) return null;
    thresholds.push(upperBound);
  }
  return thresholds;
}

export default function OfficialPriceTag({
  modelData,
  actualPrice,
  unit,
  tierIndex = 0,
  tierThresholds = [null],
  t,
}) {
  const reference = modelData?.official_price;
  if (
    !reference ||
    ![TOKEN_UNIT, REQUEST_UNIT].includes(unit) ||
    reference.unit !== unit ||
    !Array.isArray(reference.tiers) ||
    tierThresholds === null ||
    reference.tiers.length !== tierThresholds.length ||
    !reference.tiers.every(
      (tier, index) =>
        (tier.up_to_input_tokens ?? null) === tierThresholds[index],
    ) ||
    !Number.isFinite(actualPrice) ||
    actualPrice < 0
  ) {
    return null;
  }

  const officialPrice = Number(reference.tiers[tierIndex]?.price);
  if (!Number.isFinite(officialPrice) || officialPrice <= 0) return null;

  const deltaPercent = ((officialPrice - actualPrice) / officialPrice) * 100;
  const magnitude = Math.round(Math.abs(deltaPercent) * 10) / 10;
  let color = 'grey';
  let label = t('与官方同价');
  if (magnitude > 0 && deltaPercent > 0) {
    color = 'green';
    label = t('比官方便宜 {{percent}}%', {
      percent: formatPercent(magnitude),
    });
  } else if (magnitude > 0) {
    color = 'red';
    label = t('高于官方 {{percent}}%', {
      percent: formatPercent(magnitude),
    });
  }

  const tooltip = (
    <div className='space-y-1 max-w-xs'>
      <div>
        {t('官方参考价：{{price}}', {
          price: formatOfficialPrice(officialPrice, unit),
        })}
      </div>
      {reference.source_model && (
        <div>{t('来源型号：{{model}}', { model: reference.source_model })}</div>
      )}
      {reference.verified_at && (
        <div>{t('核验日期：{{date}}', { date: reference.verified_at })}</div>
      )}
      {reference.source_url && (
        <a
          href={reference.source_url}
          target='_blank'
          rel='noreferrer'
          className='underline'
          onClick={(event) => event.stopPropagation()}
        >
          {t('查看价格来源')}
        </a>
      )}
    </div>
  );

  return (
    <Tooltip content={tooltip} position='top'>
      <Tag color={color} size='small' className='whitespace-nowrap'>
        {label}
      </Tag>
    </Tooltip>
  );
}
