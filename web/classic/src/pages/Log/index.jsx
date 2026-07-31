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
import { Tabs, TabPane } from '@douyinfe/semi-ui';
import UsageLogsTable from '../../components/table/usage-logs';
import RequestDetailsTable from '../../components/table/request-details';
import { isAdmin } from '../../helpers';
import { useTranslation } from 'react-i18next';

const Token = () => {
  const { t } = useTranslation();
  return (
    <div className='mt-[60px] px-2'>
      {isAdmin() ? (
        <Tabs type='line'>
          <TabPane tab={t('使用日志')} itemKey='usage'>
            <UsageLogsTable />
          </TabPane>
          <TabPane tab={t('请求详情')} itemKey='request-details'>
            <RequestDetailsTable />
          </TabPane>
        </Tabs>
      ) : (
        <UsageLogsTable />
      )}
    </div>
  );
};

export default Token;
