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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Select,
  Table,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconPlus, IconSave } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

import { API, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;
const TOKEN_UNIT = 'usd_per_million_input_tokens';
const REQUEST_UNIT = 'usd_per_request';

function parsePrices(value) {
  try {
    const parsed = JSON.parse(value || '{}');
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : {};
  } catch {
    return {};
  }
}

function createDraft(model = '', price) {
  return {
    originalModel: price ? model : '',
    model,
    unit: price?.unit || TOKEN_UNIT,
    source_model: price?.source_model || model,
    source_url: price?.source_url || '',
    verified_at: price?.verified_at || '',
    notes: price?.notes || '',
    tiers: (price?.tiers || [{ price: '' }]).map((tier) => ({
      up_to_input_tokens:
        tier.up_to_input_tokens === undefined
          ? ''
          : String(tier.up_to_input_tokens),
      price: tier.price === '' ? '' : String(tier.price),
    })),
  };
}

function serializeDraft(draft) {
  const price = {
    unit: draft.unit,
    tiers: draft.tiers.map((tier) => ({
      ...(String(tier.up_to_input_tokens).trim()
        ? { up_to_input_tokens: Number(tier.up_to_input_tokens) }
        : {}),
      price: Number(tier.price),
    })),
  };
  if (draft.source_model.trim()) price.source_model = draft.source_model.trim();
  if (draft.source_url.trim()) price.source_url = draft.source_url.trim();
  if (draft.verified_at) price.verified_at = draft.verified_at;
  if (draft.notes.trim()) price.notes = draft.notes.trim();
  return price;
}

function validateDraft(draft, t) {
  if (!draft.model.trim()) return t('请选择模型');
  if (![TOKEN_UNIT, REQUEST_UNIT].includes(draft.unit)) {
    return t('请选择有效的计价单位');
  }
  if (draft.unit === REQUEST_UNIT && draft.tiers.length !== 1) {
    return t('按次参考价只能包含一个档位');
  }
  if (draft.source_url.trim()) {
    try {
      const url = new URL(draft.source_url);
      if (!['http:', 'https:'].includes(url.protocol)) {
        return t('来源链接必须是 HTTP 或 HTTPS 地址');
      }
    } catch {
      return t('来源链接必须是有效网址');
    }
  }
  if (draft.verified_at && !/^\d{4}-\d{2}-\d{2}$/.test(draft.verified_at)) {
    return t('核验日期格式无效');
  }

  let previousLimit = 0;
  for (let index = 0; index < draft.tiers.length; index += 1) {
    const tier = draft.tiers[index];
    const price = Number(tier.price);
    if (!Number.isFinite(price) || price <= 0) {
      return t('官方价格必须大于零');
    }
    const isLast = index === draft.tiers.length - 1;
    const rawLimit = String(tier.up_to_input_tokens).trim();
    if (isLast && rawLimit) return t('最后一个档位不能设置上限');
    if (!isLast) {
      const limit = Number(rawLimit);
      if (!Number.isInteger(limit) || limit <= previousLimit) {
        return t('档位上限必须为递增的正整数');
      }
      previousLimit = limit;
    }
  }
  return '';
}

export default function OfficialPriceSettings({ value, refresh }) {
  const { t } = useTranslation();
  const [prices, setPrices] = useState(() => parsePrices(value));
  const [savedSnapshot, setSavedSnapshot] = useState(() =>
    JSON.stringify(parsePrices(value)),
  );
  const [search, setSearch] = useState('');
  const [draft, setDraft] = useState(null);
  const [modelOptions, setModelOptions] = useState([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const next = parsePrices(value);
    setPrices(next);
    setSavedSnapshot(JSON.stringify(next));
  }, [value]);

  useEffect(() => {
    let cancelled = false;
    API.get('/api/models/', { params: { p: 1, page_size: 1000 } })
      .then((response) => {
        if (cancelled) return;
        const items = response.data?.data?.items || [];
        setModelOptions(
          [
            ...new Set(items.map((item) => item.model_name).filter(Boolean)),
          ].sort(),
        );
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  const rows = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return Object.entries(prices)
      .map(([model, price]) => ({ key: model, model, price }))
      .filter(
        (row) =>
          !keyword ||
          row.model.toLowerCase().includes(keyword) ||
          row.price.source_model?.toLowerCase().includes(keyword),
      )
      .sort((left, right) => left.model.localeCompare(right.model));
  }, [prices, search]);

  const dirty = JSON.stringify(prices) !== savedSnapshot;
  const save = async () => {
    setSaving(true);
    try {
      const serialized = JSON.stringify(prices);
      const response = await API.put('/api/option/', {
        key: 'official_price_setting.model_prices',
        value: serialized,
      });
      if (!response.data?.success) {
        showError(response.data?.message || t('保存失败'));
        return;
      }
      setSavedSnapshot(serialized);
      showSuccess(t('保存成功'));
      await refresh();
    } catch (error) {
      showError(error.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const applyDraft = () => {
    const error = validateDraft(draft, t);
    if (error) {
      showError(error);
      return;
    }
    const model = draft.model.trim();
    setPrices((current) => {
      const next = { ...current };
      if (draft.originalModel && draft.originalModel !== model) {
        delete next[draft.originalModel];
      }
      next[model] = serializeDraft(draft);
      return next;
    });
    setDraft(null);
  };

  const columns = [
    {
      title: t('模型'),
      dataIndex: 'model',
      render: (model) => <Text code>{model}</Text>,
    },
    {
      title: t('计价单位'),
      dataIndex: 'unit',
      render: (_unit, row) =>
        row.price.unit === REQUEST_UNIT ? t('每次请求') : t('每百万输入 Token'),
    },
    {
      title: t('档位'),
      dataIndex: 'tiers',
      render: (_tiers, row) => (
        <div className='flex flex-wrap gap-1'>
          {(row.price.tiers || []).map((tier, index) => (
            <Tag key={`${tier.up_to_input_tokens || 'last'}-${index}`}>
              {tier.up_to_input_tokens
                ? `≤ ${tier.up_to_input_tokens}: $${tier.price}`
                : `$${tier.price}`}
            </Tag>
          ))}
        </div>
      ),
    },
    {
      title: t('核验日期'),
      dataIndex: 'verified_at',
      render: (_verifiedAt, row) => row.price.verified_at || '-',
    },
    {
      title: t('操作'),
      width: 120,
      render: (_text, row) => (
        <div className='flex gap-1'>
          <Button
            theme='borderless'
            icon={<IconEdit />}
            aria-label={t('编辑')}
            onClick={() => setDraft(createDraft(row.model, row.price))}
          />
          <Button
            theme='borderless'
            type='danger'
            icon={<IconDelete />}
            aria-label={t('删除')}
            onClick={() =>
              Modal.confirm({
                title: t('删除官方参考价？'),
                content: row.model,
                okType: 'danger',
                onOk: () =>
                  setPrices((current) => {
                    const next = { ...current };
                    delete next[row.model];
                    return next;
                  }),
              })
            }
          />
        </div>
      ),
    },
  ];

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap gap-2'>
        <Input
          value={search}
          onChange={setSearch}
          placeholder={t('搜索模型或来源型号')}
          style={{ flex: 1, minWidth: 220 }}
        />
        <Button icon={<IconPlus />} onClick={() => setDraft(createDraft())}>
          {t('新增参考价')}
        </Button>
        <Button
          theme='solid'
          icon={<IconSave />}
          loading={saving}
          disabled={!dirty}
          onClick={save}
        >
          {t('保存修改')}
        </Button>
      </div>

      <div className='overflow-x-auto'>
        <Table
          dataSource={rows}
          columns={columns}
          pagination={false}
          size='small'
          scroll={{ x: 'max-content' }}
          empty={t('暂无官方参考价')}
        />
      </div>
      {dirty && <Text type='tertiary'>{t('存在尚未保存的参考价修改')}</Text>}

      <Modal
        title={draft?.originalModel ? t('编辑官方参考价') : t('新增官方参考价')}
        visible={Boolean(draft)}
        width={760}
        onCancel={() => setDraft(null)}
        onOk={applyDraft}
        okText={t('应用')}
        cancelText={t('取消')}
        closeOnEsc={false}
        maskClosable={false}
      >
        {draft && (
          <Form layout='vertical'>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
              <Form.Slot label={t('模型')}>
                {draft.originalModel ? (
                  <Input value={draft.model} disabled />
                ) : (
                  <Select
                    filter
                    style={{ width: '100%' }}
                    value={draft.model || undefined}
                    placeholder={t('搜索并选择模型')}
                    optionList={[
                      ...new Set([...modelOptions, ...Object.keys(prices)]),
                    ].map((model) => ({ value: model, label: model }))}
                    onChange={(model) =>
                      setDraft((current) => ({
                        ...current,
                        model,
                        source_model: current.source_model || model,
                      }))
                    }
                  />
                )}
              </Form.Slot>
              <Form.Slot label={t('计价单位')}>
                <Select
                  style={{ width: '100%' }}
                  value={draft.unit}
                  optionList={[
                    { value: TOKEN_UNIT, label: t('每百万输入 Token') },
                    { value: REQUEST_UNIT, label: t('每次请求') },
                  ]}
                  onChange={(unit) =>
                    setDraft((current) => ({
                      ...current,
                      unit,
                      tiers:
                        unit === REQUEST_UNIT
                          ? [{ up_to_input_tokens: '', price: '' }]
                          : current.tiers,
                    }))
                  }
                />
              </Form.Slot>
            </div>

            <Form.Slot label={t('价格档位')}>
              <div className='space-y-2'>
                {draft.tiers.map((tier, index) => {
                  const isLast = index === draft.tiers.length - 1;
                  return (
                    <div
                      key={index}
                      className='grid grid-cols-[1fr_1fr_auto] gap-2 rounded border p-3'
                    >
                      <Input
                        type='number'
                        disabled={isLast}
                        value={tier.up_to_input_tokens}
                        placeholder={isLast ? t('无上限') : '272000'}
                        prefix={t('输入上限')}
                        onChange={(up_to_input_tokens) =>
                          setDraft((current) => ({
                            ...current,
                            tiers: current.tiers.map((item, itemIndex) =>
                              itemIndex === index
                                ? { ...item, up_to_input_tokens }
                                : item,
                            ),
                          }))
                        }
                      />
                      <Input
                        type='number'
                        value={tier.price}
                        prefix='$'
                        placeholder={t('官方价格')}
                        onChange={(price) =>
                          setDraft((current) => ({
                            ...current,
                            tiers: current.tiers.map((item, itemIndex) =>
                              itemIndex === index ? { ...item, price } : item,
                            ),
                          }))
                        }
                      />
                      <Button
                        theme='borderless'
                        type='danger'
                        icon={<IconDelete />}
                        disabled={draft.tiers.length === 1 || isLast}
                        onClick={() =>
                          setDraft((current) => ({
                            ...current,
                            tiers: current.tiers.filter(
                              (_item, itemIndex) => itemIndex !== index,
                            ),
                          }))
                        }
                      />
                    </div>
                  );
                })}
                {draft.unit === TOKEN_UNIT && (
                  <Button
                    theme='outline'
                    icon={<IconPlus />}
                    onClick={() =>
                      setDraft((current) => {
                        const tiers = [...current.tiers];
                        tiers.splice(Math.max(0, tiers.length - 1), 0, {
                          up_to_input_tokens: '',
                          price: '',
                        });
                        return { ...current, tiers };
                      })
                    }
                  >
                    {t('新增档位')}
                  </Button>
                )}
              </div>
            </Form.Slot>

            <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
              <Form.Slot label={t('来源型号')}>
                <Input
                  value={draft.source_model}
                  onChange={(source_model) =>
                    setDraft((current) => ({ ...current, source_model }))
                  }
                />
              </Form.Slot>
              <Form.Slot label={t('核验日期')}>
                <Input
                  type='date'
                  value={draft.verified_at}
                  onChange={(verified_at) =>
                    setDraft((current) => ({ ...current, verified_at }))
                  }
                />
              </Form.Slot>
            </div>
            <Form.Slot label={t('来源链接')}>
              <Input
                value={draft.source_url}
                onChange={(source_url) =>
                  setDraft((current) => ({ ...current, source_url }))
                }
              />
            </Form.Slot>
            <Form.Slot label={t('备注')}>
              <TextArea
                value={draft.notes}
                autosize={{ minRows: 3, maxRows: 6 }}
                onChange={(notes) =>
                  setDraft((current) => ({ ...current, notes }))
                }
              />
            </Form.Slot>
          </Form>
        )}
      </Modal>
    </div>
  );
}
