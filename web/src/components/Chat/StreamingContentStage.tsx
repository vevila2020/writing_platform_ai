import { useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'

export type StreamingContentReveal = 'instant' | 'typewriter'

interface StreamingContentStageProps {
  content: string
  targetContent?: string
  streaming: boolean
  reveal?: StreamingContentReveal
  resetKey?: string
  children: (content: string) => ReactNode
}

/** Renders the latest complete streaming snapshot as a single content tree. */
export function StreamingContentStage({
  content,
  targetContent,
  streaming,
  reveal = 'instant',
  resetKey,
  children,
}: StreamingContentStageProps) {
  const target = streaming && targetContent !== undefined ? targetContent : content
  const visibleContent = useTypewriterText(target, streaming && reveal === 'typewriter', resetKey)
  return children(visibleContent)
}

// The backlog catches up over roughly this many frames. Normal provider deltas
// reveal one grapheme per frame, while a provider that emits a whole thought at
// once catches up quickly instead of delaying the stream for tens of seconds.
const TYPEWRITER_CATCH_UP_FRAMES = 24

function useTypewriterText(target: string, active: boolean, resetKey?: string) {
  const characters = useMemo(() => Array.from(target), [target])
  const [visibleCount, setVisibleCount] = useState(() => active ? 0 : characters.length)
  const visibleCountRef = useRef(visibleCount)
  const previousTargetRef = useRef(target)
  const previousResetKeyRef = useRef(resetKey)

  useEffect(() => {
    const resetChanged = previousResetKeyRef.current !== resetKey
    const previousVisible = previousTargetRef.current.slice(0, visibleCountRef.current)
    const replacementChanged = !target.startsWith(previousVisible)
    previousResetKeyRef.current = resetKey
    previousTargetRef.current = target

    if (!active) {
      visibleCountRef.current = characters.length
      setVisibleCount(characters.length)
      return undefined
    }
    if (resetChanged || replacementChanged || visibleCountRef.current > characters.length) {
      visibleCountRef.current = 0
      setVisibleCount(0)
    }

    let frameID: number | null = null
    const revealNext = () => {
      frameID = null
      const targetLength = characters.length
      const current = visibleCountRef.current
      if (current >= targetLength) return
      const backlog = targetLength - current
      const next = Math.min(targetLength, current + Math.max(1, Math.ceil(backlog / TYPEWRITER_CATCH_UP_FRAMES)))
      visibleCountRef.current = next
      setVisibleCount(next)
      if (next < targetLength) frameID = requestRevealFrame(revealNext)
    }
    if (visibleCountRef.current < characters.length) frameID = requestRevealFrame(revealNext)
    return () => {
      if (frameID !== null) cancelRevealFrame(frameID)
    }
  }, [active, characters, resetKey, target])

  const resolvedCount = active ? Math.min(visibleCount, characters.length) : characters.length
  return characters.slice(0, resolvedCount).join('')
}

function requestRevealFrame(callback: FrameRequestCallback) {
  if (typeof requestAnimationFrame === 'function') return requestAnimationFrame(callback)
  return window.setTimeout(() => callback(performance.now()), 16)
}

function cancelRevealFrame(frameID: number) {
  if (typeof cancelAnimationFrame === 'function') cancelAnimationFrame(frameID)
  else window.clearTimeout(frameID)
}
