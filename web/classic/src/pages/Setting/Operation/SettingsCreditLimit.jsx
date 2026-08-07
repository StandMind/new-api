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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  InputNumber,
  Radio,
  RadioGroup,
  Row,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { Save } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';

const defaultInvitationSetting = {
  mode: 'disabled',
  fixed_inviter_quota: 0,
  fixed_invitee_quota: 0,
  rebate_bps: 500,
  rebate_topup_count: 3,
};

export default function SettingsCreditLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [invitationLoading, setInvitationLoading] = useState(true);
  const [inputs, setInputs] = useState({
    QuotaForNewUser: '',
    PreConsumedQuota: '',
    'quota_setting.enable_free_model_pre_consume': true,
  });
  const [inputsRow, setInputsRow] = useState(inputs);
  const [invitation, setInvitation] = useState(defaultInvitationSetting);
  const [savedInvitation, setSavedInvitation] = useState(
    defaultInvitationSetting,
  );
  const refForm = useRef();
  const complianceConfirmed =
    props.options?.['payment_setting.compliance_confirmed'] === true ||
    props.options?.['payment_setting.compliance_confirmed'] === 'true';

  const invitationDirty = useMemo(
    () => JSON.stringify(invitation) !== JSON.stringify(savedInvitation),
    [invitation, savedInvitation],
  );
  const rewardsEnabled =
    invitation.mode === 'rebate' ||
    (invitation.mode === 'fixed' &&
      (invitation.fixed_inviter_quota > 0 ||
        invitation.fixed_invitee_quota > 0));
  const blockedByCompliance = rewardsEnabled && !complianceConfirmed;

  const updateInvitation = (key, value) => {
    setInvitation((current) => ({ ...current, [key]: value }));
  };

  async function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length && !invitationDirty) {
      return showWarning(t('你似乎并没有修改什么'));
    }
    if (blockedByCompliance) {
      return showError(
        t('设置非零邀请奖励额度前，需要先在支付设置中确认合规声明。'),
      );
    }

    setLoading(true);
    try {
      const requests = updateArray.map((item) =>
        API.put('/api/option/', {
          key: item.key,
          value:
            typeof inputs[item.key] === 'boolean'
              ? String(inputs[item.key])
              : inputs[item.key],
        }),
      );
      if (invitationDirty) {
        requests.push(API.put('/api/option/invitation', invitation));
      }
      const responses = await Promise.all(requests);
      if (responses.some((response) => response?.data?.success === false)) {
        throw new Error(
          responses.find((response) => response?.data?.success === false)?.data
            ?.message || t('保存失败，请重试'),
        );
      }

      const invitationResponse = responses.find(
        (response) => response?.config?.url === '/api/option/invitation',
      );
      if (invitationResponse?.data?.data) {
        setInvitation(invitationResponse.data.data);
        setSavedInvitation(invitationResponse.data.data);
      }
      setInputsRow(structuredClone(inputs));
      showSuccess(t('保存成功'));
      props.refresh();
    } catch (error) {
      showError(error?.message || t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const currentInputs = {};
    for (const key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current?.setValues(currentInputs);
  }, [props.options]);

  useEffect(() => {
    let active = true;
    API.get('/api/option/invitation')
      .then((response) => {
        if (!active) return;
        if (!response?.data?.success || !response.data.data) {
          throw new Error(response?.data?.message || 'invalid response');
        }
        setInvitation(response.data.data);
        setSavedInvitation(response.data.data);
      })
      .catch(() => {
        if (active) showError(t('请求失败'));
      })
      .finally(() => {
        if (active) setInvitationLoading(false);
      });
    return () => {
      active = false;
    };
  }, [t]);

  return (
    <Spin spinning={loading || invitationLoading}>
      <Form
        values={inputs}
        getFormApi={(formAPI) => (refForm.current = formAPI)}
        style={{ marginBottom: 15 }}
      >
        <Form.Section text={t('邀请方案')}>
          <Typography.Text type='tertiary'>
            {t('切换方案时会保留固定奖励和充值返利的全部参数。')}
          </Typography.Text>
          <div className='mt-3 mb-4'>
            <RadioGroup
              type='button'
              value={invitation.mode}
              onChange={(event) => updateInvitation('mode', event.target.value)}
            >
              <Radio value='disabled'>{t('已关闭')}</Radio>
              <Radio value='fixed'>{t('固定注册奖励')}</Radio>
              <Radio value='rebate'>{t('充值比例返利')}</Radio>
            </RadioGroup>
          </div>

          {invitation.mode === 'fixed' && (
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8}>
                <InputNumber
                  className='w-full'
                  prefix={t('邀请人额度')}
                  min={0}
                  value={invitation.fixed_inviter_quota}
                  onChange={(value) =>
                    updateInvitation(
                      'fixed_inviter_quota',
                      Math.max(0, Number(value) || 0),
                    )
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8}>
                <InputNumber
                  className='w-full'
                  prefix={t('被邀请人额度')}
                  min={0}
                  value={invitation.fixed_invitee_quota}
                  onChange={(value) =>
                    updateInvitation(
                      'fixed_invitee_quota',
                      Math.max(0, Number(value) || 0),
                    )
                  }
                />
              </Col>
            </Row>
          )}

          {invitation.mode === 'rebate' && (
            <>
              <Row gutter={16}>
                <Col xs={24} sm={12} md={8}>
                  <InputNumber
                    className='w-full'
                    prefix={t('返利比例')}
                    suffix='%'
                    min={0.01}
                    max={100}
                    step={0.01}
                    value={invitation.rebate_bps / 100}
                    onChange={(value) =>
                      updateInvitation(
                        'rebate_bps',
                        Math.min(
                          10000,
                          Math.max(0, Math.round((Number(value) || 0) * 100)),
                        ),
                      )
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <InputNumber
                    className='w-full'
                    prefix={t('有效充值次数')}
                    min={1}
                    max={100}
                    step={1}
                    value={invitation.rebate_topup_count}
                    onChange={(value) =>
                      updateInvitation(
                        'rebate_topup_count',
                        Math.min(
                          100,
                          Math.max(0, Math.trunc(Number(value) || 0)),
                        ),
                      )
                    }
                  />
                </Col>
              </Row>
              <Typography.Text type='tertiary' size='small'>
                {t(
                  '邀请关系建立后的成功钱包充值都会占用次数，包括其他方案启用期间的充值。',
                )}
              </Typography.Text>
            </>
          )}

          {invitation.mode === 'disabled' && (
            <Typography.Text type='tertiary'>
              {t('新邀请奖励已关闭，已保存的两套参数不会被清空。')}
            </Typography.Text>
          )}

          {blockedByCompliance && (
            <Banner
              type='warning'
              description={t(
                '设置非零邀请奖励额度前，需要先在支付设置中确认合规声明。',
              )}
              closeIcon={null}
              className='!rounded-lg mt-3'
            />
          )}
        </Form.Section>

        <Form.Section text={t('额度设置')}>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                label={t('新用户初始额度')}
                field='QuotaForNewUser'
                step={1}
                min={0}
                suffix='Token'
                onChange={(value) =>
                  setInputs({ ...inputs, QuotaForNewUser: String(value) })
                }
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                label={t('请求预扣费额度')}
                field='PreConsumedQuota'
                step={1}
                min={0}
                suffix='Token'
                extraText={t('请求结束后多退少补')}
                onChange={(value) =>
                  setInputs({ ...inputs, PreConsumedQuota: String(value) })
                }
              />
            </Col>
          </Row>
          <Row>
            <Col>
              <Form.Switch
                label={t('对免费模型启用预消耗')}
                field='quota_setting.enable_free_model_pre_consume'
                extraText={t(
                  '开启后，对免费模型（倍率为0，或者价格为0）的模型也会预消耗额度',
                )}
                onChange={(value) =>
                  setInputs({
                    ...inputs,
                    'quota_setting.enable_free_model_pre_consume': value,
                  })
                }
              />
            </Col>
          </Row>
        </Form.Section>

        <Button size='default' icon={<Save size={15} />} onClick={onSubmit}>
          {t('保存额度与邀请设置')}
        </Button>
      </Form>
    </Spin>
  );
}
