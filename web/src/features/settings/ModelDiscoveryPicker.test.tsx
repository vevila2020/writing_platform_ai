import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { discoverModels } from './api'
import { ModelDiscoveryPicker } from './ModelDiscoveryPicker'

vi.mock('./api', () => ({
  discoverModels: vi.fn(),
}))

describe('ModelDiscoveryPicker', () => {
  beforeEach(() => {
    vi.mocked(discoverModels).mockReset()
  })

  it('adds multiple discovered models while disabling existing ones', async () => {
    vi.mocked(discoverModels).mockResolvedValue({
      provider: 'openai',
      protocol: 'openai-chat-completions',
      base_url: 'https://models.example/v1',
      models: [
        { id: 'model-a', display_name: 'Model A' },
        { id: 'model-b', display_name: 'Model B' },
        { id: 'model-existing', display_name: 'Existing Model' },
      ],
    })
    const onAdd = vi.fn()
    const user = userEvent.setup()
    render(
      <ModelDiscoveryPicker
        endpoint={{ id: 'shared', provider: 'openai', protocol: 'openai-chat-completions', base_url: 'https://models.example/v1' }}
        existingModels={['model-existing']}
        onAdd={onAdd}
      />,
    )

    await user.click(screen.getByRole('button', { name: '选择模型' }))
    const list = await screen.findByRole('listbox', { name: '可用模型' })
    const existing = within(list).getByRole('option', { name: /Existing Model/ })
    expect(existing).toBeDisabled()

    await user.click(within(list).getByRole('option', { name: /Model A/ }))
    await user.click(within(list).getByRole('option', { name: /Model B/ }))
    await user.click(screen.getByRole('button', { name: '添加选中的 2 个模型' }))

    expect(onAdd).toHaveBeenCalledWith([
      { id: 'model-a', display_name: 'Model A' },
      { id: 'model-b', display_name: 'Model B' },
    ])
  })

  it('ranks chat models by creation time and marks the newest as recommended', async () => {
    vi.mocked(discoverModels).mockResolvedValue({
      provider: 'openai',
      protocol: 'openai-chat-completions',
      base_url: 'https://models.example/v1',
      models: [
        { id: 'image-new', display_name: 'Image New', created: 300 },
        { id: 'chat-old', display_name: 'Chat Old', created: 100 },
        { id: 'chat-new', display_name: 'Chat New', created: 200 },
        { id: 'chat-mid', display_name: 'Chat Mid', created: 150 },
      ],
    })
    const user = userEvent.setup()
    render(
      <ModelDiscoveryPicker
        endpoint={{ id: 'shared', provider: 'openai', protocol: 'openai-chat-completions', base_url: 'https://models.example/v1' }}
        existingModels={[]}
        onAdd={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '选择模型' }))
    const list = await screen.findByRole('listbox', { name: '可用模型' })
    const options = within(list).getAllByRole('option')
    expect(options.map((option) => option.textContent)).toEqual([
      expect.stringContaining('Chat New'),
      expect.stringContaining('Chat Mid'),
      expect.stringContaining('Chat Old'),
      expect.stringContaining('Image New'),
    ])
    expect(within(options[0]).getByText('最新')).toBeInTheDocument()
  })
})
