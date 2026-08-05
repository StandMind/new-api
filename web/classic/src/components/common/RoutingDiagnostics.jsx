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
import { Descriptions, Tag, Typography } from '@douyinfe/semi-ui';
import { AlertTriangle, CheckCircle2, CircleSlash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export function getRoutingModeLabel(t, mode, short = false) {
  switch (mode) {
    case 'manual':
      return t(short ? 'Manual' : 'Manual group');
    case 'fixed_channel':
      return t(short ? 'Fixed' : 'Fixed channel');
    case 'auto':
      return short ? t('Smart · Auto') : `${t('Smart routing')} · ${t('Auto')}`;
    case 'price':
      return short
        ? t('Smart · Price')
        : `${t('Smart routing')} · ${t('Price')}`;
    case 'speed':
      return short
        ? t('Smart · Speed')
        : `${t('Smart routing')} · ${t('Speed')}`;
    case 'success_rate':
      return short
        ? t('Smart · Success rate')
        : `${t('Smart routing')} · ${t('Success rate')}`;
    default:
      return mode || '-';
  }
}

export function getRoutingReasonLabel(t, reason) {
  if (!reason) return '-';
  switch (reason) {
    case 'success':
      return t('Request completed');
    case 'client_context_canceled':
      return t('Client connection closed');
    case 'client_context_deadline_exceeded':
      return t('Client request deadline exceeded');
    case 'channel_affinity_retry_suppressed':
      return t('Channel affinity disabled retry');
    case 'retry_limit_reached':
      return t('Retry limit reached');
    case 'specific_channel':
      return t('A specific channel was required');
    case 'skip_retry_error':
      return t('The error is marked as non-retryable');
    case 'non_retryable_error_code':
      return t('The error code is configured as non-retryable');
    case 'non_retryable_status':
      return t('The status code is not retryable');
    case 'route_plan_exhausted':
      return t('No route candidates remain');
    case 'response_started':
      return t('The response had already started');
    case 'channel_initialization_failed':
      return t('Channel initialization failed');
    case 'no_available_channel':
      return t('No candidate channel was available');
    case 'no_candidate_groups':
      return t('No candidate group was available');
    case 'route_plan_build_failed':
      return t('The route plan could not be built');
    case 'get_channel_failed':
      return t('Channel selection failed');
    case 'request_failed_before_upstream':
      return t('The request failed before reaching upstream');
    case 'pricing_failed':
      return t('Pricing failed before the upstream request');
    case 'billing_reserve_failed':
      return t('Quota reservation failed');
    case 'request_body_unavailable':
      return t('The request body was unavailable');
    case 'local_task_error':
      return t('A local task error stopped retrying');
    case 'task_retry_not_safe':
      return t('The task could not be retried safely');
    default:
      return reason;
  }
}

function channelLabel(attempt) {
  return attempt.channel_name
    ? `${attempt.channel_name} #${attempt.channel_id}`
    : `#${attempt.channel_id}`;
}

function AttemptIcon({ outcome }) {
  if (outcome === 'succeeded') {
    return <CheckCircle2 size={15} color='var(--semi-color-success)' />;
  }
  if (outcome === 'failed') {
    return <AlertTriangle size={15} color='var(--semi-color-danger)' />;
  }
  return <CircleSlash2 size={15} color='var(--semi-color-warning)' />;
}

export default function RoutingDiagnostics({ routing }) {
  const { t } = useTranslation();
  if (!routing) return null;

  const planned = Array.isArray(routing.planned) ? routing.planned : [];
  const attempts = Array.isArray(routing.attempts) ? routing.attempts : [];
  const summary = [
    { key: t('Routing'), value: getRoutingModeLabel(t, routing.mode) },
    routing.basis && { key: t('Routing basis'), value: routing.basis },
    Array.isArray(routing.groups) &&
      routing.groups.length > 0 && {
        key: t('Candidate groups'),
        value: routing.groups.join(' → '),
      },
    routing.final_group && {
      key: t('Final group'),
      value: routing.final_group,
    },
    routing.final_channel_id > 0 && {
      key: t('Final channel'),
      value: routing.final_channel_name
        ? `${routing.final_channel_name} #${routing.final_channel_id}`
        : `#${routing.final_channel_id}`,
    },
    routing.final_stop_reason && {
      key: t('Final stop reason'),
      value: `${getRoutingReasonLabel(t, routing.final_stop_reason)} (${routing.final_stop_reason})`,
    },
  ].filter(Boolean);

  return (
    <div className='space-y-3 mt-2'>
      <Descriptions data={summary} row />

      {planned.length > 0 && (
        <div>
          <Text strong>
            {t('Planned route')} ({planned.length})
            {routing.planned_truncated ? ` · ${t('Truncated')}` : ''}
          </Text>
          <div
            style={{
              border: '1px solid var(--semi-color-border)',
              borderRadius: 6,
              marginTop: 8,
              padding: '0 10px',
            }}
          >
            {planned.map((attempt, index) => (
              <div
                key={`${attempt.group}-${attempt.channel_id}-${index}`}
                style={{
                  display: 'grid',
                  gridTemplateColumns: '28px minmax(0, 1fr)',
                  gap: 8,
                  padding: '9px 0',
                  borderBottom:
                    index === planned.length - 1
                      ? 'none'
                      : '1px solid var(--semi-color-border)',
                }}
              >
                <Text type='tertiary' code>
                  {index + 1}
                </Text>
                <div style={{ minWidth: 0 }}>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Text strong>{channelLabel(attempt)}</Text>
                    <Text type='tertiary' code>
                      {t('Priority')} {attempt.priority}
                    </Text>
                    <Tag color='grey' size='small'>
                      {attempt.explicit ? t('Configured') : t('Inherited')}
                    </Tag>
                  </div>
                  <Text type='tertiary' size='small' code>
                    {attempt.group} · {attempt.model}
                  </Text>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div>
        <Text strong>
          {t('Attempt chain')} ({attempts.length})
          {routing.attempts_truncated ? ` · ${t('Truncated')}` : ''}
        </Text>
        {attempts.length === 0 ? (
          <div className='mt-2'>
            <Text type='tertiary'>{t('No channel attempt was recorded.')}</Text>
          </div>
        ) : (
          <div
            style={{
              border: '1px solid var(--semi-color-border)',
              borderRadius: 6,
              marginTop: 8,
              padding: '0 10px',
            }}
          >
            {attempts.map((attempt, index) => {
              const retryReason =
                attempt.retry_stop_reason || attempt.retry_reason;
              return (
                <div
                  key={`${attempt.sequence}-${attempt.channel_id}-${index}`}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '28px minmax(0, 1fr)',
                    gap: 8,
                    padding: '10px 0',
                    borderBottom:
                      index === attempts.length - 1
                        ? 'none'
                        : '1px solid var(--semi-color-border)',
                  }}
                >
                  <div className='flex flex-col items-center gap-1'>
                    <AttemptIcon outcome={attempt.outcome} />
                    <Text type='tertiary' size='small' code>
                      {attempt.sequence}
                    </Text>
                  </div>
                  <div style={{ minWidth: 0 }} className='space-y-1'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <Text strong>{channelLabel(attempt)}</Text>
                      <Tag
                        color={
                          attempt.outcome === 'succeeded'
                            ? 'green'
                            : attempt.outcome === 'failed'
                              ? 'red'
                              : 'orange'
                        }
                        size='small'
                      >
                        {t(
                          attempt.outcome === 'succeeded'
                            ? 'Succeeded'
                            : attempt.outcome === 'failed'
                              ? 'Failed'
                              : 'Skipped',
                        )}
                      </Tag>
                      {attempt.status_code > 0 && (
                        <Tag
                          color={attempt.status_code >= 400 ? 'red' : 'grey'}
                          size='small'
                        >
                          {attempt.status_code}
                        </Tag>
                      )}
                      <Text type='tertiary' size='small' code>
                        {attempt.duration_ms || 0} ms
                      </Text>
                    </div>
                    <Text type='tertiary' size='small' code>
                      {attempt.group} · {attempt.phase}
                      {attempt.upstream_model
                        ? ` · ${attempt.upstream_model}`
                        : ''}
                      {attempt.request_format
                        ? ` · ${attempt.request_format}`
                        : ''}
                    </Text>
                    {(attempt.error_type || attempt.error_code) && (
                      <div>
                        <Text size='small' code>
                          {[attempt.error_type, attempt.error_code]
                            .filter(Boolean)
                            .join(' · ')}
                        </Text>
                      </div>
                    )}
                    {attempt.error_message && (
                      <div>
                        <Text type='danger' size='small'>
                          {attempt.error_message}
                        </Text>
                      </div>
                    )}
                    {attempt.upstream_request_id && (
                      <div>
                        <Text type='tertiary' size='small'>
                          {t('Upstream Request ID')}:{' '}
                          {attempt.upstream_request_id}
                        </Text>
                      </div>
                    )}
                    {attempt.retry_decision && (
                      <div>
                        <Text type='tertiary' size='small'>
                          {t('Retry decision')}:{' '}
                        </Text>
                        <Text size='small'>
                          {t(
                            attempt.retry_decision === 'retry'
                              ? 'Retry'
                              : attempt.retry_decision === 'complete'
                                ? 'Complete'
                                : 'Stop',
                          )}
                          {retryReason
                            ? ` · ${getRoutingReasonLabel(t, retryReason)}`
                            : ''}
                        </Text>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
