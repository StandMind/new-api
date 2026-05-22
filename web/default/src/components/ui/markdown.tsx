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
import {
  Children,
  cloneElement,
  isValidElement,
  useEffect,
  useState,
  type HTMLAttributes,
  type ReactElement,
  type ReactNode,
} from 'react'
import {
  Braces,
  Check,
  Clipboard,
  Code2,
  Hash,
  Info,
  Lightbulb,
  OctagonAlert,
  Rows3,
  SquareTerminal,
  TriangleAlert,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown, { type Components } from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import remarkDirective from 'remark-directive'
import remarkGfm from 'remark-gfm'
import { createHighlighterCore, type HighlighterCore } from 'shiki/core'
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript'
import bash from 'shiki/langs/bash.mjs'
import css from 'shiki/langs/css.mjs'
import go from 'shiki/langs/go.mjs'
import html from 'shiki/langs/html.mjs'
import javascript from 'shiki/langs/javascript.mjs'
import json from 'shiki/langs/json.mjs'
import jsx from 'shiki/langs/jsx.mjs'
import markdown from 'shiki/langs/markdown.mjs'
import python from 'shiki/langs/python.mjs'
import tsx from 'shiki/langs/tsx.mjs'
import typescript from 'shiki/langs/typescript.mjs'
import yaml from 'shiki/langs/yaml.mjs'
import githubDark from 'shiki/themes/github-dark.mjs'
import githubLight from 'shiki/themes/github-light.mjs'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { remarkApiDirectives } from '@/components/ui/markdown-directives'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface MarkdownProps {
  children: string
  className?: string
}

export type MarkdownHeading = {
  depth: number
  id: string
  title: string
}

const calloutKinds = ['note', 'tip', 'important', 'warning', 'caution'] as const

type CalloutKind = (typeof calloutKinds)[number]

const calloutPattern = /^\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/i
const endpointPattern =
  /^(?:Endpoint|接口|端点):\s*(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(.+)$/i

type EndpointInfo = {
  method: string
  path: string
  description?: string
  title?: string
}

type TableKind = 'default' | 'parameters' | 'status' | 'comparison' | 'clients'

type ApiDirectiveProps = HTMLAttributes<HTMLElement> & {
  description?: string
  method?: string
  path?: string
  status?: string
  title?: string
}

const languageAliases: Record<string, string> = {
  bash: 'shellscript',
  cjs: 'javascript',
  console: 'shellscript',
  js: 'javascript',
  jsonc: 'json',
  jsx: 'jsx',
  md: 'markdown',
  plaintext: 'text',
  py: 'python',
  rb: 'ruby',
  sh: 'shellscript',
  shell: 'shellscript',
  ts: 'typescript',
  tsx: 'tsx',
  yml: 'yaml',
}

const highlightedLanguages = new Set([
  'bash',
  'css',
  'go',
  'html',
  'javascript',
  'json',
  'jsx',
  'markdown',
  'python',
  'shell',
  'shellscript',
  'sh',
  'text',
  'tsx',
  'typescript',
  'yaml',
  'zsh',
])

let highlighterPromise: Promise<HighlighterCore> | undefined

function getHighlighter() {
  highlighterPromise ??= createHighlighterCore({
    engine: createJavaScriptRegexEngine(),
    langs: [
      bash,
      css,
      go,
      html,
      javascript,
      json,
      jsx,
      markdown,
      python,
      tsx,
      typescript,
      yaml,
    ],
    themes: [githubLight, githubDark],
  })

  return highlighterPromise
}

function stripMarkdownInline(value: string) {
  return value
    .replace(/`([^`]+)`/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/[*_~]/g, '')
    .replace(/<[^>]+>/g, '')
    .trim()
}

function slugifyHeading(value: string) {
  const slug = stripMarkdownInline(value)
    .normalize('NFKD')
    .toLowerCase()
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^\p{L}\p{N}\s-]/gu, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')

  return slug || 'section'
}

function uniqueHeadingId(title: string, counts: Map<string, number>) {
  const base = slugifyHeading(title)
  const count = counts.get(base) ?? 0
  counts.set(base, count + 1)
  return count === 0 ? base : `${base}-${count + 1}`
}

export function extractMarkdownHeadings(markdown: string): MarkdownHeading[] {
  const counts = new Map<string, number>()
  const headings: MarkdownHeading[] = []
  const headingPattern = /^(#{1,4})\s+(.+?)\s*#*\s*$/gm
  const markdownWithoutCode = markdown.replace(/```[\s\S]*?```/g, '')
  let match: RegExpExecArray | null

  while ((match = headingPattern.exec(markdownWithoutCode)) !== null) {
    const title = stripMarkdownInline(match[2] ?? '')
    if (!title) continue
    const id = uniqueHeadingId(title, counts)
    const depth = match[1]?.length ?? 2
    if (depth < 2) continue
    headings.push({
      depth,
      id,
      title,
    })
  }

  return headings
}

function extractNodeText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node)
  }
  if (Array.isArray(node)) {
    return node.map(extractNodeText).join('')
  }
  if (isValidElement<{ children?: ReactNode }>(node)) {
    return extractNodeText(node.props.children)
  }
  return ''
}

function parseEndpoint(children: ReactNode): EndpointInfo | undefined {
  const text = extractNodeText(children).replace(/\s+/g, ' ').trim()
  const match = text.match(endpointPattern)
  if (!match) return undefined

  return {
    method: (match[1] ?? '').toUpperCase(),
    path: (match[2] ?? '').trim(),
  }
}

function getMethodClassName(method: string) {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/70 dark:bg-emerald-950/30 dark:text-emerald-300'
    case 'POST':
      return 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/70 dark:bg-sky-950/30 dark:text-sky-300'
    case 'PUT':
    case 'PATCH':
      return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-300'
    case 'DELETE':
      return 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300'
    default:
      return 'border-border bg-muted text-muted-foreground'
  }
}

function getTableKind(children: ReactNode): TableKind {
  const text = extractNodeText(children).toLowerCase()

  if (
    /parameter|required|description|header|请求头|参数|必填|说明/.test(text)
  ) {
    return 'parameters'
  }
  if (/status|meaning|handling|状态码|含义|建议处理/.test(text)) {
    return 'status'
  }
  if (/client|provider|base url|客户端|填写|使用场景/.test(text)) {
    return 'clients'
  }
  if (/comparison|recommended|项目|对比|建议/.test(text)) {
    return 'comparison'
  }

  return 'default'
}

function getCodeBlockTitle(
  code: string,
  language: string,
  t: ReturnType<typeof useTranslation>['t']
) {
  if (language === 'shellscript' && /^\s*curl\b/.test(code)) {
    return t('Request')
  }
  if (language === 'json' && /"choices"|"usage"|"object"|"data"/.test(code)) {
    return t('Response')
  }
  return displayLanguage(language)
}

function stripCalloutMarker(children: ReactNode): ReactNode {
  let stripped = false

  function strip(node: ReactNode): ReactNode {
    if (stripped) return node
    if (typeof node === 'string') {
      const next = node.replace(calloutPattern, '')
      stripped = next !== node
      return next
    }
    if (Array.isArray(node)) {
      return node.map(strip)
    }
    if (isValidElement<{ children?: ReactNode }>(node)) {
      const element = node as ReactElement<{ children?: ReactNode }>
      return cloneElement(element, undefined, strip(element.props.children))
    }
    return node
  }

  return strip(children)
}

function getCalloutKind(children: ReactNode): CalloutKind | undefined {
  const match = extractNodeText(children).match(calloutPattern)
  const value = match?.[1]?.toLowerCase()
  return calloutKinds.find((kind) => kind === value)
}

function normalizeLanguage(value?: string) {
  const raw =
    value
      ?.replace(/^language-/, '')
      .trim()
      .toLowerCase() ?? ''
  const candidate = languageAliases[raw] ?? raw
  if (candidate && highlightedLanguages.has(candidate)) return candidate
  return 'text'
}

function inferLanguage(code: string) {
  const value = code.trim()
  if (!value) return 'text'
  if (/^[\[{]/.test(value)) return 'json'
  if (/^<[\w!/]/.test(value)) return 'html'
  if (/^(curl|npm|pnpm|bun|yarn|pip|python|node|export)\b/m.test(value)) {
    return 'shellscript'
  }
  if (/^(from|import)\s+\w+|def\s+\w+\(|print\(/m.test(value)) {
    return 'python'
  }
  if (/\b(const|let|var|async function|fetch\(|console\.log)\b/.test(value)) {
    return 'javascript'
  }
  return 'text'
}

function displayLanguage(value: string) {
  const names: Record<string, string> = {
    javascript: 'JavaScript',
    json: 'JSON',
    jsx: 'JSX',
    markdown: 'Markdown',
    python: 'Python',
    shellscript: 'Shell',
    text: 'Text',
    tsx: 'TSX',
    typescript: 'TypeScript',
    yaml: 'YAML',
  }
  return names[value] ?? value.toUpperCase()
}

function CodeBlock({ code, language }: { code: string; language?: string }) {
  const { t } = useTranslation()
  const lang = language ? normalizeLanguage(language) : inferLanguage(code)
  const title = getCodeBlockTitle(code, lang, t)
  const [html, setHtml] = useState('')
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let mounted = true

    getHighlighter()
      .then((highlighter) =>
        highlighter.codeToHtml(code, {
          lang,
          themes: {
            light: 'github-light',
            dark: 'github-dark',
          },
        })
      )
      .then((highlighted) => {
        if (mounted) setHtml(highlighted)
      })
      .catch(() => {
        if (mounted) setHtml('')
      })

    return () => {
      mounted = false
    }
  }, [code, lang])

  const copyCode = async () => {
    await navigator.clipboard?.writeText(code)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <figure className='not-prose group bg-card my-7 overflow-hidden rounded-lg border shadow-sm'>
      <figcaption className='bg-muted/45 text-muted-foreground flex h-11 items-center justify-between border-b px-3 text-xs'>
        <div className='flex min-w-0 items-center gap-3'>
          <span className='flex items-center gap-1.5' aria-hidden='true'>
            <span className='size-2 rounded-full bg-red-400/80' />
            <span className='size-2 rounded-full bg-amber-400/80' />
            <span className='size-2 rounded-full bg-emerald-400/80' />
          </span>
          <span className='truncate font-medium'>{title}</span>
          {title !== displayLanguage(lang) ? (
            <span className='border-border/70 bg-background/70 hidden rounded border px-1.5 py-0.5 font-mono text-[0.6875rem] sm:inline'>
              {displayLanguage(lang)}
            </span>
          ) : null}
        </div>
        <TooltipProvider delay={0}>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  className='text-muted-foreground hover:text-foreground'
                  aria-label={copied ? t('Copied') : t('Copy code')}
                  onClick={copyCode}
                />
              }
            >
              {copied ? <Check /> : <Clipboard />}
              <span className='sr-only'>
                {copied ? t('Copied') : t('Copy code')}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <p>{copied ? t('Copied') : t('Copy code')}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </figcaption>
      <div
        className={cn(
          'bg-background overflow-x-auto',
          '[&_.shiki]:m-0 [&_.shiki]:max-h-[34rem] [&_.shiki]:overflow-x-auto',
          '[&_.shiki]:!bg-transparent [&_.shiki]:p-4 [&_.shiki]:text-[0.8125rem] [&_.shiki]:leading-relaxed'
        )}
      >
        {html ? (
          <div dangerouslySetInnerHTML={{ __html: html }} />
        ) : (
          <pre className='m-0 max-h-[34rem] overflow-x-auto p-4 text-[0.8125rem] leading-relaxed'>
            <code>{code}</code>
          </pre>
        )}
      </div>
    </figure>
  )
}

function MarkdownEndpoint({ method, path, title, description }: EndpointInfo) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  const copyEndpoint = async () => {
    await navigator.clipboard?.writeText(`${method} ${path}`)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <section className='not-prose bg-card my-7 overflow-hidden rounded-lg border shadow-sm'>
      <div className='bg-muted/35 text-muted-foreground flex items-center justify-between border-b px-4 py-2.5 text-xs font-medium'>
        <div className='flex min-w-0 items-center gap-2'>
          <SquareTerminal className='size-3.5 shrink-0' />
          <span className='truncate'>{title || t('Endpoint')}</span>
        </div>
        <TooltipProvider delay={0}>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  className='text-muted-foreground hover:text-foreground'
                  aria-label={copied ? t('Copied') : t('Copy')}
                  onClick={copyEndpoint}
                />
              }
            >
              {copied ? <Check /> : <Clipboard />}
              <span className='sr-only'>
                {copied ? t('Copied') : t('Copy')}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <p>{copied ? t('Copied') : t('Copy')}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      <div className='flex min-w-0 flex-col gap-3 p-4 sm:flex-row sm:items-center'>
        <span
          className={cn(
            'inline-flex h-7 w-fit items-center rounded-md border px-2.5 font-mono text-xs font-semibold',
            getMethodClassName(method)
          )}
        >
          {method}
        </span>
        <code className='bg-muted text-foreground min-w-0 overflow-x-auto rounded-md px-3 py-2 font-mono text-sm leading-none'>
          {path}
        </code>
      </div>
      {description ? (
        <p className='text-muted-foreground m-0 border-t px-4 py-3 text-sm leading-6'>
          {description}
        </p>
      ) : null}
    </section>
  )
}

function ApiDirectiveEndpoint({
  description,
  method,
  path,
  title,
}: ApiDirectiveProps) {
  if (!method || !path) return null

  return (
    <MarkdownEndpoint
      method={method}
      path={path}
      title={title}
      description={description}
    />
  )
}

function ApiDirectivePanel({
  children,
  title,
  status,
  variant = 'default',
}: ApiDirectiveProps & {
  variant?: 'default' | 'parameters' | 'request' | 'response'
}) {
  const { t } = useTranslation()
  const labels = {
    default: title || t('Details'),
    parameters: title || t('Parameters'),
    request: title || t('Request'),
    response: title || t('Response'),
  }
  const icons = {
    default: <Info className='size-4' />,
    parameters: <Rows3 className='size-4' />,
    request: <Code2 className='size-4' />,
    response: <Braces className='size-4' />,
  }

  return (
    <section className='not-prose bg-card my-7 overflow-hidden rounded-lg border shadow-sm'>
      <header className='bg-muted/35 text-muted-foreground flex items-center justify-between border-b px-4 py-3 text-sm font-medium'>
        <div className='flex min-w-0 items-center gap-2'>
          {icons[variant]}
          <span className='truncate'>{labels[variant]}</span>
        </div>
        {status ? (
          <span
            className={cn(
              'rounded-md border px-2 py-0.5 font-mono text-xs',
              status.startsWith('2')
                ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/70 dark:bg-emerald-950/30 dark:text-emerald-300'
                : 'border-border bg-background text-muted-foreground'
            )}
          >
            {status}
          </span>
        ) : null}
      </header>
      <div className='api-doc-body px-4 py-4 text-sm leading-7 [&>*:first-child]:mt-0 [&>*:last-child]:mb-0'>
        {children}
      </div>
    </section>
  )
}

function ApiDirectiveSection({
  children,
  description,
  method,
  path,
  title,
}: ApiDirectiveProps) {
  return (
    <section className='not-prose bg-card my-8 overflow-hidden rounded-lg border shadow-sm'>
      {(title || description || method || path) && (
        <header className='bg-muted/30 border-b px-5 py-4'>
          {title ? (
            <h3 className='text-foreground text-base font-semibold'>{title}</h3>
          ) : null}
          {description ? (
            <p className='text-muted-foreground mt-1 text-sm leading-6'>
              {description}
            </p>
          ) : null}
          {method && path ? (
            <div className='mt-3 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center'>
              <span
                className={cn(
                  'inline-flex h-7 w-fit items-center rounded-md border px-2.5 font-mono text-xs font-semibold',
                  getMethodClassName(method)
                )}
              >
                {method}
              </span>
              <code className='bg-muted text-foreground min-w-0 overflow-x-auto rounded-md px-3 py-2 font-mono text-sm leading-none'>
                {path}
              </code>
            </div>
          ) : null}
        </header>
      )}
      <div className='api-doc-body px-5 py-5 text-sm leading-7 [&>*:first-child]:mt-0 [&>*:last-child]:mb-0'>
        {children}
      </div>
    </section>
  )
}

function ApiTab({ children }: ApiDirectiveProps) {
  return <>{children}</>
}

function ApiTabs({ children, title }: ApiDirectiveProps) {
  const { t } = useTranslation()
  const tabs = Children.toArray(children).flatMap((child, index) => {
    if (!isValidElement<ApiDirectiveProps>(child) || child.type !== ApiTab) {
      return []
    }

    return [
      {
        content: child.props.children,
        title: child.props.title || `${t('Example')} ${index + 1}`,
      },
    ]
  })
  const [activeTab, setActiveTab] = useState(0)
  const activeIndex = activeTab < tabs.length ? activeTab : 0

  if (tabs.length === 0) {
    return (
      <div className='not-prose my-7 [&>*:first-child]:mt-0 [&>*:last-child]:mb-0'>
        {children}
      </div>
    )
  }

  return (
    <section className='not-prose bg-card my-7 overflow-hidden rounded-lg border shadow-sm'>
      <header className='bg-muted/35 border-b px-3 py-3'>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <div className='text-muted-foreground flex items-center gap-2 text-sm font-medium'>
            <Code2 className='size-4' />
            <span>{title || t('Example')}</span>
          </div>
          <div className='bg-muted inline-flex w-fit max-w-full gap-1 overflow-x-auto rounded-md p-1'>
            {tabs.map((tab, index) => (
              <button
                key={`${tab.title}-${index}`}
                type='button'
                className={cn(
                  'text-muted-foreground hover:text-foreground h-7 rounded px-2.5 text-xs font-medium whitespace-nowrap transition-colors',
                  index === activeIndex &&
                    'bg-background text-foreground shadow-sm'
                )}
                onClick={() => setActiveTab(index)}
              >
                {tab.title}
              </button>
            ))}
          </div>
        </div>
      </header>
      <div className='api-doc-body [&>*:first-child]:mt-0 [&>*:last-child]:mb-0'>
        {tabs[activeIndex]?.content}
      </div>
    </section>
  )
}

function MarkdownTable({ children }: { children: ReactNode }) {
  const kind = getTableKind(children)

  return (
    <div
      className={cn(
        'not-prose bg-card my-7 overflow-x-auto rounded-lg border shadow-sm',
        kind !== 'default' && '[&_tbody_tr:hover]:bg-muted/45'
      )}
    >
      <table
        className={cn(
          'w-full border-collapse text-sm',
          kind === 'parameters' &&
            '[&_td:first-child]:font-mono [&_td:first-child]:text-[0.8125rem] [&_td:first-child]:font-semibold',
          kind === 'status' &&
            '[&_td:first-child]:font-mono [&_td:first-child]:font-semibold',
          kind === 'clients' && '[&_td:first-child]:font-semibold',
          kind === 'comparison' && '[&_td:first-child]:font-semibold'
        )}
      >
        {children}
      </table>
    </div>
  )
}

function MarkdownHeading({
  level,
  id,
  children,
}: {
  level: 1 | 2 | 3 | 4
  id: string
  children: ReactNode
}) {
  const Tag = `h${level}` as 'h1' | 'h2' | 'h3' | 'h4'

  return (
    <Tag id={id} className='group scroll-mt-24'>
      <a
        href={`#${id}`}
        className='text-foreground hover:text-foreground inline-flex items-center gap-2 no-underline'
      >
        <span>{children}</span>
        <Hash className='text-muted-foreground size-4 opacity-0 transition-opacity group-hover:opacity-100' />
      </a>
    </Tag>
  )
}

