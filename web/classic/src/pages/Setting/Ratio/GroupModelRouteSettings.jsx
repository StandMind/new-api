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
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconChevronDown,
  IconChevronUp,
  IconEdit,
  IconPlus,
  IconRefresh,
  IconSearch,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

import { API, showError, showSuccess } from '../../../helpers';
import GroupModelRouteEditorModal from './GroupModelRouteEditorModal';
import {
  countGroupModelRouteChannels,
  getGroupModelRouteKey,
  groupModelRouteMatchesSearch,
  parseGroupModelRouteGroups,
} from './groupModelRouteUtils';

const { Text, Title } = Typography;

export default function GroupModelRouteSettings({ groupRatio }) {
  const { t } = useTranslation();
  const [savedRoutes, setSavedRoutes] = useState([]);
  const [routeListLoading, setRouteListLoading] = useState(false);
  const [routeListError, setRouteListError] = useState('');
  const [search, setSearch] = useState('');
  const [groupFilter, setGroupFilter] = useState('');
  const [expandedRowKeys, setExpandedRowKeys] = useState([]);
  const [editorTarget, setEditorTarget] = useState(null);
  const [restoringKey, setRestoringKey] = useState('');

  const groups = useMemo(
    () =>
      [
        ...new Set([
          ...parseGroupModelRouteGroups(groupRatio),
          ...savedRoutes.map((route) => route.group),
        ]),
      ].sort(),
    [groupRatio, savedRoutes],
  );
  const routeGroups = useMemo(
    () => [...new Set(savedRoutes.map((route) => route.group))].sort(),
    [savedRoutes],
  );
  const filteredRoutes = useMemo(
    () =>
      savedRoutes.filter(
        (route) =>
          (!groupFilter || route.group === groupFilter) &&
          groupModelRouteMatchesSearch(route, search),
      ),
    [groupFilter, savedRoutes, search],
  );
  const allVisibleExpanded =
    filteredRoutes.length > 0 &&
    filteredRoutes.every((route) =>
      expandedRowKeys.includes(getGroupModelRouteKey(route.group, route.model)),
    );

  const loadSavedRoutes = useCallback(async () => {
    setRouteListLoading(true);
    setRouteListError('');
    try {
      const response = await API.get('/api/group-model-routes/list');
      if (!response.data?.success) {
        const message = response.data?.message || t('加载渠道链失败');
        setRouteListError(message);
        showError(message);
        return;
      }
      setSavedRoutes(response.data.data || []);
    } catch (error) {
      const message = error.message || t('加载渠道链失败');
      setRouteListError(message);
      showError(message);
    } finally {
      setRouteListLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadSavedRoutes();
  }, [loadSavedRoutes]);

  const toggleAllVisible = () => {
    setExpandedRowKeys((current) => {
      const next = new Set(current);
      for (const route of filteredRoutes) {
        const key = getGroupModelRouteKey(route.group, route.model);
        if (allVisibleExpanded) {
          next.delete(key);
        } else {
          next.add(key);
        }
      }
      return [...next];
    });
  };

  const handleSaved = async (group, model) => {
    const key = getGroupModelRouteKey(group, model);
    setExpandedRowKeys((current) => [...new Set([...current, key])]);
    setEditorTarget(null);
    await loadSavedRoutes();
  };

  const handleRestored = async (group, model) => {
    const key = getGroupModelRouteKey(group, model);
    setExpandedRowKeys((current) => current.filter((item) => item !== key));
    setEditorTarget(null);
    await loadSavedRoutes();
  };

  const switchToEdit = useCallback((group, model) => {
    setEditorTarget({ mode: 'edit', group, model });
  }, []);

  const restoreRoute = (route) => {
    const key = getGroupModelRouteKey(route.group, route.model);
    Modal.confirm({
      title: t('恢复继承渠道链？'),
      content: t(
        '该操作会移除显式渠道链，并恢复使用当前 Ability 的优先级和权重',
      ),
      okText: t('恢复继承渠道链'),
      cancelText: t('取消'),
      okType: 'danger',
      onOk: async () => {
        setRestoringKey(key);
        try {
          const response = await API.delete('/api/group-model-routes', {
            params: { group: route.group, model: route.model },
          });
          if (!response.data?.success) {
            throw new Error(response.data?.message || t('恢复默认渠道链失败'));
          }
          showSuccess(t('已恢复 Ability 默认渠道链'));
          await handleRestored(route.group, route.model);
        } catch (error) {
          showError(error.message || t('恢复默认渠道链失败'));
          throw error;
        } finally {
          setRestoringKey('');
        }
      },
    });
  };

  const columns = useMemo(
    () => [
      {
        title: t('分组'),
        dataIndex: 'group',
        width: 130,
        render: (group) => <Tag color='blue'>{group}</Tag>,
      },
      {
        title: t('模型'),
        dataIndex: 'model',
        render: (model) => (
          <span className='block max-w-72 truncate font-mono' title={model}>
            {model}
          </span>
        ),
      },
      {
        title: t('渠道链摘要'),
        key: 'summary',
        width: 290,
        render: (_, route) => (
          <div className='flex flex-wrap items-center gap-1.5'>
            <span>
              {t('{{count}} 个优先级层级', {
                count: route.tiers.length,
              })}
            </span>
            <Text type='tertiary'>·</Text>
            <span>
              {t('{{count}} 个渠道', {
                count: countGroupModelRouteChannels(route),
              })}
            </span>
            <Text type='tertiary' size='small' className='font-mono'>
              {route.tiers.map((tier) => `P${tier.priority}`).join(' → ')}
            </Text>
          </div>
        ),
      },
      {
        title: t('更新时间'),
        dataIndex: 'updated_at',
        width: 180,
        render: (updatedAt) => (
          <Text type='tertiary' size='small'>
            {updatedAt ? new Date(updatedAt).toLocaleString() : '-'}
          </Text>
        ),
      },
      {
        title: '',
        key: 'actions',
        width: 90,
        align: 'right',
        render: (_, route) => {
          const key = getGroupModelRouteKey(route.group, route.model);
          return (
            <Space spacing={2}>
              <Tooltip content={t('编辑渠道链')}>
                <Button
                  icon={<IconEdit />}
                  theme='borderless'
                  aria-label={t('编辑渠道链')}
                  onClick={(event) => {
                    event.stopPropagation();
                    setEditorTarget({
                      mode: 'edit',
                      group: route.group,
                      model: route.model,
                    });
                  }}
                />
              </Tooltip>
              <Tooltip content={t('恢复继承渠道链')}>
                <Button
                  icon={<IconRefresh />}
                  theme='borderless'
                  loading={restoringKey === key}
                  aria-label={t('恢复继承渠道链')}
                  onClick={(event) => {
                    event.stopPropagation();
                    restoreRoute(route);
                  }}
                />
              </Tooltip>
            </Space>
          );
        },
      },
    ],
    [restoringKey, t],
  );

  const expandedRowRender = (route) => (
    <div className='divide-y divide-gray-100'>
      {route.tiers.map((tier) => (
        <div
          key={tier.priority}
          className='grid gap-3 px-3 py-3 md:grid-cols-[8rem_minmax(0,1fr)]'
        >
          <div>
            <Tag color='grey'>
              {t('优先级 {{priority}}', { priority: tier.priority })}
            </Tag>
          </div>
          <div className='flex min-w-0 flex-wrap gap-2'>
            {tier.channels.map((channel) => (
              <div
                key={channel.channel_id}
                className='flex min-w-0 items-center gap-2 rounded-md border border-solid border-gray-200 bg-white px-2.5 py-1.5'
              >
                {!channel.eligible && (
                  <Tooltip content={t('当前不可用')}>
                    <Tag color='red' size='small'>
                      {t('不可用')}
                    </Tag>
                  </Tooltip>
                )}
                <span
                  className='max-w-56 truncate'
                  title={channel.channel_name || t('已删除或不可用的渠道')}
                >
                  <span className='font-mono text-gray-500'>
                    #{channel.channel_id}
                  </span>{' '}
                  {channel.channel_name || t('已删除或不可用的渠道')}
                </span>
                <Tag color={channel.eligible ? 'grey' : 'red'}>
                  {t('权重 {{weight}}', { weight: channel.weight })}
                </Tag>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );

  let routeContent;
  if (routeListError && savedRoutes.length === 0) {
    routeContent = (
      <div className='flex min-h-48 flex-col items-center justify-center gap-3 rounded-lg border border-solid border-gray-200 text-center'>
        <Text type='danger'>{routeListError}</Text>
        <Button
          icon={<IconRefresh />}
          theme='outline'
          loading={routeListLoading}
          onClick={loadSavedRoutes}
        >
          {t('重试')}
        </Button>
      </div>
    );
  } else if (!routeListLoading && savedRoutes.length === 0) {
    routeContent = (
      <div className='flex min-h-56 flex-col items-center justify-center gap-3 rounded-lg border border-solid border-gray-200'>
        <Empty description={t('暂无显式渠道链')} />
        <Button
          icon={<IconPlus />}
          onClick={() => setEditorTarget({ mode: 'create' })}
        >
          {t('新建渠道链')}
        </Button>
      </div>
    );
  } else {
    routeContent = (
      <Table
        columns={columns}
        dataSource={filteredRoutes}
        rowKey={(route) => getGroupModelRouteKey(route.group, route.model)}
        expandedRowRender={expandedRowRender}
        expandedRowKeys={expandedRowKeys}
        onExpandedRowsChange={setExpandedRowKeys}
        pagination={false}
        loading={routeListLoading}
        size='small'
        scroll={{ x: 900 }}
        empty={
          <div className='flex flex-col items-center gap-3 py-8'>
            <Empty description={t('没有匹配的渠道链')} />
            <Button
              theme='outline'
              onClick={() => {
                setSearch('');
                setGroupFilter('');
              }}
            >
              {t('清除筛选')}
            </Button>
          </div>
        }
      />
    );
  }

  return (
    <div className='space-y-5 py-3'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
        <div>
          <div className='flex items-center gap-2'>
            <Title heading={5}>{t('分组模型渠道链')}</Title>
            <Tag color='grey'>{savedRoutes.length}</Tag>
          </div>
          <Text type='tertiary'>
            {t('展开任意路线即可查看完整优先级层级、渠道和权重')}
          </Text>
        </div>
        <Button
          icon={<IconPlus />}
          onClick={() => setEditorTarget({ mode: 'create' })}
        >
          {t('新建渠道链')}
        </Button>
      </div>

      {savedRoutes.length > 0 && (
        <div className='flex flex-col gap-2 lg:flex-row lg:items-center'>
          <Input
            prefix={<IconSearch />}
            showClear
            value={search}
            placeholder={t('搜索分组、模型、渠道名称或 ID')}
            onChange={setSearch}
            onClear={() => setSearch('')}
            style={{ flex: 1 }}
          />
          <Select
            value={groupFilter || undefined}
            placeholder={t('全部分组')}
            optionList={[
              { value: '', label: t('全部分组') },
              ...routeGroups.map((group) => ({
                value: group,
                label: group,
              })),
            ]}
            onChange={setGroupFilter}
            style={{ width: 180 }}
          />
          <Button
            icon={allVisibleExpanded ? <IconChevronUp /> : <IconChevronDown />}
            theme='outline'
            disabled={filteredRoutes.length === 0}
            onClick={toggleAllVisible}
          >
            {allVisibleExpanded ? t('全部收起') : t('全部展开')}
          </Button>
          <Tooltip content={t('刷新')}>
            <Button
              icon={<IconRefresh />}
              theme='outline'
              loading={routeListLoading}
              aria-label={t('刷新')}
              onClick={loadSavedRoutes}
            />
          </Tooltip>
        </div>
      )}

      {routeContent}

      {editorTarget && (
        <GroupModelRouteEditorModal
          key={
            editorTarget.mode === 'edit'
              ? getGroupModelRouteKey(editorTarget.group, editorTarget.model)
              : 'create'
          }
          target={editorTarget}
          groups={groups}
          savedRoutes={savedRoutes}
          onClose={() => setEditorTarget(null)}
          onSwitchToEdit={switchToEdit}
          onSaved={handleSaved}
          onRestored={handleRestored}
        />
      )}
    </div>
  );
}
