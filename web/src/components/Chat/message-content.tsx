import { useVirtualizedMessageState } from './VirtualizedMessageState'
import { Children, Fragment, cloneElement, isValidElement, memo, useMemo } from 'react'
import type { ReactNode } from 'react'
import { cjk } from '@streamdown/cjk'
import { math } from '@streamdown/math'
import { ChevronDown, Feather } from 'lucide-react'
import {
  Streamdown,
  defaultRehypePlugins,
  type Components as StreamdownComponents,
  type ControlsConfig,
  type StreamdownTranslations,
} from 'streamdown'
import { useTranslation } from 'react-i18next'
import { ImagePreviewDialog } from '@/components/common/ImagePreviewDialog'
import { projectFileAssetURL, type ThinkingChatMessage } from '@/lib/api'
import { findDialogueHighlightRanges } from '@/lib/dialogue-highlight'
import { useBottomScrollLock } from '@/hooks/useBottomScrollLock'
import { isWorkspaceImagePath } from '@/lib/workspace-file-kind'
import { Reasoning, ReasoningContent, ReasoningTrigger } from '@/components/ai-elements/reasoning'
import { cn } from '@/lib/utils'
import { agentContentPreview } from './agent-content-preview'
import { AgentSourceBadge } from './message-source-badge'

const chatMarkdownPlugins = { cjk, math }
const chatMarkdownControls = {
  code: { copy: true, download: false },
  table: { copy: true, download: false, fullscreen: true },
  image: false,
  mermaid: false,
} satisfies ControlsConfig
// Keep Streamdown's HTML sanitization without its final URL hardener: that
// hardener replaces InkDock's bare workspace image paths before our image
// component can resolve them through the authenticated Project asset API.
const chatMarkdownRehypePlugins = [
  defaultRehypePlugins.raw,
  defaultRehypePlugins.sanitize,
]

interface MarkdownContentProps {
  content: string
  highlightDialogue: boolean
  projectId: string
  streaming?: boolean
  animateWords?: boolean
}

interface ChatMarkdownImageProps {
  src?: string
  alt?: string
  title?: string
  projectId?: string
}

export function StreamingPlaceholder() {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-2 py-1 text-xs font-medium text-[var(--nova-text-muted)]" role="status">
      <ThinkingIcon streaming />{t('chat.trace.thinkingActive')}
    </div>
  )
}

export function sanitizeThinkTags(text: string): string {
  let result = text
  // Remove provider-internal tokens and textual tool-call residue from history;
  // current conversations execute parsed tool calls on the backend.
  result = result.replace(/\]<\]minimax\[>\[/g, '')
  result = result.replace(/<tool_call>[\s\S]*?<\/tool_call>/gi, '')
  result = result.replace(/<invoke\s+name="[^"]*"[\s\S]*?<\/invoke>/gi, '')
  // Remove paired or unclosed <think> blocks.
  result = result.replace(/<think>[\s\S]*?(?:<\/think>|$)/gi, '')
  // Some providers omit the opening tag; drop the prefix through the first close.
  const close = result.search(/<\s*\/\s*think\s*>/i)
  if (close >= 0) {
    result = result.slice(close).replace(/<\s*\/\s*think\s*>/i, '')
  }
  // Remove any remaining standalone tags.
  return result.replace(/<\/?\s*think\s*>/gi, '')
}

export const MarkdownContent = memo(function MarkdownContent({
  content,
  highlightDialogue,
  projectId,
  streaming = false,
  animateWords = true,
}: MarkdownContentProps) {
  const { t } = useTranslation()
  const components = useMemo(() => ({
    ...(highlightDialogue ? dialogueMarkdownComponents : markdownComponents),
    img: (props: ChatMarkdownImageProps) => <ChatMarkdownImage {...props} projectId={projectId} />,
  }) as StreamdownComponents, [highlightDialogue, projectId])
  const translations = useMemo<Partial<StreamdownTranslations>>(() => ({
    copied: t('chat.action.copyMessageDone'),
    copyCode: t('chat.markdown.copyCode'),
    copyTable: t('chat.markdown.copyTable'),
    copyTableAsCsv: t('chat.markdown.copyTableAsCsv'),
    copyTableAsMarkdown: t('chat.markdown.copyTableAsMarkdown'),
    copyTableAsTsv: t('chat.markdown.copyTableAsTsv'),
    exitFullscreen: t('chat.markdown.exitFullscreen'),
    viewFullscreen: t('chat.markdown.viewFullscreen'),
  }), [t])
  // Let the surrounding message or panel own vertical scrolling while tables
  // retain Streamdown's horizontal overflow behavior.
  return (
    <Streamdown
      animated={animateWords}
      className="nova-streamdown min-w-0 space-y-2"
      components={components}
      controls={chatMarkdownControls}
      isAnimating={streaming}
      lineNumbers={false}
      mode="streaming"
      plugins={chatMarkdownPlugins}
      rehypePlugins={chatMarkdownRehypePlugins}
      tableMaxHeight={0}
      translations={translations}
    >
      {content}
    </Streamdown>
  )
})

