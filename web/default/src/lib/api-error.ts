/*
Copyright (C) 2023-2026 QuantumNous

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
import i18next from 'i18next'

const NETWORK_ERROR_KEY = 'Network connection failed or server not responding'

type UnknownRecord = Record<string, unknown>

export type ApiErrorDetails = {
  errorCode?: string
  errorMessage: string
}

function asRecord(value: unknown): UnknownRecord | undefined {
  return typeof value === 'object' && value !== null
    ? (value as UnknownRecord)
    : undefined
}

function asNonEmptyString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined
}

function getResponseData(error: unknown): unknown {
  const response = asRecord(asRecord(error)?.response)
  return response && 'data' in response ? response.data : error
}

function getErrorCode(data: unknown): string | undefined {
  const record = asRecord(data)
  const code =
    asRecord(record?.error)?.code ?? record?.errorCode ?? record?.code
  if (typeof code === 'string' && code) return code
  if (typeof code === 'number') return String(code)
  return undefined
}

function getDataMessage(data: unknown): string | undefined {
  const record = asRecord(data)
  const nestedError = record?.error
  const nestedMessage = asNonEmptyString(asRecord(nestedError)?.message)
  if (nestedMessage) return nestedMessage

  const message = asNonEmptyString(record?.message)
  if (message) return message

  const stringError = asNonEmptyString(nestedError)
  if (stringError) return stringError

  return asNonEmptyString(data)
}

export function extractApiErrorDetails(
  error: unknown,
  fallbackKey = NETWORK_ERROR_KEY
): ApiErrorDetails {
  const data = getResponseData(error)
  const dataMessage = getDataMessage(data)
  const outerMessage =
    data === error ? undefined : asNonEmptyString(asRecord(error)?.message)

  return {
    errorCode: getErrorCode(data),
    errorMessage:
      dataMessage ||
      outerMessage ||
      i18next.t(fallbackKey, { lng: i18next.resolvedLanguage }),
  }
}

export function extractApiErrorMessage(
  error: unknown,
  fallbackKey?: string
): string {
  return extractApiErrorDetails(error, fallbackKey).errorMessage
}
