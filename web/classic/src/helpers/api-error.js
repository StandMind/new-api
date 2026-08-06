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

import i18n from '../i18n/i18n';
import { normalizeLanguage } from '../i18n/language';

const NETWORK_ERROR_KEY = '网络连接失败或服务器无响应';

const asObject = (value) =>
  typeof value === 'object' && value !== null ? value : null;

const asNonEmptyString = (value) =>
  typeof value === 'string' && value.trim() ? value : null;

const getResponseData = (error) => {
  const response = asObject(asObject(error)?.response);
  return response && Object.prototype.hasOwnProperty.call(response, 'data')
    ? response.data
    : error;
};

const getDataMessage = (data) => {
  const record = asObject(data);
  const nestedError = record?.error;
  return (
    asNonEmptyString(asObject(nestedError)?.message) ||
    asNonEmptyString(record?.message) ||
    asNonEmptyString(nestedError) ||
    asNonEmptyString(data)
  );
};

const getErrorCode = (data) => {
  const record = asObject(data);
  const code =
    asObject(record?.error)?.code ?? record?.errorCode ?? record?.code;
  if (typeof code === 'string' && code) return code;
  if (typeof code === 'number') return String(code);
  return null;
};

export const extractApiErrorDetails = (
  error,
  fallbackKey = NETWORK_ERROR_KEY,
) => {
  const data = getResponseData(error);
  const dataMessage = getDataMessage(data);
  const outerMessage =
    data === error ? null : asNonEmptyString(asObject(error)?.message);

  return {
    errorCode: getErrorCode(data),
    errorMessage: dataMessage || outerMessage || i18n.t(fallbackKey),
  };
};

export const extractApiErrorMessage = (error, fallbackKey) =>
  extractApiErrorDetails(error, fallbackKey).errorMessage;

export const getLanguageHeaders = () => {
  const language = normalizeLanguage(
    i18n.resolvedLanguage || i18n.language || 'en',
  );
  return language ? { 'Accept-Language': language } : {};
};
