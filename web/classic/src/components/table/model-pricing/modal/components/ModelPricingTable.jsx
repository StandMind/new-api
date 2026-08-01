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
import { Avatar, Table, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import { IconCoinMoneyStroked } from '@douyinfe/semi-icons';

import { BILLING_PRICING_VARS } from '../../../../../constants';
import {
  calculateModelPrice,
  parseTiersFromExpr,
} from '../../../../../helpers';
import { splitBillingExprAndRequestRules } from '../../../../../pages/Setting/Ratio/components/requestRuleExpr';
import OfficialPriceTag, {
  getInputTokenTierThresholds,
  sortPricingTiersByInputThreshold,
} from './OfficialPriceTag';

const { Text } = Typography;

function formatTokenHint(value, compact = true) {
  const number = Number(value);
  if (!Number.isFinite(number) || number <= 0) return String(value ?? '');
  if (!compact) {
    return number.toLocaleString(undefined, { maximumFractionDigits: 20 });
  }
  if (number >= 1000000) {
    return `${Number((number / 1000000).toFixed(1))}M`;
  }
  if (number >= 1000) {
    return `${Number((number / 1000).toFixed(1))}K`;
  }
  return String(number);
}

function getInputTokenTierRanges(tiers) {
  if (!Array.isArray(tiers) || tiers.length === 0) return null;

  const ranges = [];
  let previousUpperBound = null;
  let previousUpperInclusive = false;

  for (let index = 0; index < tiers.length; index += 1) {
    const conditions = Array.isArray(tiers[index]?.conditions)
      ? tiers[index].conditions
      : [];
    const isLastTier = index === tiers.length - 1;

    if (isLastTier) {
      if (conditions.length !== 0) return null;
      ranges.push({
        lowerBound: previousUpperBound,
        lowerInclusive: previousUpperBound !== null && !previousUpperInclusive,
        upperBound: null,
        upperInclusive: false,
      });
      continue;
    }

    if (conditions.length !== 1) return null;
    const condition = conditions[0];
    if (
      condition.var !== 'len' ||
      !['<', '<='].includes(condition.op) ||
      !Number.isFinite(condition.value) ||
      condition.value <= 0 ||
      (previousUpperBound !== null && condition.value <= previousUpperBound)
    ) {
      return null;
    }

    ranges.push({
      lowerBound: previousUpperBound,
      lowerInclusive: previousUpperBound !== null && !previousUpperInclusive,
      upperBound: condition.value,
      upperInclusive: condition.op === '<=',
    });
    previousUpperBound = condition.value;
    previousUpperInclusive = condition.op === '<=';
  }

  return ranges;
}

function formatInputTokenTierRange(range, compact) {
  const lowerOperator = range.lowerInclusive ? '≥' : '>';
  const upperOperator = range.upperInclusive ? '≤' : '<';
  const formatValue = (value) => formatTokenHint(value, compact);

  if (range.lowerBound === null && range.upperBound !== null) {
    return `${upperOperator} ${formatValue(range.upperBound)}`;
  }
  if (range.lowerBound !== null && range.upperBound === null) {
    return `${lowerOperator} ${formatValue(range.lowerBound)}`;
  }
  if (range.lowerBound !== null && range.upperBound !== null) {
    return `${lowerOperator} ${formatValue(range.lowerBound)} · ${upperOperator} ${formatValue(range.upperBound)}`;
  }
  return '-';
}

function formatTierCondition(conditions, t) {
  const variableLabels = { p: t('输入'), c: t('补全'), len: t('长度') };
  return (conditions || [])
    .map((condition) => {
      const label = variableLabels[condition.var] || condition.var;
      return `${label} ${condition.op} ${formatTokenHint(condition.value)}`;
    })
    .join(' && ');
}

function formatTierPrice(value, ratio, tokenUnit, displayPrice) {
  const divisor = tokenUnit === 'K' ? 1000 : 1;
  const precision = tokenUnit === 'K' ? 6 : 4;
  const formatted = displayPrice(Number(value) * ratio);
  const numeric = Number.parseFloat(formatted.replace(/[^0-9.-]/g, ''));
  const symbolEnd = formatted.search(/[0-9-]/);
  const symbol = symbolEnd > 0 ? formatted.slice(0, symbolEnd) : '$';
  if (!Number.isFinite(numeric)) return '-';
  return `${symbol}${(numeric / divisor).toFixed(precision)}`;
}

function regularPriceFields(modelData, priceData, siteDisplayType, t) {
  if (modelData.quota_type === 1) {
    return [{ key: 'price', label: t('价格'), value: priceData.price }];
  }

  if (siteDisplayType === 'TOKENS' || priceData.isTokensDisplay) {
    const candidates = [
      ['input', '输入倍率', priceData.inputRatio],
      ['output', '补全倍率', priceData.completionRatio],
      ['cache', '缓存读取倍率', priceData.cacheRatio],
      ['cache-write', '缓存创建倍率', priceData.createCacheRatio],
      ['image', '图片输入倍率', priceData.imageRatio],
      ['audio-input', '音频输入倍率', priceData.audioInputRatio],
      ['audio-output', '音频补全倍率', priceData.audioOutputRatio],
    ];
    return candidates
      .filter(([, , value]) => value !== null && value !== undefined)
      .map(([key, label, value]) => ({
        key,
        label: t(label),
        value: `${value}x`,
      }));
  }

  const candidates = [
    ['input', '输入', priceData.inputPrice],
    ['output', '补全', priceData.completionPrice],
    ['cache', '缓存读', priceData.cachePrice],
    ['cache-write', '缓存创建', priceData.createCachePrice],
    ['image', '图片输入', priceData.imagePrice],
    ['audio-input', '音频输入', priceData.audioInputPrice],
    ['audio-output', '音频输出', priceData.audioOutputPrice],
  ];
  return candidates
    .filter(([, , value]) => value !== null && value !== undefined)
    .map(([key, label, value]) => ({ key, label: t(label), value }));
}

const ModelPricingTable = ({
  modelData,
  groupRatio,
  currency,
  siteDisplayType,
  tokenUnit,
  displayPrice,
  usableGroup,
  t,
}) => {
  const [hoveredPricingGroup, setHoveredPricingGroup] = React.useState(null);
  const modelEnableGroups = Array.isArray(modelData?.enable_groups)
    ? modelData.enable_groups
    : [];
  const effectiveGroupRatio = {
    ...(groupRatio || {}),
    ...(modelData?.group_ratio || {}),
  };
  const getEffectiveGroupRatio = (group) => {
    const ratio = effectiveGroupRatio[group];
    return typeof ratio === 'number' && Number.isFinite(ratio) ? ratio : 1;
  };
  const availableGroups = Object.keys(usableGroup || {})
    .filter((group) => group !== '' && modelEnableGroups.includes(group))
    .sort((left, right) => {
      const ratioDelta =
        getEffectiveGroupRatio(left) - getEffectiveGroupRatio(right);
      return ratioDelta || left.localeCompare(right);
    });

  const isTiered =
    modelData?.billing_mode === 'tiered_expr' && modelData?.billing_expr;
  let tiers = [];
  if (isTiered) {
    const { billingExpr } = splitBillingExprAndRequestRules(
      modelData.billing_expr,
    );
    tiers = sortPricingTiersByInputThreshold(parseTiersFromExpr(billingExpr));
  }
  const inputTokenTierRanges =
    isTiered && tiers.length > 1 ? getInputTokenTierRanges(tiers) : null;
  const usesInputTokenTiers = inputTokenTierRanges !== null;

  if (isTiered && tiers.length === 0) {
    return (
      <div>
        <div className='flex items-center mb-4'>
          <Avatar size='small' color='orange' className='mr-2 shadow-md'>
            <IconCoinMoneyStroked size={16} />
          </Avatar>
          <Text className='text-lg font-medium'>{t('分组价格')}</Text>
        </div>
        <div className='rounded-lg border border-amber-200 bg-amber-50 p-3'>
          <Text strong>{t('特殊计费表达式')}</Text>
          <div className='text-xs text-gray-500 mt-1'>
            {t('该表达式无法展开为标准分档价格，请核对原始表达式。')}
          </div>
          <code className='block text-xs text-gray-600 break-all mt-3'>
            {modelData.billing_expr}
          </code>
        </div>
      </div>
    );
  }

  const rows = [];
  let activeFields = [];
  const officialTierThresholds = isTiered
    ? getInputTokenTierThresholds(tiers)
    : [null];
  if (isTiered) {
    activeFields = BILLING_PRICING_VARS.filter((variable) =>
      tiers.some((tier) => Number(tier[variable.field]) > 0),
    );
    availableGroups.forEach((group) => {
      const ratio = getEffectiveGroupRatio(group);
      tiers.forEach((tier, tierIndex) => {
        rows.push({
          key: `${group}-${tierIndex}`,
          group,
          ratio,
          tier,
          tierIndex,
          officialUnit: 'usd_per_million_input_tokens',
          officialActualPrice: Number(tier.inputPrice) * ratio,
        });
      });
    });
  } else {
    availableGroups.forEach((group) => {
      const ratio = getEffectiveGroupRatio(group);
      const priceData = calculateModelPrice({
        record: modelData,
        selectedGroup: group,
        groupRatio: effectiveGroupRatio,
        tokenUnit,
        displayPrice,
        currency,
        quotaDisplayType: siteDisplayType,
      });
      rows.push({
        key: group,
        group,
        ratio,
        priceData,
        fields: regularPriceFields(modelData, priceData, siteDisplayType, t),
        officialUnit:
          modelData.quota_type === 1
            ? 'usd_per_request'
            : 'usd_per_million_input_tokens',
        officialActualPrice:
          modelData.quota_type === 1
            ? Number(modelData.model_price) * ratio
            : Number(modelData.model_ratio) * 2 * ratio,
      });
    });
    activeFields = rows[0]?.fields || [];
  }

  const getGroupCellRowSpan = (row) => {
    if (!isTiered || tiers.length <= 1) return 1;
    return row.tierIndex === 0 ? tiers.length : 0;
  };

  const groupColumn = {
    title: t('分组'),
    dataIndex: 'group',
    width: 230,
    render: (group, row) => {
      const rowSpan = getGroupCellRowSpan(row);
      return {
        children:
          rowSpan === 0 ? null : (
            <div className='flex flex-wrap items-center gap-1.5 min-w-[200px]'>
              <Tag color='white' size='small'>
                {group}
              </Tag>
              <OfficialPriceTag
                modelData={modelData}
                actualPrice={row.officialActualPrice}
                unit={row.officialUnit}
                tierIndex={row.tierIndex ?? 0}
                tierThresholds={officialTierThresholds}
                t={t}
              />
            </div>
          ),
        props: {
          rowSpan,
          style: rowSpan === 0 ? { display: 'none' } : undefined,
        },
      };
    },
  };
  const ratioColumn = {
    title: t('分组倍率'),
    dataIndex: 'ratio',
    width: 90,
    render: (ratio, row) => {
      const rowSpan = getGroupCellRowSpan(row);
      return {
        children:
          rowSpan === 0 ? null : (
            <Tag color='blue' size='small'>
              {ratio}x
            </Tag>
          ),
        props: {
          rowSpan,
          style: rowSpan === 0 ? { display: 'none' } : undefined,
        },
      };
    },
  };

  const columns = [groupColumn, ratioColumn];
  if (isTiered && tiers.length > 1) {
    columns.push({
      title: usesInputTokenTiers ? t('单次请求输入 Token') : t('档位'),
      dataIndex: 'tier',
      width: usesInputTokenTiers ? 190 : 150,
      render: (tier, row) => {
        const range = inputTokenTierRanges?.[row.tierIndex];
        if (!range) {
          return (
            <div className='min-w-[120px]'>
              <div>{tier.label || t('默认')}</div>
              {formatTierCondition(tier.conditions, t) && (
                <div className='text-xs text-gray-500 mt-1'>
                  {formatTierCondition(tier.conditions, t)}
                </div>
              )}
            </div>
          );
        }

        return (
          <div className='min-w-[150px]'>
            <Tooltip
              content={`${t('输入 Token')} ${formatInputTokenTierRange(range, false)}`}
              position='top'
            >
              <span className='inline-flex cursor-help font-medium tabular-nums'>
                {formatInputTokenTierRange(range, true)}
              </span>
            </Tooltip>
            <div className='text-xs text-gray-500 mt-1'>
              {tier.label || t('默认')}
            </div>
          </div>
        );
      },
    });
  }

  if (isTiered) {
    activeFields.forEach((variable) => {
      columns.push({
        title: `${t(variable.shortLabel)} / ${tokenUnit === 'K' ? '1K' : '1M'} tokens`,
        dataIndex: variable.field,
        width: 145,
        render: (_value, row) => {
          const value = Number(row.tier[variable.field]);
          return value > 0 ? (
            <Text strong>
              {formatTierPrice(value, row.ratio, tokenUnit, displayPrice)}
            </Text>
          ) : (
            '-'
          );
        },
      });
    });
  } else {
    activeFields.forEach((field, fieldIndex) => {
      columns.push({
        title: field.label,
        dataIndex: field.key,
        width: 135,
        render: (_value, row) => (
          <Text strong>{row.fields[fieldIndex]?.value ?? '-'}</Text>
        ),
      });
    });
  }

  return (
    <div>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='orange' className='mr-2 shadow-md'>
          <IconCoinMoneyStroked size={16} />
        </Avatar>
        <div>
          <Text className='text-lg font-medium'>{t('分组价格')}</Text>
          <div className='text-xs text-gray-600'>
            {usesInputTokenTiers
              ? t(
                  '价格档位根据单次请求的完整输入上下文 Token 数确定，不按账户累计用量计算。',
                )
              : t('不同用户分组的价格信息')}
          </div>
        </div>
      </div>
      <div
        className='overflow-x-auto max-w-full'
        onMouseOver={(event) => {
          const row = event.target?.closest?.('tr[data-pricing-group]');
          setHoveredPricingGroup(row?.dataset.pricingGroup ?? null);
        }}
        onMouseLeave={() => setHoveredPricingGroup(null)}
      >
        <Table
          dataSource={rows}
          columns={columns}
          onRow={(row) => ({
            'data-pricing-group': row.group,
            className:
              hoveredPricingGroup === row.group
                ? 'semi-table-row-hovered'
                : undefined,
          })}
          pagination={false}
          size='small'
          bordered={false}
          scroll={{ x: 'max-content' }}
          className='!rounded-lg min-w-max'
        />
      </div>
    </div>
  );
};

export default ModelPricingTable;
