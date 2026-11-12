import { expect, test } from '../support/fixtures'
import { createAndOpenBook, createProjectFile } from '../support/api'

for (const theme of ['dark', 'light'] as const) {
  test(`Advance selection persists and chapter plans appear in ${theme}`, async ({ page, request }) => {
    test.setTimeout(90_000)
    const book = await createAndOpenBook(request, `Advance ${theme}`)
    const english = theme === 'light'
    await page.route(/\/api\/(?:projects\/[^/]+\/)?settings$/, async route => {
      if (route.request().method() !== 'GET') return route.continue()
      const response = await route.fetch()
      const settings = await response.json()
      await route.fulfill({ response, json: {
        ...settings, effective: { ...settings.effective, theme, language: english ? 'en-US' : 'zh-CN' },
      } })
    })
    await page.setViewportSize({ width: 1440, height: 960 })
    await page.goto('/')
    await page.getByLabel(english ? 'Workbench sidebar' : '工作台侧边栏')
      .getByRole('button', { name: english ? 'Writing' : '写作', exact: true }).click()
    await expect(page.getByText(english ? 'No outlines yet' : '还没有细纲', { exact: true })).toBeVisible()

    const latestTitle = `Chapter ten: ${'A choice with lasting consequences '.repeat(8)}`.trim()
    await createProjectFile(request, book.projectId, 'setting/chapter-groups/group01-opening.md', '# Opening group\n\nThe group promise.')
    await createProjectFile(request, book.projectId, 'setting/chapter-plans/ch00010-choice.md', `# ${latestTitle}\n\nA completed detailed plan.`)
    await createProjectFile(request, book.projectId, 'setting/chapter-plans/ch00002-door.md', '# Chapter two\n\nAn earlier detailed plan.')
    await page.reload()
    const planning = page.getByLabel(english ? 'Book outline' : '作品大纲', { exact: true })
    await expect(planning.getByRole('button', { name: latestTitle, exact: true })).toBeVisible()
    const history = planning.getByRole('button', { name: english ? /^Outline History/ : /^历史细纲/ })
    await history.click()
    await expect(history).toHaveAttribute('aria-expanded', 'true')
    await expect(planning.getByRole('button', { name: 'Chapter two', exact: true })).toBeVisible()
    await expect(planning.getByRole('button', { name: 'Opening group', exact: true })).toBeVisible()
    await page.screenshot({ path: test.info().outputPath(`plans-${theme}.png`), animations: 'disabled' })

    const showAgent = page.getByRole('button', { name: english ? 'Show Writing Agent' : '显示创作 Agent', exact: true })
    if (await showAgent.count()) await showAgent.click()
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 960 })
      if (width === 390) {
        await page.reload()
        await page.getByRole('tab', { name: 'Agent', exact: true }).click()
      }
      await page.getByRole('button', { name: english ? 'Input actions' : '输入动作', exact: true }).click()
      const skillMenu = page.getByRole('menuitem', { name: english ? 'Writing Skill' : '写作 Skill', exact: true })
      await skillMenu.click()
      const advance = page.getByRole('menuitem', { name: /^Advance\b/ })
      await expect(advance).toBeVisible()
      if (width === 1440) {
        await advance.click()
        await expect.poll(async () => (await (await request.get('/api/settings')).json()).effective.writing_skill_default)
          .toBe('novel-serial')
      }
      await expect(skillMenu).toContainText('Advance')
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await page.screenshot({ path: test.info().outputPath(`advance-${theme}-${width}.png`), animations: 'disabled' })
      await page.keyboard.press('Escape')
      await page.keyboard.press('Escape')
    }
    const saved = await (await request.get(`/api/projects/${book.projectId}/settings`)).json()
    expect(saved.effective.writing_skill_default).toBe('novel-serial')
  })
}
