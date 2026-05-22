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

export function ApiDebugPanel({ debug }: ApiDebugPanelProps) {
  const { t } = useTranslation()
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState(debug?.model ?? '')
  const [body, setBody] = useState(formatBody(debug?.body))
  const [response, setResponse] = useState('')
  const [isSending, setIsSending] = useState(false)

  useEffect(() => {
    setModel(debug?.model ?? '')
    setBody(formatBody(debug?.body))
    setResponse('')
  }, [debug])

  const endpoint = useMemo(() => {
    if (!debug) return ''
    if (debug.path_template && model.trim()) {
      return debug.path_template.replace(
        '{model}',
        encodeURIComponent(model.trim())
      )
    }
    return debug.path
  }, [debug, model])

  if (!debug) return null

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

      const result = await fetch(endpoint, {
        method: debug.method,
        headers,
        body: debug.method === 'GET' ? undefined : JSON.stringify(payload),
      })
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
              onChange={(event) => setModel(event.target.value)}
            />
          </Label>
        ) : null}
        <Label className='grid gap-2'>
          <span>{t('Endpoint')}</span>
          <Input value={endpoint} readOnly />
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
            onChange={(event) => setBody(event.target.value)}
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
