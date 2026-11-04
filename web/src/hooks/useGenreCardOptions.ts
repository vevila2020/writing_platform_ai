import { useMemo } from 'react'
import { skillAvailableForAgent } from '@/features/agents/agent-registry'
import type { AgentSkillSettings } from '@/features/settings/types'
import type { SkillSummary } from '@/lib/api'
import { useAgentSkillCatalog } from './useAgentSkillCatalog'

/** Genre cards opt into keyword routing and the Writing research panel. */
export const GENRE_CARD_CAPABILITY = 'genre-card'

export type GenreCardOption = Pick<SkillSummary, 'name' | 'description' | 'scope' | 'path' | 'active' | 'agent' | 'capabilities'>

/**
 * Lists the genre cards available to the Writing Agent for the research panel.
 *
 * The catalog is shared with the Agent command surfaces, so mounted panels do
 * not create extra Skills or settings requests.
 */
export function useGenreCardOptions(projectId: string, enabled = true): GenreCardOption[] {
  const catalog = useAgentSkillCatalog(projectId, enabled).data
  return useMemo(
    () => genreCardOptionsFromSnapshot(catalog?.skills.skills || [], catalog?.settings.effective?.agent_skills),
    [catalog],
  )
}

export function genreCardOptionsFromSnapshot(skills: SkillSummary[], agentSkills?: AgentSkillSettings): GenreCardOption[] {
  return skills
    .filter((skill) => skill.active)
    .filter((skill) => skill.enabled !== false)
    .filter((skill) => skillAvailableForAgent(skill, 'ide', agentSkills))
    .filter((skill) => (skill.capabilities || []).includes(GENRE_CARD_CAPABILITY))
    .sort((a, b) => {
      if (a.scope !== b.scope && (a.scope === 'builtin' || b.scope === 'builtin')) return a.scope === 'builtin' ? -1 : 1
      return a.name.localeCompare(b.name)
    })
}
