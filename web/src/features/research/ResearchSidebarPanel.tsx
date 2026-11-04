import { useMemo, useState } from 'react'
import { FileText, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { flattenFileTree } from '@/components/workbench/workbench-utils'
import { GENRE_CARD_CAPABILITY, useGenreCardOptions } from '@/hooks/useGenreCardOptions'
import type { FileNode } from '@/hooks/useWorkspace'
import {
  buildDeconstructPrompt,
  buildDirectResearchPrompt,
  buildGenreCardPrompt,
  buildMarketScanPrompt,
  RESEARCH_PLATFORMS,
  type ResearchPlatform,
} from './research-prompts'

export interface ResearchStartOptions {
  /** Submits immediately instead of prefilling the composer. */
  autoSend?: boolean
  /** Opens a fresh conversation so research never overwrites the current chat. */
  newConversation?: boolean
}

interface ResearchSidebarPanelProps {
  projectId: string
  tree: FileNode[]
  onStartResearch: (prompt: string, options?: ResearchStartOptions) => void
  onSelectFile: (path: string) => boolean | void | Promise<boolean | void>
  onRefreshWorkspace: () => void | Promise<void>
}

const RESEARCH_DIRECTORY = 'setting/research/'

/**
 * Writing sidebar 研究 tab: direct questions, 扫榜/拆文 prompts, genre cards,
 * and the research reports saved under `setting/research/`.
 *
 * Only mounted while the research view is active so other sidebar views never
 * trigger the Skills catalog request.
 */
export function ResearchSidebarPanel({
  projectId,
  tree,
  onStartResearch,
  onSelectFile,
  onRefreshWorkspace,
}: ResearchSidebarPanelProps) {
  const { t } = useTranslation()
  const [question, setQuestion] = useState('')
  const [platform, setPlatform] = useState<ResearchPlatform>('any')
  const [material, setMaterial] = useState('')
  const genreCards = useGenreCardOptions(projectId, Boolean(projectId))

  const notes = useMemo(
    () => flattenFileTree(tree)
      .filter((path) => path.startsWith(RESEARCH_DIRECTORY) && path.endsWith('.md'))
      .sort((a, b) => b.localeCompare(a)),
    [tree],
  )

  if (!projectId) {
    return (
      <div className="p-3 text-xs text-[var(--nova-text-muted)]">
        {t('research.noProject')}
      </div>
    )
  }

  const submitQuestion = () => {
    const trimmed = question.trim()
    if (!trimmed) return
    onStartResearch(buildDirectResearchPrompt(trimmed, t), { autoSend: true, newConversation: true })
    setQuestion('')
  }
  const startScan = () => {
    onStartResearch(buildMarketScanPrompt({ platform, material }, t), { newConversation: true })
  }
  const startDeconstruct = () => {
    onStartResearch(buildDeconstructPrompt({ platform, material }, t), { newConversation: true })
  }

  return (
    <div className="flex h-full min-h-0 flex-col overflow-y-auto p-3 text-xs">
      <h2 className="mb-1 text-sm font-medium text-[var(--nova-text)]">{t('research.title')}</h2>
      <p className="mb-3 text-[11px] leading-relaxed text-[var(--nova-text-muted)]">{t('research.description')}</p>

      <section className="mb-4">
        <h3 className="mb-1.5 text-[11px] font-medium text-[var(--nova-text-muted)]">{t('research.direct.title')}</h3>
        <Textarea
          value={question}
          onChange={(event) => setQuestion(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault()
              submitQuestion()
            }
          }}
          rows={2}
          placeholder={t('research.direct.placeholder')}
          aria-label={t('research.direct.title')}
          className="min-h-14 resize-none text-xs"
        />
        <Button
          size="sm"
          className="mt-1.5 w-full"
          disabled={!question.trim()}
          onClick={submitQuestion}
        >
          {t('research.direct.submit')}
        </Button>
      </section>

      <section className="mb-4">
        <h3 className="mb-1.5 text-[11px] font-medium text-[var(--nova-text-muted)]">{t('research.scan.title')}</h3>
        <label className="mb-1 block text-[11px] text-[var(--nova-text-muted)]" htmlFor="research-platform">
          {t('research.scan.platform')}
        </label>
        <select
          id="research-platform"
          value={platform}
          onChange={(event) => setPlatform(event.target.value as ResearchPlatform)}
          className="nova-field mb-1.5 h-7 w-full rounded border px-2 text-xs outline-none"
        >
          {RESEARCH_PLATFORMS.map((value) => (
            <option key={value} value={value}>{t(`research.platform.${value}`)}</option>
          ))}
        </select>
        <Textarea
          value={material}
          onChange={(event) => setMaterial(event.target.value)}
          rows={3}
          placeholder={t('research.scan.materialPlaceholder')}
          aria-label={t('research.scan.materialPlaceholder')}
          className="min-h-16 resize-none text-xs"
        />
        <div className="mt-1.5 grid grid-cols-2 gap-1.5">
          <Button size="sm" variant="secondary" onClick={startScan}>{t('research.scan.action')}</Button>
          <Button size="sm" variant="outline" onClick={startDeconstruct}>{t('research.deconstruct.action')}</Button>
        </div>
      </section>

      <section className="mb-4">
        <h3 className="mb-1.5 text-[11px] font-medium text-[var(--nova-text-muted)]">{t('research.genre.title')}</h3>
        {genreCards.length === 0 ? (
          <p className="text-[11px] leading-relaxed text-[var(--nova-text-faint)]">
            {t('research.genre.empty', { capability: GENRE_CARD_CAPABILITY })}
          </p>
        ) : (
          <div className="grid grid-cols-2 gap-1.5">
            {genreCards.map((card) => (
              <button
                key={card.name}
                type="button"
                title={card.description}
                onClick={() => onStartResearch(buildGenreCardPrompt(card.name, t), { newConversation: true })}
                className="nova-nav-item border border-[var(--nova-border)] bg-[var(--nova-surface-2)] px-2 py-1 text-left text-[11px] leading-snug"
              >
                <span className="line-clamp-2">{card.description}</span>
              </button>
            ))}
          </div>
        )}
      </section>

      <section className="min-h-0">
        <div className="mb-1.5 flex items-center justify-between">
          <h3 className="text-[11px] font-medium text-[var(--nova-text-muted)]">{t('research.notes.title')}</h3>
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label={t('research.notes.refresh')}
            title={t('research.notes.refresh')}
            onClick={() => { void onRefreshWorkspace() }}
          >
            <RefreshCw />
          </Button>
        </div>
        {notes.length === 0 ? (
          <p className="text-[11px] leading-relaxed text-[var(--nova-text-faint)]">{t('research.notes.empty')}</p>
        ) : (
          <ul className="space-y-0.5">
            {notes.map((path) => (
              <li key={path}>
                <button
                  type="button"
                  onClick={() => { void onSelectFile(path) }}
                  className="flex w-full items-center gap-1.5 rounded px-1.5 py-1 text-left text-[11px] text-[var(--nova-text-muted)] hover:bg-[var(--nova-hover)] hover:text-[var(--nova-text)]"
                >
                  <FileText className="size-3.5 shrink-0" />
                  <span className="truncate">{path.slice(RESEARCH_DIRECTORY.length)}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