function MarkdownCallout({
  kind,
  children,
}: {
  kind: CalloutKind
  children: ReactNode
}) {
  const { t } = useTranslation()
  const labels: Record<CalloutKind, string> = {
    note: t('Note'),
    tip: t('Tip'),
    important: t('Important'),
    warning: t('Warning'),
    caution: t('Caution'),
  }
  const icons: Record<CalloutKind, ReactNode> = {
    note: <Info className='size-4' />,
    tip: <Lightbulb className='size-4' />,
    important: <OctagonAlert className='size-4' />,
    warning: <TriangleAlert className='size-4' />,
    caution: <OctagonAlert className='size-4' />,
  }
  const styles: Record<CalloutKind, string> = {
    note: 'border-sky-200 bg-sky-50/70 text-sky-950 dark:border-sky-900/70 dark:bg-sky-950/30 dark:text-sky-100',
    tip: 'border-emerald-200 bg-emerald-50/70 text-emerald-950 dark:border-emerald-900/70 dark:bg-emerald-950/30 dark:text-emerald-100',
    important:
      'border-violet-200 bg-violet-50/70 text-violet-950 dark:border-violet-900/70 dark:bg-violet-950/30 dark:text-violet-100',
    warning:
      'border-amber-200 bg-amber-50/70 text-amber-950 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-100',
    caution:
      'border-red-200 bg-red-50/70 text-red-950 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-100',
  }

  return (
    <aside className={cn('not-prose my-6 rounded-lg border p-4', styles[kind])}>
      <div className='mb-2 flex items-center gap-2 text-sm font-semibold'>
        {icons[kind]}
        <span>{labels[kind]}</span>
      </div>
      <div className='text-sm leading-6 [&>*:first-child]:mt-0 [&>*:last-child]:mb-0'>
        {children}
      </div>
    </aside>
  )
}

