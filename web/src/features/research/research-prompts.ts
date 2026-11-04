/**
 * Prompt builders for the Writing research panel.
 *
 * Every research prompt starts with the `/novel-research` command so the
 * Agent resolves the Skill before the first model call. Genre prompts start
 * with `/genre-*` for the same reason. Bodies live in the locale files, which
 * keeps the 扫榜/拆文 wording editable without touching this module.
 */

export const RESEARCH_SKILL_COMMAND = '/novel-research'

export const RESEARCH_PLATFORMS = ['any', 'qidian', 'fanqie', 'jjwxc', 'qimao', 'zhihu'] as const
export type ResearchPlatform = (typeof RESEARCH_PLATFORMS)[number]

export type ResearchPromptTranslate = (key: string, options?: Record<string, unknown>) => string

export interface ResearchMaterialInput {
  platform: ResearchPlatform
  /** A work title, link, or pasted excerpt. Empty means the user was asked to supply it. */
  material: string
}

function platformLabel(platform: ResearchPlatform, t: ResearchPromptTranslate): string {
  return t(`research.platform.${platform}`)
}

function materialText(material: string, t: ResearchPromptTranslate): string {
  const trimmed = material.trim()
  return trimmed || t('research.prompt.materialEmpty')
}

/** A free-form question such as 「最近起点玄幻什么书火」. */
export function buildDirectResearchPrompt(question: string, t: ResearchPromptTranslate): string {
  return [
    RESEARCH_SKILL_COMMAND,
    '',
    t('research.prompt.direct', { question: question.trim() || t('research.prompt.questionEmpty') }),
  ].join('\n')
}

/** 扫榜: prefills the composer so the author can add constraints before sending. */
export function buildMarketScanPrompt(input: ResearchMaterialInput, t: ResearchPromptTranslate): string {
  return [
    RESEARCH_SKILL_COMMAND,
    '',
    t('research.prompt.marketScan', {
      platform: platformLabel(input.platform, t),
      material: materialText(input.material, t),
    }),
  ].join('\n')
}

/** 拆文: prefills the composer with a three-pass deconstruction request. */
export function buildDeconstructPrompt(input: ResearchMaterialInput, t: ResearchPromptTranslate): string {
  return [
    RESEARCH_SKILL_COMMAND,
    '',
    t('research.prompt.deconstruct', {
      platform: platformLabel(input.platform, t),
      material: materialText(input.material, t),
    }),
  ].join('\n')
}

/** Loads one genre card explicitly; the backend also auto-loads it on keyword match. */
export function buildGenreCardPrompt(skillName: string, t: ResearchPromptTranslate): string {
  const name = skillName.trim().replace(/^\/+/, '')
  return [`/${name}`, '', t('research.prompt.genreCard')].join('\n')
}
