export const ONBOARDING_OPEN_EVENT = 'nova:onboarding-open'
export const SETTINGS_SECTION_EVENT = 'nova:settings-open-section'
export const WRITING_AGENT_INIT_EVENT = 'nova:writing-agent-init'
// Opens a fresh Writing Agent conversation before delivering the same init payload.
// Background research launches use this so they never overwrite the current chat.
export const WRITING_AGENT_NEW_CONVERSATION_EVENT = 'nova:writing-agent-new-conversation'

export interface SettingsSectionRequest {
  section?: string
}

// Retain one navigation intent while the settings route is lazily mounting.
let pendingSettingsSection: string | undefined
export function requestSettingsSection(section: string) {
  pendingSettingsSection = section
  window.dispatchEvent(new CustomEvent(SETTINGS_SECTION_EVENT, { detail: { section } }))
}
export function takeSettingsSection(): string | undefined {
  const section = pendingSettingsSection
  pendingSettingsSection = undefined
  return section
}
