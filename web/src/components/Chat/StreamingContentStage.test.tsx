import { act, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StreamingContentStage } from './StreamingContentStage'

describe('StreamingContentStage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('reveals typewriter content frame by frame and shows the full snapshot when streaming settles', () => {
    const frames: FrameRequestCallback[] = []
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      frames.push(callback)
      return frames.length
    }))
    vi.stubGlobal('cancelAnimationFrame', vi.fn())

    const { rerender } = render(
      <StreamingContentStage content="" targetContent="abcdefghijklmnopqrstuvwxyz" streaming reveal="typewriter" resetKey="run-1">
        {text => <span data-testid="stream-text">{text}</span>}
      </StreamingContentStage>,
    )

    expect(screen.getByTestId('stream-text')).toHaveTextContent('')
    act(() => frames.shift()?.(0))
    expect(screen.getByTestId('stream-text').textContent).toBe('ab')

    rerender(
      <StreamingContentStage content="abcdefghijklmnopqrstuvwxyz" streaming={false} reveal="typewriter" resetKey="run-1">
        {text => <span data-testid="stream-text">{text}</span>}
      </StreamingContentStage>,
    )
    expect(screen.getByTestId('stream-text')).toHaveTextContent('abcdefghijklmnopqrstuvwxyz')
  })

  it('reveals a replacement stream from the beginning and instant reveal skips the queue', () => {
    const frames: FrameRequestCallback[] = []
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      frames.push(callback)
      return frames.length
    }))
    vi.stubGlobal('cancelAnimationFrame', vi.fn())

    const { rerender } = render(
      <StreamingContentStage content="" targetContent="first thought" streaming reveal="typewriter" resetKey="run-1">
        {text => <span data-testid="stream-text">{text}</span>}
      </StreamingContentStage>,
    )
    act(() => frames.shift()?.(0))
    expect(screen.getByTestId('stream-text').textContent).not.toBe('')

    rerender(
      <StreamingContentStage content="" targetContent="replacement" streaming reveal="typewriter" resetKey="run-2">
        {text => <span data-testid="stream-text">{text}</span>}
      </StreamingContentStage>,
    )
    expect(screen.getByTestId('stream-text')).toHaveTextContent('')

    rerender(
      <StreamingContentStage content="immediate text" targetContent="ignored while settled" streaming={false} reveal="instant" resetKey="run-2">
        {text => <span data-testid="stream-text">{text}</span>}
      </StreamingContentStage>,
    )
    expect(screen.getByTestId('stream-text')).toHaveTextContent('immediate text')
  })
})
