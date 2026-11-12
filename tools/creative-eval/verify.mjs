import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { mkdtemp, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import { createServer } from 'node:http'
import os from 'node:os'
import path from 'node:path'
import { stringify } from 'smol-toml'
import { evaluateCreative, toolDirectory } from './run.mjs'
import InkDockProvider from './provider.mjs'

// Explicit integration verification with a real product backend and local
// model fixture. No development server, real credentials, or paid APIs.
const temporary = await mkdtemp(path.join(os.tmpdir(), 'inkdock-eval-verify-'))
let calls = 0
let cancelNextRequest
const controller = new AbortController()
for (const signal of ['SIGINT', 'SIGTERM']) process.once(signal, () => controller.abort(new Error(`Verification cancelled by ${signal}`)))
const fixture = createServer(async (request, response) => {
  try {
    let raw = ''
    for await (const chunk of request) raw += chunk
    const body = JSON.parse(raw)
    if (cancelNextRequest) {
      const cancel = cancelNextRequest
      cancelNextRequest = undefined
      cancel()
      return
    }
    assert.ok(++calls < 100, 'Fixture exceeded its expected bounded request count')
    const messages = JSON.stringify(body.messages)
    const game = body.tools?.some(item => item.function.name === 'submit_interactive_turn')
    let narrative = 'Mira cannot create water. The dry cup remains empty; her copper token is unchanged.'
    let tools = []
    if (game) {
      const latestUser = [...body.messages].reverse().find(item => item.role === 'user')
      const action = JSON.stringify(latestUser)
      if (action.includes('alternate branch, an archivist')) narrative = 'The archivist tells Mira the password NIGHTGLASS-742. She remembers it and remains outside.'
      else if (action.includes('currently know about entering')) narrative = 'Mira knows no password and has met no archivist. She waits outside the locked archive.'
      else if (messages.includes('locked archive')) narrative = 'Mira waits outside the locked archive, knowing no secrets.'
      tools = [{ name: 'submit_interactive_turn', arguments: {
        state_changes: [
          { op: 'replace', actor_id: 'story', field_id: '当前详细地点', value: messages.includes('locked archive') ? 'Outside the archive' : 'Dry stone room' },
          { op: 'replace', actor_id: 'story', field_id: '当前事件', value: narrative },
        ], choices: ['Wait', 'Observe'],
      } }]
    } else if (body.tools?.some(item => item.function.name === 'write')) {
      const hasResult = body.messages.some(item => item.role === 'tool')
      if (messages.includes('setting/chapter-plans/ch00002-the-signal.md')) {
        if (!hasResult) {
          tools = [
            { name: 'write', arguments: { path: 'setting/chapter-plans/ch00002-the-signal.md', content: [
              '## 细纲（第 2 章）',
              '',
              '### 第 2 章：the signal',
              '',
              '- 核心事件：Mira decides to trace the letter seal without opening it.',
              '- 字数目标：800 字',
              '- 目标情绪：从警惕走到克制的决心。',
              '- 主角目标 / 关键选择：查明寄信人，但不能惊动站长。',
              '- 章首钩子：印章上的纹路在灯下动了一下。',
              '- 章尾钩子：站长说他也收到过同样的信。',
              '',
              '#### 这章怎么走',
              '',
              '- 起因：Mira 在灯下检查信封。',
              '- 发展：她认出印章纹路和站长的账本有关。',
              '- 转折：她决定不拆信，先去找站长。',
              '- 高潮：站长承认自己也收到过同样的信。',
              '- 结尾：站长准备讲寄信人的事。',
            ].join('\n') } },
            { name: 'write', arguments: { path: 'chapters/ch00002-the-signal.md', content: 'Mira turned the sealed letter over in the lamplight.\n\nThe wax bore a spiral she had seen once before, on the station keeper\'s ledger.\n\nShe did not break the seal. She copied the spiral onto the back of her hand and went to find him.\n\nThe keeper looked at the mark and said, "I received one too."\n' } },
            { name: 'write', arguments: { path: 'setting/progress.md', content: '# Progress\n\n- Current chapter: 2\n- Chapter 1: Mira arrived at the dry station and found a sealed letter on the cot. She has not opened it.\n- Chapter 2: Mira traced the letter seal to the station keeper, who admitted receiving the same letter.\n- Next: hear the keeper\'s story about the sender.\n' } },
            { name: 'write', arguments: { path: 'setting/character-states.md', content: '## Mira\n\n- Latest appearance: Chapter 2\n- Location: the dry station\n- Physical state: tired, alert\n- Goal: learn who sent the letter without alerting the keeper\n- Possessions: one sealed letter, one copper token\n- Knowledge: the keeper received the same letter; the letter is still unopened\n' } },
            { name: 'write', arguments: { path: 'setting/tracking.md', content: '# 连续性台账\n\n## 当前位置\n\n- 当前章：第 2 章\n- 故事时间：到站第二天夜里\n- 场景：干站站长室\n\n## 长期约束\n\n- 不能提前揭开信的内容和寄信人。\n\n## 活跃伏笔\n\n- F001｜那封信的寄信人｜埋设第 1 章｜预计回收第 3 章｜重要度高｜状态推进中\n\n## 作者真相 / 读者已知\n\n### 作者真相（尚未揭示）\n\n- 信是站长的旧友寄来的。\n\n### 读者已知\n\n- Mira 和站长都收到过同样的信。\n\n## 下一章承诺\n\n- 站长要讲旧友的事。\n\n## 连贯性风险\n\n- 不能提前说出寄信人的身份。\n\n## 近三章速记\n\n- 第 1 章：Mira 到站，发现密封信。\n- 第 2 章：Mira 追查印章，站长承认也收到同样的信。\n' } },
          ]
        }
        narrative = 'Saved the requested plan, chapter, and state files.'
      } else {
        if (!hasResult) {
          const scoped = messages.includes('Mira paused at the threshold.')
          tools = [{ name: 'write', arguments: { path: 'chapters/scene.md', content: scoped
            ? '# Arrival\n\nMira paused at the threshold.\n\nThe brass clock was silent.\n'
            : '# The empty cup\n\nMira cannot create water. Her cup remains empty.\n' } }]
          if (body.model === 'fixture-bad' && scoped) tools.push({ name: 'write', arguments: { path: 'chapters/untouched.md', content: 'This unauthorized edit must fail evaluation.\n' } })
        }
        narrative = 'Saved the requested chapter.'
      }
    } else {
      if (body.model === 'fixture-grader') {
        assert.ok(messages.includes('Original user request:'), 'Semantic grader did not receive the original request')
        assert.ok(messages.includes('Initial project facts:'), 'Semantic grader did not receive fixture facts')
      }
      narrative = JSON.stringify({ pass: true, score: 1, reason: 'Deterministic harness fixture; real semantics are not evaluated here.' })
    }
    const tool_calls = tools.map((tool, index) => ({ id: `fixture-${calls}-${index}`, index, type: 'function', function: { name: tool.name, arguments: JSON.stringify(tool.arguments) } }))
    const usage = { prompt_tokens: 10, completion_tokens: 10, total_tokens: 20 }
    if (!body.stream) {
      response.writeHead(200, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify({ id: 'fixture', choices: [{ index: 0, message: { role: 'assistant', content: narrative }, finish_reason: 'stop' }], usage }))
      return
    }
    response.writeHead(200, { 'Content-Type': 'text/event-stream' })
    const frame = { id: `fixture-${calls}`, object: 'chat.completion.chunk', created: 1, model: 'fixture' }
    response.write(`data: ${JSON.stringify({ ...frame, choices: [{ index: 0, delta: { role: 'assistant', content: narrative, ...(tools.length ? { tool_calls } : {}) }, finish_reason: tools.length ? 'tool_calls' : 'stop' }] })}\n\n`)
    response.end(`data: ${JSON.stringify({ ...frame, choices: [], usage })}\n\ndata: [DONE]\n\n`)
  } catch (error) {
    response.writeHead(500, { 'Content-Type': 'application/json' })
    response.end(JSON.stringify({ error: error.message }))
  }
})

try {
  fixture.listen(0, '127.0.0.1')
  await once(fixture, 'listening')
  const baseURL = `http://127.0.0.1:${fixture.address().port}/v1`
  const settingsFile = path.join(temporary, 'settings.toml')
  const secret = 'fixture-secret-not-for-report-export'
  const modelSettings = {
    model_endpoints: [{ id: 'fixture', provider: 'openai-compatible', protocol: 'openai-chat-completions', base_url: baseURL, api_key: secret }],
    model_profiles: ['fixture', 'fixture-bad', 'fixture-grader'].map(id => ({ id, endpoint_id: 'fixture', model: id, context_window_tokens: 100000 })),
    agent_models: { default: { profile_id: 'fixture', thinking_level: 'off' } },
  }
  await writeFile(settingsFile, stringify(modelSettings))
  process.env.INKDOCK_EVAL_FIXTURE_GRADER_KEY = secret
  const configPath = path.join(temporary, 'run.json')
  const configuration = { settings_file: settingsFile,
    variants: [{ label: 'fixture', profile_id: 'fixture' }],
    grader: { profile_id: 'fixture-grader' },
  }
  await writeFile(configPath, JSON.stringify(configuration))
  const executable = path.join(temporary, process.platform === 'win32' ? 'inkdock.exe' : 'inkdock')
  const build = spawn('go', ['build', '-o', executable, './cmd/inkdock'], { cwd: path.resolve(toolDirectory, '../..'), stdio: 'inherit', signal: controller.signal })
  assert.equal((await once(build, 'exit'))[0], 0)
  const positive = await evaluateCreative({ configPath, binary: executable, signal: controller.signal })
  assert.equal(positive.summary.stats.successes, 5, JSON.stringify(positive.summary.results.map(item => ({ description: item.testCase?.description, error: item.error, reason: item.gradingResult?.reason }))))
  assert.equal(positive.summary.stats.errors, 0)
  assert.equal(positive.summary.stats.failures, 0)
  const report = await readFile(path.join(positive.outputDirectory, 'report.json'), 'utf8')
  assert.ok(!report.includes(secret), 'Report exported a credential')
  assert.ok((await readFile(path.join(positive.outputDirectory, 'report.html'), 'utf8')).includes('writing-scoped-edit'))
  for (const entry of await readdir(positive.outputDirectory, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    const evidence = JSON.parse(await readFile(path.join(positive.outputDirectory, entry.name, 'evidence.json'), 'utf8'))
    if (evidence.case_id === 'writing-serial-plan-first') {
      assert.match(evidence.files['setting/chapter-plans/ch00002-the-signal.md'], /Mira decides to trace the letter seal/)
      assert.match(evidence.files['setting/tracking.md'], /F001/)
    }
    await assert.rejects(readFile(path.join(positive.outputDirectory, entry.name, '.inkdock/config.toml')), { code: 'ENOENT' })
    const remaining = await readdir(path.join(positive.outputDirectory, entry.name, '.inkdock'))
    assert.ok(!remaining.some(file => file.startsWith('config.toml')), 'Temporary credential backup was retained')
  }
  configuration.variants = [{ label: 'baseline', profile_id: 'fixture' }, { label: 'candidate', profile_id: 'fixture-bad' }]
  configuration.grader = { id: 'openai:chat:fixture-grader', config: { apiBaseUrl: baseURL, apiKeyEnvar: 'INKDOCK_EVAL_FIXTURE_GRADER_KEY' } }
  await writeFile(configPath, JSON.stringify(configuration))
  const negative = await evaluateCreative({ configPath, binary: executable, filter: 'writing-scoped-edit', signal: controller.signal })
  assert.equal(negative.summary.stats.successes, 1)
  assert.equal(negative.summary.stats.errors, 0)
  assert.equal(negative.summary.stats.failures, 1, 'The deliberate unauthorized edit was not detected')
  const cancellation = new AbortController()
  cancelNextRequest = () => cancellation.abort(new Error('Verification requested cancellation'))
  const scenario = JSON.parse(await readFile(new URL('./cases.json', import.meta.url), 'utf8'))[0]
  const cancelled = await new InkDockProvider({ label: 'cancelled', variant: { profile_id: 'fixture' },
    settings: modelSettings, binary: executable, skillsDirectory: path.resolve(toolDirectory, '../../skills'),
    outputDirectory: positive.outputDirectory }).callApi(scenario.prompt, { vars: { scenario } }, { abortSignal: cancellation.signal })
  assert.equal(cancelNextRequest, undefined, 'Cancellation never reached an in-flight model request')
  assert.ok(cancelled.error, 'Cancellation was recorded as successful output')
  const cancelledDirectory = path.dirname(cancelled.metadata.evidence_path)
  const cancelledEvidence = JSON.parse(await readFile(cancelled.metadata.evidence_path, 'utf8'))
  assert.notEqual(cancelledEvidence.complete, true)
  assert.ok(!(await readdir(path.join(cancelledDirectory, '.inkdock'))).some(file => file.startsWith('config.toml')))
  console.log('[creative-eval] Verified five product scenarios, two-variant regression detection, report export, credential cleanup, and in-flight cancellation')
} finally {
  fixture.closeAllConnections()
  await new Promise(resolve => fixture.close(resolve))
  await rm(temporary, { recursive: true, force: true })
}
