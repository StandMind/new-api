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
  Modal,
  Select,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconSave } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

import { API, showError, showSuccess } from '../../../helpers';
import GroupModelRouteTierEditor from './GroupModelRouteTierEditor';
import {
  cloneGroupModelRouteTiers,
  getGroupModelRouteKey,
  serializeGroupModelRouteTiers,
} from './groupModelRouteUtils';

const { Text } = Typography;

export default function GroupModelRouteEditorModal({
  target,
  groups,
  savedRoutes,
  onClose,
  onSwitchToEdit,
  onSaved,
  onRestored,
}) {
  const { t } = useTranslation();
  const editing = target.mode === 'edit';
  const editGroup = editing ? target.group : '';
  const editModel = editing ? target.model : '';
  const [draftGroup, setDraftGroup] = useState(editGroup);
  const [draftModel, setDraftModel] = useState(editModel);
  const [models, setModels] = useState([]);
  const [tiers, setTiers] = useState([]);
  const [candidates, setCandidates] = useState([]);
  const [explicit, setExplicit] = useState(false);
  const [loadedKey, setLoadedKey] = useState('');
  const [initialSnapshot, setInitialSnapshot] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [reloadVersion, setReloadVersion] = useState(0);

  const group = editing ? editGroup : draftGroup;
  const model = editing ? editModel : draftModel.trim();
  const routeKey = group && model ? getGroupModelRouteKey(group, model) : '';
  const routeLoaded = Boolean(routeKey && routeKey === loadedKey);
  const dirty =
    routeLoaded && serializeGroupModelRouteTiers(tiers) !== initialSnapshot;
  const savedRoute = useMemo(
    () =>
      savedRoutes.find(
        (route) => route.group === group && route.model === model,
      ),
    [group, model, savedRoutes],
  );
  const savedChannelNames = useMemo(() => {
    const names = new Map();
    for (const tier of savedRoute?.tiers || []) {
      for (const channel of tier.channels) {
        names.set(channel.channel_id, channel.channel_name);
      }
    }
    return names;
  }, [savedRoute]);
  const candidateIds = useMemo(
    () => new Set(candidates.map((candidate) => candidate.channel_id)),
    [candidates],
  );

  useEffect(() => {
    void API.get('/api/channel/models').then((response) => {
      if (!response.data?.success || !response.data?.data) return;
      setModels(
        [
          ...new Set(response.data.data.map((item) => item.id).filter(Boolean)),
        ].sort(),
      );
    });
  }, []);

  useEffect(() => {
    if (!group || !model) {
      setLoadedKey('');
      setInitialSnapshot('');
      setTiers([]);
      setCandidates([]);
      setExplicit(false);
      setLoadError('');
      return;
    }
    if (!editing && savedRoute) {
      onSwitchToEdit(group, model);
      return;
    }

    let cancelled = false;
    setLoadedKey('');
    setInitialSnapshot('');
    setTiers([]);
    setCandidates([]);
    setLoadError('');
    setLoading(true);
    void API.get('/api/group-model-routes', {
      params: { group, model },
    })
      .then((response) => {
        if (cancelled) return;
        if (!response.data?.success || !response.data?.data) {
          setLoadError(response.data?.message || t('加载渠道链失败'));
          return;
        }
        const nextTiers = cloneGroupModelRouteTiers(
          response.data.data.route?.tiers,
        );
        setExplicit(Boolean(response.data.data.explicit));
        setCandidates(response.data.data.candidates || []);
        setTiers(nextTiers);
        setInitialSnapshot(serializeGroupModelRouteTiers(nextTiers));
        setLoadedKey(routeKey);
      })
      .catch((error) => {
        if (!cancelled) {
          setLoadError(error.message || t('加载渠道链失败'));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    editing,
    group,
    model,
    onSwitchToEdit,
    reloadVersion,
    routeKey,
    savedRoute,
    t,
  ]);

  const requestClose = () => {
    if (!dirty) {
      onClose();
      return;
    }
    Modal.confirm({
      title: t('放弃未保存的渠道链修改？'),
      content: t('尚未保存的优先级和渠道修改将会丢失'),
      okText: t('放弃修改'),
      cancelText: t('取消'),
      okType: 'danger',
      onOk: onClose,
    });
  };

  const validate = () => {
    if (!routeLoaded) {
      showError(t('请等待渠道链加载完成'));
      return false;
    }
    if (tiers.length === 0) {
      showError(t('请至少添加一个优先级层级'));
      return false;
    }
    const priorities = new Set();
    for (const tier of tiers) {
      if (!Number.isInteger(tier.priority) || priorities.has(tier.priority)) {
        showError(t('每个层级必须使用唯一的整数优先级'));
        return false;
      }
      priorities.add(tier.priority);
      if (tier.channels.length === 0) {
        showError(t('每个优先级层级至少需要一个渠道'));
        return false;
      }
      if (
        tier.channels.some(
          (channel) => !Number.isInteger(channel.weight) || channel.weight < 0,
        )
      ) {
        showError(t('渠道权重必须是非负整数'));
        return false;
      }
      if (
        tier.channels.some((channel) => !candidateIds.has(channel.channel_id))
      ) {
        showError(t('渠道链包含不可用渠道，请在保存前移除或替换'));
        return false;
      }
    }
    return true;
  };

  const saveRoute = async () => {
    if (!validate()) return;
    setSaving(true);
    try {
      const response = await API.put('/api/group-model-routes', {
        group,
        model,
        tiers: tiers.map((tier) => ({
          priority: tier.priority,
          channels: tier.channels,
        })),
      });
      if (!response.data?.success) {
        showError(response.data?.message || t('保存渠道链失败'));
        return;
      }
      showSuccess(t('分组模型渠道链已保存'));
      await onSaved(group, model);
    } catch (error) {
      showError(error.message || t('保存渠道链失败'));
    } finally {
      setSaving(false);
    }
  };

  const restoreRoute = () => {
    Modal.confirm({
      title: t('恢复继承渠道链？'),
      content: t(
        '该操作会移除显式渠道链，并恢复使用当前 Ability 的优先级和权重',
      ),
      okText: t('恢复继承渠道链'),
      cancelText: t('取消'),
      okType: 'danger',
      onOk: async () => {
        setRestoring(true);
        try {
          const response = await API.delete('/api/group-model-routes', {
            params: { group, model },
          });
          if (!response.data?.success) {
            throw new Error(response.data?.message || t('恢复默认渠道链失败'));
          }
          showSuccess(t('已恢复 Ability 默认渠道链'));
          await onRestored(group, model);
        } catch (error) {
          showError(error.message || t('恢复默认渠道链失败'));
          throw error;
        } finally {
          setRestoring(false);
        }
      },
    });
  };

  let routeContent;
  if (!group || !model) {
    routeContent = (
      <div className='flex min-h-40 items-center justify-center'>
        <Text type='tertiary'>{t('请选择分组并输入精确模型名称')}</Text>
      </div>
    );
  } else if (loading || !routeLoaded) {
    routeContent = (
      <div className='flex min-h-40 items-center justify-center'>
        {loadError ? (
          <div className='flex flex-col items-center gap-3 text-center'>
            <Text type='danger'>{loadError}</Text>
            <Button
              theme='outline'
              onClick={() => setReloadVersion((current) => current + 1)}
            >
              {t('重试')}
            </Button>
          </div>
        ) : (
          <Spin />
        )}
      </div>
    );
  } else {
    routeContent = (
      <div className='space-y-4'>
        <div className='flex flex-wrap items-center gap-2'>
          <Tag color={explicit ? 'blue' : 'grey'}>
            {explicit ? t('显式渠道链') : t('继承 Ability 配置')}
          </Tag>
          <Text type='tertiary' size='small'>
            {t('{{count}} 个候选渠道', { count: candidates.length })}
          </Text>
        </div>
        <GroupModelRouteTierEditor
          tiers={tiers}
          candidates={candidates}
          savedChannelNames={savedChannelNames}
          onChange={setTiers}
        />
      </div>
    );
  }

  return (
    <Modal
      visible
      width={960}
      style={{
        top: 0,
        margin: 'min(80px, 8vh) auto',
        maxWidth: 'calc(100vw - 24px)',
      }}
      title={editing ? t('编辑分组模型渠道链') : t('新建分组模型渠道链')}
      onCancel={requestClose}
      maskClosable={!saving && !restoring}
      closeOnEsc={!saving && !restoring}
      bodyStyle={{ maxHeight: 'calc(100vh - 190px)', overflowY: 'auto' }}
      footer={
        <div className='flex flex-wrap justify-end gap-2'>
          <Button
            theme='outline'
            onClick={requestClose}
            disabled={saving || restoring}
          >
            {t('取消')}
          </Button>
          {editing && explicit && (
            <Button
              icon={<IconRefresh />}
              theme='outline'
              loading={restoring}
              disabled={saving}
              onClick={restoreRoute}
            >
              {t('恢复继承渠道链')}
            </Button>
          )}
          <Button
            icon={<IconSave />}
            loading={saving}
            disabled={!routeLoaded || loading || restoring}
            onClick={saveRoute}
          >
            {t('保存渠道链')}
          </Button>
        </div>
      }
    >
      <div className='space-y-4 py-2'>
        {editing ? (
          <div className='flex flex-wrap gap-2'>
            <Tag color='blue'>{editGroup}</Tag>
            <Tag color='grey' className='font-mono'>
              {editModel}
            </Tag>
          </div>
        ) : (
          <div className='grid gap-3 sm:grid-cols-2'>
            <label className='space-y-1'>
              <Text strong>{t('分组')}</Text>
              <Select
                value={draftGroup || undefined}
                placeholder={t('选择分组')}
                optionList={groups.map((item) => ({
                  value: item,
                  label: item,
                }))}
                onChange={(value) => setDraftGroup(value)}
                style={{ width: '100%' }}
              />
            </label>
            <label className='space-y-1'>
              <Text strong>{t('模型')}</Text>
              <Select
                filter
                allowCreate
                value={draftModel || undefined}
                optionList={models.map((item) => ({
                  value: item,
                  label: item,
                }))}
                placeholder={t('输入精确模型名称')}
                onChange={(value) => setDraftModel(value)}
                style={{ width: '100%' }}
              />
            </label>
          </div>
        )}
        {routeContent}
      </div>
    </Modal>
  );
}
