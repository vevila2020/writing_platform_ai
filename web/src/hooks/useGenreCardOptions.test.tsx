import { describe, expect, it } from 'vitest'
import type { SkillSummary } from '@/lib/api'
import { genreCardOptionsFromSnapshot } from './useGenreCardOptions'

function skill(overrides: Partial<SkillSummary> & { name: string }): SkillSummary {
  return {
    description: `${overrides.name} description`,
    category: 'writing',
    scope: 'builtin',
    path: `skills/${overrides.name}/SKILL.md`,
    editable: false,
    active: true,
    enabled: true,
    ...overrides,
  }
}

describe('genreCardOptionsFromSnapshot', () => {
  it('keeps only enabled, active genre cards available to the IDE agent', () => {
    const options = genreCardOptionsFromSnapshot([
      skill({ name: 'genre-xianxia', capabilities: ['genre-card'] }),
      skill({ name: 'genre-disabled', capabilities: ['genre-card'], enabled: false }),
      skill({ name: 'genre-inactive', capabilities: ['genre-card'], active: false }),
      skill({ name: 'genre-image', capabilities: ['genre-card'], agent: 'image' }),
      skill({ name: 'novel-craft', capabilities: ['writing-workflow'] }),
      skill({ name: 'genre-no-capability' }),
    ])

    expect(options.map((option) => option.name)).toEqual(['genre-xianxia'])
  })

  it('respects an explicit per-agent settings override', () => {
    const options = genreCardOptionsFromSnapshot([
      skill({ name: 'genre-xianxia', capabilities: ['genre-card'] }),
      skill({ name: 'genre-urban', capabilities: ['genre-card'] }),
    ], { ide: { 'genre-xianxia': false } })

    expect(options.map((option) => option.name)).toEqual(['genre-urban'])
  })

  it('lists builtin cards before other scopes and keeps names deterministic', () => {
    const options = genreCardOptionsFromSnapshot([
      skill({ name: 'genre-urban', capabilities: ['genre-card'], scope: 'user' }),
      skill({ name: 'genre-xianxia', capabilities: ['genre-card'] }),
      skill({ name: 'genre-era', capabilities: ['genre-card'] }),
    ])

    expect(options.map((option) => option.name)).toEqual(['genre-era', 'genre-xianxia', 'genre-urban'])
  })
})
