import { describe, expect, it } from 'vitest'
import {
  buildDeconstructPrompt,
  buildDirectResearchPrompt,
  buildGenreCardPrompt,
  buildMarketScanPrompt,
  RESEARCH_SKILL_COMMAND,
  type ResearchPromptTranslate,
} from './research-prompts'

const t: ResearchPromptTranslate = (key, options) => {
  const values = options
    ? Object.entries(options).map(([name, value]) => `${name}=${String(value)}`).join(',')
    : ''
  return `${key}${values ? `(${values})` : ''}`
}

describe('research prompts', () => {
  it('prefixes direct questions with the research Skill command', () => {
    const prompt = buildDirectResearchPrompt('最近起点玄幻什么书火', t)

    expect(prompt.startsWith(`${RESEARCH_SKILL_COMMAND}\n\n`)).toBe(true)
    expect(prompt).toContain('question=最近起点玄幻什么书火')
  })

  it('asks for the question before guessing when the question is empty', () => {
    const prompt = buildDirectResearchPrompt('   ', t)

    expect(prompt).toContain('research.prompt.questionEmpty')
  })

  it('builds market-scan prompts with platform and material', () => {
    const prompt = buildMarketScanPrompt({ platform: 'qidian', material: '《凡人修仙传》' }, t)

    expect(prompt.startsWith(`${RESEARCH_SKILL_COMMAND}\n\n`)).toBe(true)
    expect(prompt).toContain('platform=research.platform.qidian')
    expect(prompt).toContain('material=《凡人修仙传》')
  })

  it('asks the author for material instead of inventing a work', () => {
    const scan = buildMarketScanPrompt({ platform: 'any', material: ' ' }, t)
    const deconstruct = buildDeconstructPrompt({ platform: 'fanqie', material: '' }, t)

    expect(scan).toContain('material=research.prompt.materialEmpty')
    expect(deconstruct).toContain('material=research.prompt.materialEmpty')
  })

  it('builds deconstruction prompts with the selected platform', () => {
    const prompt = buildDeconstructPrompt({ platform: 'jjwxc', material: 'https://example.test/book' }, t)

    expect(prompt.startsWith(`${RESEARCH_SKILL_COMMAND}\n\n`)).toBe(true)
    expect(prompt).toContain('platform=research.platform.jjwxc')
    expect(prompt).toContain('research.prompt.deconstruct')
  })

  it('prefixes genre prompts with the Skill command and normalizes slashes', () => {
    expect(buildGenreCardPrompt('genre-xianxia', t).startsWith('/genre-xianxia\n\n')).toBe(true)
    expect(buildGenreCardPrompt('/genre-urban', t).startsWith('/genre-urban\n\n')).toBe(true)
  })
})
