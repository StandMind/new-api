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
  Descriptions,
  Form,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconEyeOpened, IconRefresh, IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../../helpers';

const { Text } = Typography;
const PAGE_SIZE = 50;

function formatTime(timestamp) {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
}

export default function RequestDetailsTable() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [records, setRecords] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [draftFilters, setDraftFilters] = useState({
    request_id: '',
    username: '',
    model_name: '',
    outcome: '',
  });
  const [filters, setFilters] = useState(draftFilters);
  const [selectedRequestId, setSelectedRequestId] = useState('');
  const [selectedDetail, setSelectedDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);

  async function loadRecords(targetPage = page, targetFilters = filters) {
    setLoading(true);
    try {
      const res = await API.get('/api/request-detail/', {
        params: {
          p: targetPage,
          page_size: PAGE_SIZE,
          ...targetFilters,
        },
      });
      if (!res.data.success) {
        throw new Error(res.data.message || t('加载请求详情失败'));
      }
      setRecords(res.data.data?.items || []);
      setTotal(res.data.data?.total || 0);
    } catch (error) {
      showError(error.message || t('加载请求详情失败'));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadRecords();
  }, [page, filters]);

  useEffect(() => {
    if (!selectedRequestId) {
      setSelectedDetail(null);
      return;
    }
    let cancelled = false;
    async function loadDetail() {
      setDetailLoading(true);
      try {
        const res = await API.get(
          `/api/request-detail/${encodeURIComponent(selectedRequestId)}`,
        );
        if (!cancelled) {
          if (!res.data.success) {
            throw new Error(res.data.message || t('请求详情已过期'));
          }
          setSelectedDetail(res.data.data);
        }
      } catch (error) {
        if (!cancelled) showError(error.message || t('请求详情已过期'));
      } finally {
        if (!cancelled) setDetailLoading(false);
      }
    }
    loadDetail();
    return () => {
      cancelled = true;
    };
  }, [selectedRequestId]);

  const columns = useMemo(
    () => [
      {
        title: t('时间'),
        dataIndex: 'created_at',
        width: 180,
        render: formatTime,
      },
      {
        title: t('结果'),
        dataIndex: 'outcome',
        width: 90,
        render: (value) => (
          <Tag color={value === 'failed' ? 'red' : 'green'}>
            {value === 'failed' ? t('失败') : t('成功')}
          </Tag>
        ),
      },
      {
        title: t('用户'),
        dataIndex: 'username',
        width: 130,
        render: (value, record) => value || `#${record.user_id}`,
      },
      {
        title: t('模型'),
        dataIndex: 'model_name',
        width: 190,
      },
      {
        title: t('请求'),
        width: 300,
        render: (_, record) => (
          <Text code>{`${record.method} ${record.path}`}</Text>
        ),
      },
      {
        title: t('状态码'),
        dataIndex: 'status_code',
        width: 90,
      },
      {
        title: t('请求 ID'),
        dataIndex: 'request_id',
        width: 260,
        render: (value) => <Text code>{value}</Text>,
      },
      {
        title: t('操作'),
        fixed: 'right',
        width: 80,
        render: (_, record) => (
          <Button
            theme='borderless'
            icon={<IconEyeOpened />}
            aria-label={t('查看请求详情')}
            onClick={() => setSelectedRequestId(record.request_id)}
          />
        ),
      },
    ],
    [t],
  );

  const detailDescription = selectedDetail
    ? [
        { key: t('请求 ID'), value: selectedDetail.detail.request_id },
        { key: t('时间'), value: formatTime(selectedDetail.detail.created_at) },
        {
          key: t('请求'),
          value: `${selectedDetail.detail.method} ${selectedDetail.detail.path}`,
        },
        { key: t('用户'), value: selectedDetail.detail.username || '-' },
        { key: t('模型'), value: selectedDetail.detail.model_name || '-' },
        { key: t('分组'), value: selectedDetail.detail.group || '-' },
        { key: t('状态码'), value: selectedDetail.detail.status_code },
        {
          key: t('错误代码'),
          value: selectedDetail.detail.error_code || '-',
        },
        {
          key: t('错误信息'),
          value: selectedDetail.detail.error_message || '-',
        },
      ]
    : [];
  const requestPayload = selectedDetail?.payload
    ? {
        headers: selectedDetail.payload.headers,
        query: selectedDetail.payload.query,
        body: selectedDetail.payload.body,
        routing: selectedDetail.payload.routing,
      }
    : null;
  const requestPayloadText =
    requestPayload &&
    Object.values(requestPayload).some((value) => value !== undefined)
      ? JSON.stringify(requestPayload, null, 2)
      : '';
  const response = selectedDetail?.payload?.response;
  const responseBodyText =
    response?.body === undefined || response.body === null
      ? ''
      : typeof response.body === 'string'
        ? response.body
        : JSON.stringify(response.body, null, 2);

  return (
    <div className='space-y-3'>
      <Form
        layout='horizontal'
        onSubmit={() => {
          setPage(1);
          setFilters({ ...draftFilters });
        }}
      >
        <Form.Input
          field='request_id'
          placeholder={t('请求 ID')}
          value={draftFilters.request_id}
          onChange={(value) =>
            setDraftFilters({ ...draftFilters, request_id: value })
          }
        />
        <Form.Input
          field='username'
          placeholder={t('用户名')}
          value={draftFilters.username}
          onChange={(value) =>
            setDraftFilters({ ...draftFilters, username: value })
          }
        />
        <Form.Input
          field='model_name'
          placeholder={t('模型')}
          value={draftFilters.model_name}
          onChange={(value) =>
            setDraftFilters({ ...draftFilters, model_name: value })
          }
        />
        <Form.Select
          field='outcome'
          placeholder={t('全部结果')}
          value={draftFilters.outcome}
          optionList={[
            { value: '', label: t('全部结果') },
            { value: 'failed', label: t('失败') },
            { value: 'success', label: t('成功') },
          ]}
          onChange={(value) =>
            setDraftFilters({ ...draftFilters, outcome: value })
          }
        />
        <Space>
          <Button htmlType='submit' icon={<IconSearch />}>
            {t('搜索')}
          </Button>
          <Button
            type='tertiary'
            icon={<IconRefresh />}
            loading={loading}
            onClick={() => loadRecords()}
          >
            {t('刷新')}
          </Button>
        </Space>
      </Form>

      <Table
        columns={columns}
        dataSource={records}
        rowKey='request_id'
        loading={loading}
        scroll={{ x: 1320 }}
        pagination={{
          currentPage: page,
          pageSize: PAGE_SIZE,
          total,
          onPageChange: setPage,
        }}
      />

      <Modal
        title={t('请求详情')}
        visible={selectedRequestId !== ''}
        width={880}
        footer={null}
        onCancel={() => setSelectedRequestId('')}
      >
        <Spin spinning={detailLoading}>
          {selectedDetail && (
            <div className='space-y-4'>
              <Descriptions data={detailDescription} row />
              {requestPayloadText && (
                <div>
                  <Text strong>{t('脱敏后的请求内容')}</Text>
                  <pre
                    style={{
                      maxHeight: 420,
                      overflow: 'auto',
                      marginTop: 8,
                      padding: 12,
                      border: '1px solid var(--semi-color-border)',
                      borderRadius: 6,
                      whiteSpace: 'pre-wrap',
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {requestPayloadText}
                  </pre>
                </div>
              )}
              <div>
                <Text strong>{t('响应内容')}</Text>
                {response ? (
                  <div className='space-y-2 mt-2'>
                    <Descriptions
                      row
                      data={[
                        { key: t('状态码'), value: response.status_code },
                        {
                          key: 'Content-Type',
                          value: response.content_type || '-',
                        },
                        {
                          key: t('大小'),
                          value: `${Number(response.body_size || 0).toLocaleString()} bytes`,
                        },
                      ]}
                    />
                    {response.truncated && (
                      <Text type='warning'>
                        {t('响应正文超过采集上限，已截断显示。')}
                      </Text>
                    )}
                    {responseBodyText ? (
                      <pre
                        style={{
                          maxHeight: 420,
                          overflow: 'auto',
                          padding: 12,
                          border: '1px solid var(--semi-color-border)',
                          borderRadius: 6,
                          whiteSpace: 'pre-wrap',
                          overflowWrap: 'anywhere',
                        }}
                      >
                        {responseBodyText}
                      </pre>
                    ) : (
                      <Text type='tertiary'>
                        {response.omitted_reason
                          ? t('未保存：{{reason}}', {
                              reason: response.omitted_reason,
                            })
                          : t('响应正文为空。')}
                      </Text>
                    )}
                  </div>
                ) : (
                  <div className='mt-2'>
                    <Text type='tertiary'>{t('该记录未采集响应正文。')}</Text>
                  </div>
                )}
              </div>
            </div>
          )}
        </Spin>
      </Modal>
    </div>
  );
}