const markdownComponents = {
  a: ChatMarkdownLink,
  img: ChatMarkdownImage,
}

const dialogueMarkdownComponents = {
  ...markdownComponents,
  p: ({ children }: { children?: ReactNode }) => <p>{highlightDialogueNodes(children)}</p>,
  li: ({ children }: { children?: ReactNode }) => <li>{highlightDialogueNodes(children)}</li>,
  h1: ({ children }: { children?: ReactNode }) => <h1>{highlightDialogueNodes(children)}</h1>,
  h2: ({ children }: { children?: ReactNode }) => <h2>{highlightDialogueNodes(children)}</h2>,
  h3: ({ children }: { children?: ReactNode }) => <h3>{highlightDialogueNodes(children)}</h3>,
  h4: ({ children }: { children?: ReactNode }) => <h4>{highlightDialogueNodes(children)}</h4>,
  h5: ({ children }: { children?: ReactNode }) => <h5>{highlightDialogueNodes(children)}</h5>,
  h6: ({ children }: { children?: ReactNode }) => <h6>{highlightDialogueNodes(children)}</h6>,
  blockquote: ({ children }: { children?: ReactNode }) => <blockquote>{highlightDialogueNodes(children)}</blockquote>,
}

function ChatMarkdownLink({ href, children }: { href?: string; title?: string; children?: ReactNode }) {
  const external = /^https?:\/\//i.test(href?.trim() || '')
  return (
    <a
      href={href}
      {...(external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
    >
      {children}
    </a>
  )
}

function ChatMarkdownImage({ src = '', alt = '', title = '', projectId = '' }: ChatMarkdownImageProps) {
  const { t } = useTranslation()
  const imageSrc = normalizeChatImageSrc(src, projectId)
  if (!imageSrc) return null
  const imageTitle = alt || title || t('chat.image.previewTitle')
  const imagePath = shouldShowImagePath(src) ? src : undefined

  return (
    <ImagePreviewDialog src={imageSrc} title={imageTitle} alt={alt || imageTitle} path={imagePath}>
      <button type="button" className="nova-chat-image-button" aria-label={t('chat.image.openPreview')}>
        <img src={imageSrc} alt={alt || imageTitle} loading="lazy" />
      </button>
    </ImagePreviewDialog>
  )
}

function normalizeChatImageSrc(src: string, projectId: string) {
  const trimmed = src.trim()
  if (!trimmed) return ''
  if (/^(https?:|data:|blob:|\/)/i.test(trimmed)) return trimmed
  let workspacePath = trimmed
  try {
    // Markdown normalizes non-ASCII destinations to encoded URIs. Decode that
    // representation once before URLSearchParams encodes the API query value.
    workspacePath = decodeURIComponent(trimmed)
  } catch {
    // Invalid percent sequences can be literal filename characters.
  }
  if (isWorkspaceImagePath(workspacePath)) return chatAssetURL(projectId, workspacePath)
  return trimmed
}

export function chatAssetURL(projectId: string, path: string) {
  return projectId ? projectFileAssetURL(projectId, path) : ''
}

function shouldShowImagePath(src: string) {
  const trimmed = src.trim()
  return Boolean(trimmed && !/^(data:|blob:)/i.test(trimmed))
}

function highlightDialogueNodes(children: ReactNode): ReactNode {
  return Children.map(children, (child, index) => {
    if (typeof child === 'string') return highlightDialogueText(child, true, `md-${index}`)
    if (!isValidElement(child)) return child
    const props = child.props as { children?: ReactNode }
    if (props.children === undefined) return child
    return cloneElement(child, undefined, highlightDialogueNodes(props.children))
  })
}

function highlightDialogueText(text: string, enabled: boolean, keyPrefix: string): ReactNode {
  if (!enabled || !text) return text
  const nodes: ReactNode[] = []
  const ranges = findDialogueHighlightRanges(text)
  let lastIndex = 0

  ranges.forEach((range, index) => {
    if (range.from > lastIndex) nodes.push(text.slice(lastIndex, range.from))
    nodes.push(
      <span key={`${keyPrefix}-dialogue-${index}`} className="nova-dialogue-highlight">
        {text.slice(range.from, range.to)}
      </span>,
    )
    lastIndex = range.to
  })

  if (lastIndex < text.length) nodes.push(text.slice(lastIndex))
  if (nodes.length === 0) return text
  return <Fragment>{nodes}</Fragment>
}

const thinkingPlugins = { cjk }
// Keep transcript references as text. Only a reply should load images or offer actions.
const thinkingComponents: StreamdownComponents = {
  strong: ({ children }) => <strong>{children}</strong>,
  img: ({ alt }) => <>{alt}</>,
  a: ({ children }) => <>{children}</>,
  pre: ({ children }) => <pre>{children}</pre>,
  code: ({ children }) => <code>{children}</code>,
}
const thinkingRehypePlugins: NonNullable<React.ComponentProps<typeof Streamdown>['rehypePlugins']> = []

function ThinkingIcon({ streaming }: { streaming: boolean }) {
  return (
    <span aria-hidden="true" className={cn(
      'grid size-6 shrink-0 place-items-center rounded-md',
      streaming ? 'bg-[var(--nova-warning-bg)] text-[var(--nova-warning)]' : 'bg-[var(--nova-surface-2)] text-[var(--nova-text-muted)]',
    )}>
      <Feather className={cn('size-3.5', streaming && 'nova-thinking-pulse')} />
    </span>
  )
}

/** Follow the stream until the reader explicitly folds or opens this entry. */
export function ThinkingBlock({ message, content, streaming, showAgentSource = true }: { message: ThinkingChatMessage; content: string; streaming: boolean; showAgentSource?: boolean }) {
  const { t } = useTranslation()
  const text = (streaming ? message.streaming_target_content : undefined) ?? content
  const hasText = /\S/.test(text)
  const [expansion, setExpanded] = useVirtualizedMessageState<'auto' | boolean>('thinking-expanded', 'auto')
  const expanded = expansion === 'auto' ? streaming : expansion
  const scroll = useBottomScrollLock<HTMLDivElement>({
    enabled: streaming && hasText && expanded,
    resetKey: message.id || message.created_at,
    contentKey: text.length,
  })
  const source = showAgentSource && message.subagent && <AgentSourceBadge message={message} compact />

  if (!hasText) return streaming ? (
    <div className="flex items-center gap-2"><StreamingPlaceholder />{source}</div>
  ) : null

  return (
    <Reasoning isStreaming={streaming} open={expanded} onOpenChange={setExpanded}
      className="mb-0 min-w-0 rounded-lg border border-transparent data-[state=open]:border-[var(--nova-border-soft)] data-[state=open]:bg-[var(--nova-surface)]">
      <ReasoningTrigger
        aria-label={t(expanded ? 'chat.trace.collapseThinking' : 'chat.trace.expandThinking')}
        className="gap-2 rounded-md px-2 py-1.5 text-xs hover:bg-[var(--nova-hover)] focus-visible:outline-2 focus-visible:outline-[var(--nova-accent)]"
      >
        <ThinkingIcon streaming={streaming} />
        <span className="shrink-0 font-medium text-[var(--nova-text)]">{t(streaming ? 'chat.trace.thinkingActive' : 'chat.trace.thinking')}</span>
        <span className="min-w-0 flex-1 truncate text-left text-[var(--nova-text-faint)]">{!expanded && agentContentPreview(text)}</span>
        {source}
        <ChevronDown aria-hidden="true" className={cn('size-3.5 shrink-0 text-[var(--nova-text-faint)]', !expanded && '-rotate-90')} />
      </ReasoningTrigger>
      <ReasoningContent className="mt-0">
        <div
          ref={scroll.ref}
          role="region"
          tabIndex={0}
          aria-label={t('chat.trace.thinkingContent')}
          data-nova-scroll-lock="thinking"
          onScroll={scroll.onScroll}
          onWheel={scroll.onWheel}
          onKeyDown={scroll.onKeyDown}
          className="mx-2 mb-2 max-h-[min(18rem,40dvh)] overflow-y-auto overscroll-contain rounded-sm [scrollbar-width:thin] [overflow-anchor:none] focus-visible:outline-1 focus-visible:outline-[var(--nova-accent)]"
        >
          <Streamdown
            className="nova-thinking-prose"
            components={thinkingComponents}
            plugins={thinkingPlugins}
            rehypePlugins={thinkingRehypePlugins}
            skipHtml
            controls={false}
            tableMaxHeight={0}
          >
            {text}
          </Streamdown>
        </div>
      </ReasoningContent>
    </Reasoning>
  )
}
