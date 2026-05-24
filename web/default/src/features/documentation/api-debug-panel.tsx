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
import { useEffect, useMemo, useState } from 'react'
import { Play, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import type { DocumentationDebugExample } from './types'

type ApiDebugPanelProps = {
  debug?: DocumentationDebugExample
}

function formatBody(value: unknown) {
  if (value === undefined || value === null || value === '') return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

function formatResponse(value: unknown) {
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

function getBrowserOrigin() {
  if (typeof window === 'undefined') return ''
  return window.location.origin
}

function normalizeEndpointPath(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return ''

  try {
    const parsed = new URL(trimmed)
    return `${parsed.pathname}${parsed.search}${parsed.hash}` || '/'
  } catch {
    return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  }
}

function buildRequestUrl(origin: string, endpointPath: string) {
  const normalizedPath = normalizeEndpointPath(endpointPath)
  if (!origin) return normalizedPath
  return `${origin.replace(/\/+$/, '')}${normalizedPath}`
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function findJsonStringEnd(value: string, start: number) {
  let escaped = false
  for (let index = start + 1; index < value.length; index += 1) {
    const char = value[index]
    if (escaped) {
      escaped = false
      continue
    }
    if (char === '\\') {
      escaped = true
      continue
    }
    if (char === '"') return index
  }
  return -1
}

function skipJsonWhitespace(value: string, start: number) {
  let index = start
  while (index < value.length && /\s/.test(value[index])) index += 1
  return index
}

function findJsonValueEnd(value: string, start: number) {
  if (value[start] === '"') {
    const end = findJsonStringEnd(value, start)
    return end >= 0 ? end + 1 : -1
  }

  let depth = 0
  let inString = false
  let escaped = false
  for (let index = start; index < value.length; index += 1) {
    const char = value[index]
    if (inString) {
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === '"') {
        inString = false
      }
      continue
    }

    if (char === '"') {
      inString = true
      continue
    }
    if (char === '{' || char === '[') {
      depth += 1
      continue
    }
    if (char === '}' || char === ']') {
      if (depth === 0) return index
      depth -= 1
      if (depth === 0) return index + 1
      continue
    }
    if (char === ',' && depth === 0) return index
  }

  return value.length
}

function findTopLevelJsonPropertyValue(
  value: string,
  propertyName: string
): { start: number; end: number } | undefined {
  let index = skipJsonWhitespace(value, 0)
  if (value[index] !== '{') return undefined
  index += 1

  while (index < value.length) {
    index = skipJsonWhitespace(value, index)
    if (value[index] === '}') return undefined
    if (value[index] !== '"') return undefined

    const keyStart = index
    const keyEnd = findJsonStringEnd(value, keyStart)
    if (keyEnd < 0) return undefined

    let key: unknown
    try {
      key = JSON.parse(value.slice(keyStart, keyEnd + 1))
    } catch {
      return undefined
    }

    index = skipJsonWhitespace(value, keyEnd + 1)
    if (value[index] !== ':') return undefined
    index = skipJsonWhitespace(value, index + 1)

    const valueStart = index
    const valueEnd = findJsonValueEnd(value, valueStart)
    if (valueEnd < 0) return undefined
    if (key === propertyName) {
      return {
        start: valueStart,
        end: valueEnd,
      }
    }

    index = skipJsonWhitespace(value, valueEnd)
    if (value[index] === ',') {
      index += 1
      continue
    }
    if (value[index] === '}') return undefined
    return undefined
  }

  return undefined
}

function applyModelToBody(
  payload: unknown,
  modelName: string,
  debug?: DocumentationDebugExample
) {
  const selectedModel = modelName.trim()
  if (!debug?.model || !isObjectRecord(payload)) return payload
  if (
    !Object.prototype.hasOwnProperty.call(payload, 'model') &&
    debug.path_template
  ) {
    return payload
  }
  return {
    ...payload,
    model: selectedModel,
  }
}

function updateBodyTextWithModel(
  bodyText: string,
  modelName: string,
  debug?: DocumentationDebugExample
) {
  if (!bodyText.trim()) return bodyText
  if (!debug?.model) return bodyText

  const modelRange = findTopLevelJsonPropertyValue(bodyText, 'model')
  if (modelRange) {
    return `${bodyText.slice(0, modelRange.start)}${JSON.stringify(
      modelName.trim()
    )}${bodyText.slice(modelRange.end)}`
  }

  try {
    const payload = JSON.parse(bodyText)
    const nextPayload = applyModelToBody(payload, modelName, debug)
    return nextPayload === payload ? bodyText : formatBody(nextPayload)
  } catch {
    return bodyText
  }
}

function getModelFromBodyText(bodyText: string) {
  if (!bodyText.trim()) return undefined

  try {
    const payload = JSON.parse(bodyText)
    if (isObjectRecord(payload) && typeof payload.model === 'string') {
      return payload.model
    }
  } catch {
    return undefined
  }

  return undefined
}

function isStreamRequested(payload: unknown, endpointPath: string) {
  if (isObjectRecord(payload) && payload.stream === true) return true
  return endpointPath.toLowerCase().includes('stream')
}

function shouldReadStreamResponse(
  payload: unknown,
  endpointPath: string,
  response: Response
) {
  const contentType = response.headers.get('content-type')?.toLowerCase() ?? ''
  if (contentType.includes('text/event-stream')) return true
  if (contentType.includes('application/json')) return false
  return isStreamRequested(payload, endpointPath)
}

async function readStreamingText(
  response: Response,
  onChunk: (chunk: string) => void
) {
  const reader = response.body?.getReader()
  if (!reader) return response.text()

  const decoder = new TextDecoder()
  let text = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    const chunk = decoder.decode(value, { stream: true })
    if (!chunk) continue
    text += chunk
    onChunk(chunk)
  }

  const tail = decoder.decode()
  if (tail) {
    text += tail
    onChunk(tail)
  }

  return text
}

export function ApiDebugPanel({ debug }: ApiDebugPanelProps) {
  const { t } = useTranslation()
  const apiOrigin = useMemo(getBrowserOrigin, [])
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState(debug?.model ?? '')
  const [endpointPath, setEndpointPath] = useState('')
  const [body, setBody] = useState(formatBody(debug?.body))
  const [response, setResponse] = useState('')
  const [isSending, setIsSending] = useState(false)

  useEffect(() => {
    setModel(debug?.model ?? '')
    setBody(formatBody(debug?.body))
    setResponse('')
  }, [debug])

  const defaultEndpointPath = useMemo(() => {
    if (!debug) return ''
    if (debug.path_template && model.trim()) {
      return debug.path_template.replace(
        '{model}',
        encodeURIComponent(model.trim())
      )
    }
    return debug.path
  }, [debug, model])

  useEffect(() => {
    setEndpointPath(normalizeEndpointPath(defaultEndpointPath))
  }, [defaultEndpointPath])

  const requestUrl = useMemo(
    () => buildRequestUrl(apiOrigin, endpointPath),
    [apiOrigin, endpointPath]
  )

  if (!debug) return null

  const handleModelChange = (value: string) => {
    setModel(value)
    setBody((currentBody) => updateBodyTextWithModel(currentBody, value, debug))
  }

  const handleBodyChange = (value: string) => {
    setBody(value)
    const nextModel = getModelFromBodyText(value)
    if (nextModel !== undefined) {
      setModel(nextModel)
    }
  }

  const sendRequest = async () => {
    if (debug.auth && !apiKey.trim()) {
      setResponse(t('Enter an API key first.'))
      return
    }

    let payload: unknown
    if (debug.method !== 'GET' && body.trim()) {
      try {
        payload = JSON.parse(body)
      } catch {
        setResponse(t('The request body is not valid JSON.'))
        return
      }
    }
    payload = applyModelToBody(payload, model, debug)

    setIsSending(true)
    setResponse('')

    try {
      const headers: Record<string, string> = {
        ...(debug.headers ?? {}),
      }
      if (debug.method !== 'GET') {
        headers['Content-Type'] = 'application/json'
      }
      if (debug.auth === 'bearer') {
        headers.Authorization = `Bearer ${apiKey.trim()}`
      }
      if (debug.auth === 'anthropic') {
        headers['x-api-key'] = apiKey.trim()
      }

      const result = await fetch(requestUrl, {
        method: debug.method,
        headers,
        body: debug.method === 'GET' ? undefined : JSON.stringify(payload),
      })

      if (shouldReadStreamResponse(payload, endpointPath, result)) {
        let hasChunk = false
        const text = await readStreamingText(result, (chunk) => {
          hasChunk = true
          setResponse((current) => `${current}${chunk}`)
        })

        if (!hasChunk) {
          setResponse(text)
        }
        return
      }

      const text = await result.text()
      try {
        setResponse(formatResponse(JSON.parse(text)))
      } catch {
        setResponse(text)
      }
    } catch (error) {
      setResponse(
        `${t('Request failed')}: ${
          error instanceof Error ? error.message : String(error)
        }`
      )
    } finally {
      setIsSending(false)
    }
  }

  return (
    <section className='border-border bg-muted/30 space-y-4 rounded-lg border p-4'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
        <div className='space-y-1'>
          <h3 className='text-base font-semibold'>{t('API tester')}</h3>
          <p className='text-muted-foreground text-sm'>
            {t(
              'This example calls the endpoint for the current document. The API key is only used in this browser page and is not stored locally.'
            )}
          </p>
        </div>
        <Button type='button' onClick={sendRequest} disabled={isSending}>
          <Play data-icon='inline-start' />
          {isSending ? t('Sending') : t('Send request')}
        </Button>
      </div>

      <div className='grid gap-4 md:grid-cols-2'>
        <Label className='grid gap-2'>
          <span>{t('API key')}</span>
          <Input
            value={apiKey}
            onChange={(event) => setApiKey(event.target.value)}
            placeholder='sk-...'
            type='password'
            autoComplete='off'
          />
        </Label>
        {debug.model ? (
          <Label className='grid gap-2'>
            <span>{t('Model')}</span>
            <Input
              value={model}
              onChange={(event) => handleModelChange(event.target.value)}
            />
          </Label>
        ) : null}
        <Label className='grid gap-2'>
          <span>{t('Endpoint')}</span>
          <div className='flex min-w-0 rounded-md'>
            <div className='border-input bg-muted text-muted-foreground flex max-w-[45%] min-w-0 shrink-0 items-center rounded-l-md border border-r-0 px-3 text-sm sm:max-w-[55%]'>
              <span className='truncate'>{apiOrigin}</span>
            </div>
            <Input
              value={endpointPath}
              onChange={(event) => setEndpointPath(event.target.value)}
              className='rounded-l-none'
              spellCheck={false}
            />
          </div>
        </Label>
        <Label className='grid gap-2'>
          <span>{t('Method')}</span>
          <Input value={debug.method} readOnly />
        </Label>
      </div>

      {debug.method !== 'GET' ? (
        <Label className='grid gap-2'>
          <span>{t('Request body')}</span>
          <Textarea
            className='min-h-44 font-mono text-xs'
            spellCheck={false}
            value={body}
            onChange={(event) => handleBodyChange(event.target.value)}
          />
        </Label>
      ) : null}

      <div className='space-y-2'>
        <div className='flex items-center justify-between gap-3'>
          <span className='text-sm font-medium'>{t('Response')}</span>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={() => setResponse('')}
          >
            <Trash2 data-icon='inline-start' />
            {t('Clear response')}
          </Button>
        </div>
        <pre className='bg-background border-border min-h-24 overflow-x-auto rounded-md border p-3 text-xs whitespace-pre-wrap'>
          {response || t('No response yet.')}
        </pre>
      </div>
    </section>
  )
}
