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
  cloneElement,
  isValidElement,
  useEffect,
  useState,
  type ReactElement,
  type ReactNode,
} from 'react'
import {
  Check,
  Clipboard,
  Hash,
  Info,
  Lightbulb,
  OctagonAlert,
  TriangleAlert,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown, { type Components } from 'react-markdown'
import rehypeRaw from 'rehype-raw'
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
    <figure className='not-prose group bg-card my-6 overflow-hidden rounded-lg border shadow-sm'>
      <figcaption className='bg-muted/50 text-muted-foreground flex h-10 items-center justify-between border-b px-3 text-xs'>
        <span className='font-medium'>{displayLanguage(lang)}</span>
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
    pre: ({ children: preChildren }) => <>{preChildren}</>,
    table: ({ children: tableChildren }) => (
      <div className='not-prose my-6 overflow-x-auto rounded-lg border'>
        <table className='w-full border-collapse text-sm'>
          {tableChildren}
        </table>
      </div>
    ),
    tbody: ({ children: tableChildren }) => <tbody>{tableChildren}</tbody>,
    td: ({ children: tableChildren }) => (
      <td className='border-t px-4 py-3 align-top leading-6'>
        {tableChildren}
      </td>
    ),
    th: ({ children: tableChildren }) => (
      <th className='bg-muted/60 text-foreground px-4 py-3 text-left text-xs font-semibold tracking-wide uppercase'>
        {tableChildren}
      </th>
    ),
    thead: ({ children: tableChildren }) => (
      <thead className='border-b'>{tableChildren}</thead>
    ),
    tr: ({ children: tableChildren }) => (
      <tr className='even:bg-muted/25'>{tableChildren}</tr>
    ),
  }

  return (
    <div
      className={cn(
        'prose prose-neutral prose-sm dark:prose-invert max-w-none',
        'prose-headings:font-semibold prose-headings:tracking-tight prose-headings:scroll-mt-24',
        'prose-h1:text-3xl prose-h1:leading-tight prose-h2:mt-10 prose-h2:border-b prose-h2:pb-2 prose-h2:text-2xl prose-h3:mt-8 prose-h3:text-xl prose-h4:mt-6 prose-h4:text-base',
        'prose-p:my-4 prose-p:leading-7',
        'prose-a:text-primary prose-a:no-underline hover:prose-a:underline',
        'prose-strong:font-semibold',
        'prose-ul:my-4 prose-ol:my-4 prose-li:my-1.5',
        'prose-img:rounded-lg prose-img:shadow-sm',
        '[&>*:first-child]:mt-0 [&>*:last-child]:mb-0',
        '[overflow-wrap:anywhere] break-words',
        className
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw]}
        components={components}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
