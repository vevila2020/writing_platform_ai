import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18next from '@/i18n'
import type { FileNode } from '@/hooks/useWorkspace'
import { ResearchSidebarPanel } from './ResearchSidebarPanel'

const genreCards = vi.hoisted(() => ({
  useGenreCardOptions: vi.fn(() => [
    { name: 'genre-xianxia', description: 'Xianxia card' },
    { name: 'genre-urban', description: 'Urban card' },
  ]),
}))

vi.mock('@/hooks/useGenreCardOptions', () => ({
  GENRE_CARD_CAPABILITY: 'genre-card',
  useGenreCardOptions: genreCards.useGenreCardOptions,
}))

const tree: FileNode[] = [{
  name: 'setting',
  type: 'dir',
  children: [{
    name: 'research',
    type: 'dir',
    children: [
      { name: '2026-11-01-qidian.md', type: 'file' },
      { name: '2026-10-01-fanqie.md', type: 'file' },
    ],
  }],
}, { name: 'notes.txt', type: 'file' }]

function setup(overrides: Partial<Parameters<typeof ResearchSidebarPanel>[0]> = {}) {
  const onStartResearch = vi.fn()
  const onSelectFile = vi.fn()
  const onRefreshWorkspace = vi.fn()
  render(
    <ResearchSidebarPanel
      projectId="project-a"
      tree={tree}
      onStartResearch={onStartResearch}
      onSelectFile={onSelectFile}
      onRefreshWorkspace={onRefreshWorkspace}
      {...overrides}
    />,
  )
  return { onStartResearch, onSelectFile, onRefreshWorkspace }
}

describe('ResearchSidebarPanel', () => {
  beforeEach(async () => {
    await i18next.changeLanguage('en-US')
    genreCards.useGenreCardOptions.mockClear()
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('submits a direct question immediately with the research Skill command', async () => {
    const user = userEvent.setup()
    const { onStartResearch } = setup()

    await user.type(screen.getByLabelText('Ask research'), 'what xianxia titles are hot on Qidian?')
    await user.click(screen.getByRole('button', { name: 'Ask' }))

    expect(onStartResearch).toHaveBeenCalledTimes(1)
    const [prompt, options] = onStartResearch.mock.calls[0]
    expect(prompt.startsWith('/novel-research\n\n')).toBe(true)
    expect(prompt).toContain('what xianxia titles are hot on Qidian?')
    expect(options).toEqual({ autoSend: true, newConversation: true })
  })

  it('prefills scan and deconstruct prompts without auto-sending', async () => {
    const user = userEvent.setup()
    const { onStartResearch } = setup()

    await user.selectOptions(screen.getByLabelText('Platform'), 'qidian')
    await user.type(screen.getByLabelText('Title, link, or pasted excerpt (optional)'), '《凡人修仙传》')
    await user.click(screen.getByRole('button', { name: 'Scan' }))
    await user.click(screen.getByRole('button', { name: 'Deconstruct' }))

    expect(onStartResearch).toHaveBeenCalledTimes(2)
    const [scanPrompt, scanOptions] = onStartResearch.mock.calls[0]
    const [deconstructPrompt, deconstructOptions] = onStartResearch.mock.calls[1]
    expect(scanPrompt.startsWith('/novel-research\n\n')).toBe(true)
    expect(scanPrompt).toContain('Qidian')
    expect(scanPrompt).toContain('《凡人修仙传》')
    expect(deconstructPrompt.startsWith('/novel-research\n\n')).toBe(true)
    expect(deconstructPrompt).toContain('Qidian')
    expect(scanOptions).toEqual({ newConversation: true })
    expect(deconstructOptions).toEqual({ newConversation: true })
  })

  it('prefills a genre card prompt when a card is clicked', async () => {
    const user = userEvent.setup()
    const { onStartResearch } = setup()

    const card = screen.getByRole('button', { name: 'Xianxia card' })
    // Hovering shows the full description, not the Skill slug.
    expect(card).toHaveAttribute('title', 'Xianxia card')
    await user.click(card)

    expect(onStartResearch).toHaveBeenCalledTimes(1)
    const [prompt, options] = onStartResearch.mock.calls[0]
    expect(prompt.startsWith('/genre-xianxia\n\n')).toBe(true)
    expect(options).toEqual({ newConversation: true })
  })

  it('lists research reports newest first and opens the selected report', async () => {
    const user = userEvent.setup()
    const { onSelectFile } = setup()

    const newest = screen.getByText('2026-11-01-qidian.md')
    expect(screen.getByText('2026-10-01-fanqie.md')).toBeInTheDocument()
    expect(screen.queryByText('notes.txt')).not.toBeInTheDocument()

    await user.click(newest)
    expect(onSelectFile).toHaveBeenCalledWith('setting/research/2026-11-01-qidian.md')
  })

  it('refreshes the workspace from the notes section', async () => {
    const user = userEvent.setup()
    const { onRefreshWorkspace } = setup()

    await user.click(screen.getByRole('button', { name: 'Refresh directory' }))
    expect(onRefreshWorkspace).toHaveBeenCalledTimes(1)
  })

  it('shows an empty state without a project', () => {
    setup({ projectId: '' })

    expect(screen.getByText('Open a book to run market scans and book deconstruction here.')).toBeInTheDocument()
  })
})
