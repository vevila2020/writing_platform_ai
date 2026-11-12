import { describe, expect, it } from 'vitest'
import type { SkillSummary } from '@/lib/api'
import {
  BUILTIN_WRITING_SKILLS,
  DEFAULT_WRITING_SKILL,
  resolveWritingSkillSelection,
  writingSkillOptionsFromSnapshot,
} from './useWritingSkillOptions'

function skill(overrides: Partial<SkillSummary> & { name: string }): SkillSummary {
  return {
    description: `${overrides.name} description`,
    category: 'writing',
    scope: 'builtin',
    path: `skills/${overrides.name}/SKILL.md`,
    editable: false,
    active: true,
    enabled: true,
    agent: 'ide',
    capabilities: ['writing-workflow'],
    ...overrides,
  }
}

describe('writingSkillOptionsFromSnapshot', () => {
  it('keeps only active writing workflows available to the IDE agent', () => {
    const options = writingSkillOptionsFromSnapshot([
      skill({ name: 'novel-serial' }),
      skill({ name: 'novel-disabled', enabled: false }),
      skill({ name: 'novel-inactive', active: false }),
      skill({ name: 'novel-image', agent: 'image' }),
      skill({ name: 'novel-craft', capabilities: ['genre-card'] }),
    ])

    expect(options.map((option) => option.name)).toEqual(['novel-serial'])
  })

  it('lists the default first, then builtin workflows, then other scopes', () => {
    const options = writingSkillOptionsFromSnapshot([
      skill({ name: 'novel-serial' }),
      skill({ name: 'novel-standard' }),
      skill({ name: 'workspace-flow', scope: 'workspace' }),
      skill({ name: 'novel-lite' }),
      skill({ name: 'user-flow', scope: 'user' }),
    ])

    expect(options.map((option) => option.name)).toEqual([
      DEFAULT_WRITING_SKILL,
      'novel-serial',
      'novel-standard',
      'user-flow',
      'workspace-flow',
    ])
  })

  it('includes novel-serial in the builtin fallback list', () => {
    expect(BUILTIN_WRITING_SKILLS).toContain('novel-serial')
  })
})

describe('resolveWritingSkillSelection', () => {
  const options = [skill({ name: 'novel-lite' }), skill({ name: 'novel-serial' })]

  it('keeps a configured available workflow', () => {
    expect(resolveWritingSkillSelection('novel-serial', options)).toBe('novel-serial')
  })

  it('falls back to the default when the configured workflow is unavailable', () => {
    expect(resolveWritingSkillSelection('missing', options)).toBe(DEFAULT_WRITING_SKILL)
  })
})
