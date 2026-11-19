/** Keeps trace summaries faithful to the first meaningful line of model output. */
export function agentContentPreview(content: string) {
  const firstLine = content.trimStart().split(/\r?\n/, 1)[0]
  return firstLine.replace(/^#{1,6}\s+/, '').replace(/(\*\*|__|`)(.+?)\1/g, '$2').trim()
}
