import { expect, test } from '../support/fixtures'
import { createAndOpenBook, createProjectFile, readProjectFile } from '../support/api'
import { openWritingAgent } from '../support/agent-chat'

for (const theme of ['dark', 'light'] as const) {
  test(`Writing inserts an illustration only at the current chapter caret in ${theme}`, async ({ page, request }) => {
    const book = await createAndOpenBook(request, `Illustration insertion ${theme}`)
    const firstPath = 'chapters/insert-first.md'
    const secondPath = 'chapters/insert-second.md'
    const thirdPath = 'chapters/insert-third.md'
    const original = '# Chapter\n\nBeforeAfter\n'
    for (const path of [firstPath, secondPath, thirdPath]) {
      await createProjectFile(request, book.projectId, path, original)
    }
    const settings = await (await request.get('/api/settings')).json()
    await page.route(/\/api\/(?:projects\/[^/]+\/)?settings$/, route => route.fulfill({ json: {
      ...settings, effective: { ...settings.effective, theme, language: 'zh-CN' },
    } }))
    const illustration = {
      schema: 'chapter_illustration.v1', chapter_path: firstPath,
      image_path: 'assets/illustrations/generated.png', meta_path: 'assets/illustrations/meta.json',
      markdown: '![Generated scene](assets/illustrations/generated.png)', alt_text: 'Generated scene',
      profile_id: 'default', provider: 'openai', model: 'test-image',
    }
    await page.route('**/session/messages?*', route => route.fulfill({ json: {
      messages: [{ id: 'generated-illustration', role: 'assistant', metadata: {
        tool_presentation: { call: 'image', result: 'image' },
      }, parts: [{ type: 'dynamic-tool', toolName: 'image', toolCallId: 'illustration-call',
        state: 'output-available', input: {}, output: illustration,
      }] }], page: { has_more: false, total: 1 },
    } }))
    await page.route('**/files/asset?*', route => route.fulfill({
      contentType: 'image/png',
      body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=', 'base64'),
    }))
    await page.setViewportSize({ width: 1440, height: 960 })
    await page.goto('/')
    await openWritingAgent(page)
    await page.getByRole('button', { name: '项目文件', exact: true }).click()
    const editor = page.locator('.tiptap.ProseMirror:not(.nova-composer-token-editor):visible')
    const openChapter = async (name: string) => {
      await page.getByRole('treeitem', { name, exact: true }).click()
      await expect(editor).toContainText('Chapter')
    }
    await openChapter('insert-first.md')
    await openChapter('insert-second.md')
    await editor.locator('p').filter({ hasText: 'BeforeAfter' }).click()
    await page.keyboard.press('Home')
    for (let index = 0; index < 'Before'.length; index += 1) await page.keyboard.press('ArrowRight')
    await page.getByRole('button', { name: '插入正文', exact: true }).click()
    await expect(editor.locator('img')).toHaveCount(1)
    await expect(editor.locator('p')).toHaveText(['Before', 'After'])
    await page.screenshot({ path: test.info().outputPath(`illustration-${theme}.png`), animations: 'disabled' })

    await openChapter('insert-first.md')
    await expect(editor.locator('img')).toHaveCount(0)
    await openChapter('insert-third.md')
    await expect(editor.locator('img')).toHaveCount(0)
    expect((await readProjectFile(request, book.projectId, firstPath)).content).toBe(original)
    expect((await readProjectFile(request, book.projectId, thirdPath)).content).toBe(original)
    const inserted = (await readProjectFile(request, book.projectId, secondPath)).content
    expect(inserted).toMatch(/Before[\s\S]*!\[Generated scene\][\s\S]*After/)
    expect(inserted.split(illustration.image_path)).toHaveLength(2)

    await page.getByRole('tab', { name: /insert second/ }).click()
    await expect(editor.locator('img')).toHaveCount(1)
    await page.getByRole('button', { name: /关闭.*insert second/ }).click()
    await openChapter('insert-second.md')
    await expect(editor.locator('img')).toHaveCount(1)
    await page.getByRole('button', { name: '插入正文', exact: true }).click()
    await expect(editor.locator('img')).toHaveCount(2)

    await openChapter('insert-third.md')
    const sourceToggle = page.getByRole('radio', { name: '源码', exact: true }).filter({ visible: true })
    await sourceToggle.click()
    const source = page.locator('.monaco-editor:visible')
    await source.locator('.view-line').filter({ hasText: 'BeforeAfter' }).click()
    await page.keyboard.press('Home')
    for (let index = 0; index < 'Before'.length; index += 1) await page.keyboard.press('ArrowRight')
    await page.getByRole('button', { name: '插入正文', exact: true }).click()
    await expect(sourceToggle).toBeChecked()
    await page.keyboard.press('Control+s')
    await expect.poll(async () => (await readProjectFile(request, book.projectId, thirdPath)).content)
      .toMatch(/Before!\[Generated scene\]\([^\n]+\)After/)
    await source.locator('.view-line').first().click()
    await page.keyboard.press('Control+z')
    await page.keyboard.press('Control+s')
    await expect.poll(async () => (await readProjectFile(request, book.projectId, thirdPath)).content).toBe(original)
  })
}