export function Markdown({ children, className }: MarkdownProps) {
  const headingCounts = new Map<string, number>()

  const nextHeadingId = (content: ReactNode) =>
    uniqueHeadingId(extractNodeText(content), headingCounts)

  const components: Components = {
    a: ({ href, ...props }) => {
      const isExternal = href ? /^(https?:)?\/\//.test(href) : false
      return (
        <a
          href={href}
          target={isExternal ? '_blank' : undefined}
          rel={isExternal ? 'noopener noreferrer' : undefined}
          {...props}
        />
      )
    },
    blockquote: ({ children: blockquoteChildren }) => {
      const kind = getCalloutKind(blockquoteChildren)
      if (kind) {
        return (
          <MarkdownCallout kind={kind}>
            {stripCalloutMarker(blockquoteChildren)}
          </MarkdownCallout>
        )
      }

      return (
        <blockquote className='border-l-primary bg-muted/50 my-6 rounded-r-lg border-l-4 px-4 py-3 text-sm leading-6'>
          {blockquoteChildren}
        </blockquote>
      )
    },
    code: ({ className: codeClassName, children: codeChildren, ...props }) => {
      const value = String(codeChildren ?? '').replace(/\n$/, '')
      const language = /language-([^\s]+)/.exec(codeClassName ?? '')?.[1]
      const isBlock = Boolean(language || value.includes('\n'))

      if (isBlock) {
        return <CodeBlock code={value} language={language} />
      }

      return (
        <code
          className={cn(
            'bg-muted text-foreground rounded px-1.5 py-0.5 text-[0.85em] font-medium before:content-none after:content-none',
            codeClassName
          )}
          {...props}
        >
          {codeChildren}
        </code>
      )
    },
    h1: ({ children: headingChildren }) => (
      <MarkdownHeading level={1} id={nextHeadingId(headingChildren)}>
        {headingChildren}
      </MarkdownHeading>
    ),
    h2: ({ children: headingChildren }) => (
      <MarkdownHeading level={2} id={nextHeadingId(headingChildren)}>
        {headingChildren}
      </MarkdownHeading>
    ),
    h3: ({ children: headingChildren }) => (
      <MarkdownHeading level={3} id={nextHeadingId(headingChildren)}>
        {headingChildren}
      </MarkdownHeading>
    ),
    h4: ({ children: headingChildren }) => (
      <MarkdownHeading level={4} id={nextHeadingId(headingChildren)}>
        {headingChildren}
      </MarkdownHeading>
    ),
    hr: () => <hr className='my-8' />,
    p: ({ children: paragraphChildren }) => {
      const endpoint = parseEndpoint(paragraphChildren)
      if (endpoint) {
        return <MarkdownEndpoint {...endpoint} />
      }

      return <p>{paragraphChildren}</p>
    },
    pre: ({ children: preChildren }) => <>{preChildren}</>,
    table: ({ children: tableChildren }) => (
      <MarkdownTable>{tableChildren}</MarkdownTable>
    ),
    tbody: ({ children: tableChildren }) => <tbody>{tableChildren}</tbody>,
    td: ({ children: tableChildren }) => (
      <td className='border-t px-4 py-3.5 align-top leading-6'>
        {tableChildren}
      </td>
    ),
    th: ({ children: tableChildren }) => (
      <th className='bg-muted/55 text-foreground px-4 py-3 text-left text-xs font-semibold tracking-wide uppercase'>
        {tableChildren}
      </th>
    ),
    thead: ({ children: tableChildren }) => (
      <thead className='border-b'>{tableChildren}</thead>
    ),
    tr: ({ children: tableChildren }) => (
      <tr className='even:bg-muted/20 transition-colors'>{tableChildren}</tr>
    ),
  }

  Object.assign(components, {
    'api-endpoint': (props: ApiDirectiveProps) => (
      <ApiDirectiveEndpoint {...props} />
    ),
    'api-parameters': (props: ApiDirectiveProps) => (
      <ApiDirectivePanel {...props} variant='parameters' />
    ),
    'api-request': (props: ApiDirectiveProps) => (
      <ApiDirectivePanel {...props} variant='request' />
    ),
    'api-response': (props: ApiDirectiveProps) => (
      <ApiDirectivePanel {...props} variant='response' />
    ),
    'api-section': (props: ApiDirectiveProps) => (
      <ApiDirectiveSection {...props} />
    ),
    'api-tab': ApiTab,
    'api-tabs': ApiTabs,
  })

  return (
    <div className={cn('markdown-content', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkDirective, remarkApiDirectives]}
        rehypePlugins={[rehypeRaw]}
        components={components}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
