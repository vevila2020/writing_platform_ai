# Changelog

InkDock 仅在此记录用户可感知的重大功能、重要不兼容或数据变更、安全更新，以及影响核心流程的关键修复。完整记录可查看 [Git history](https://github.com/vevila2020/writing_platform_ai/commits/master)。

InkDock records major user-visible features, important compatibility or data changes, security updates, and fixes affecting core workflows. See [Git history](https://github.com/vevila2020/writing_platform_ai/commits/master) for the full history.

`Unreleased` 以最近一个已发布版本为基线，只描述升级用户最终可感知的净变化，不记录内部重构或未发布中间格式。

`Unreleased` describes the final user-visible changes since the latest release, excluding internal refactors and unpublished intermediate formats.

格式参考 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)。

## [Unreleased]

### Brief / 简要说明

#### 中文

- 写作侧栏新增「研究」：直接提问或发起扫榜、拆文，研究简报保存在 `setting/research/`；内置 10 张题材卡与 `novel-research`、`novel-deslop` Skill。
- 新增 Advance 长篇写作流程（`novel-serial`）：分阶段规划、按需召回技法与研究、审稿后提交连续性，支持旧书接入、改稿冲突核对和中断恢复。
- 新增只读 `check_writing` 工具，检查字数、截断、复读、工程词泄漏、细纲照搬、AI 句式，以及细纲与台账结构。
- 修复部分 Anthropic 兼容服务的并行工具结果配对错误，以及 OpenAI Responses 兼容服务返回内置工具（如联网搜索）时的流式中断。
- 修复浏览器工具运行结束后误报 `context canceled`；源码运行缺少 ripgrep 时启动会提示并从 PATH 回退。
- 思考内容在生成时自动展开、完成后收起；长思考展开不再卡顿或先出现空白。

#### English

- The Writing sidebar adds Research: ask questions directly or start market scans and deconstruction, with reports saved under `setting/research/`; adds 10 genre cards and the `novel-research` and `novel-deslop` Skills.
- Adds Advance (`novel-serial`) for staged planning, targeted craft and research recall, reviewed chapter continuity commits, existing-book adoption, revision conflicts, and interruption recovery.
- Adds the read-only `check_writing` tool for word count, truncation, repetition, engineering-word leakage, outline copying, AI-style prose patterns, and plan/ledger structure.
- Fixes parallel tool-result pairing with some Anthropic-compatible services and stream interruptions with OpenAI Responses services that return provider-side tools such as web search.
- Fixes false `context canceled` errors when closing browser tools, and warns at startup with a PATH fallback when a source run lacks ripgrep.
- Thinking expands while it streams and collapses when finished; long reasoning opens without stalling or blank frames.

### Added / 新增

- 写作侧栏新增「研究」标签页：支持直接研究提问、扫榜与拆文入口、题材卡，以及 `setting/research/` 下研究简报的浏览；研究请求在新的写作会话中发起，不覆盖当前对话。
- The Writing sidebar adds a Research view for direct questions, market scans, deconstruction prompts, genre cards, and saved reports under `setting/research/`; research opens in a new Writing conversation without replacing the current chat.
- 内置 10 张题材卡（玄幻/仙侠、都市异能、系统流、无限流、悬疑、科幻、历史、年代、古言、现言）。Skill front matter 新增 `keywords` 字段与 `genre-card` 能力；写作与通用 Agent 每轮最多自动加载一张匹配的题材卡，显式 `/skill` 优先，预算不足时自动题材卡会被跳过。
- Adds 10 built-in genre cards (xianxia/fantasy, urban powers, system, infinite flow, mystery, sci-fi, historical, period, ancient romance, modern romance). Skill front matter gains a `keywords` field and a `genre-card` capability; Writing and general Agents auto-load at most one matching card per turn. Explicit `/skill` requests win, and an automatic card is dropped when the context budget is tight.
- 内置 `novel-research`（扫榜、拆文与市场研究，要求标注来源和日期，取不到数据时说明缺口）与 `novel-deslop`（去 AI 味与表达修订）Skill；`novel-standard` 增加平台审稿规则参考，`novel-craft` 补充开篇、章首章尾与爽点编排。
- Adds the `novel-research` Skill (market scans, deconstruction, and market research with sources and dates, reporting gaps instead of inventing data) and the `novel-deslop` Skill (de-AI editing and prose revision). `novel-standard` gains a platform review rubric, and `novel-craft` expands its opening, chapter-hook, and payoff guidance.
- 新增 Advance 长篇写作 Skill：分阶段开书，逐章细纲进入规划侧栏，写前校验与按需召回，先审稿再提交；`prepare_advance` / `commit_advance` 将连续性底账与进度、人物状态、台账视图作为可恢复、可撤销的整组变更保存，拒绝过期正文版本，支持旧书初始化和改稿后的后文冲突处理。
- Adds the Advance writing Skill with staged book planning, chapter outlines in the planning sidebar, prerequisite checks, targeted recall, and review before commit. `prepare_advance` / `commit_advance` save canonical continuity and generated progress, character-state, and tracking views as one recoverable, undoable change group, rejecting stale prose and supporting existing-book initialization and later-chapter revision conflicts.
- 新增只读 `check_writing` 工具：检查章节字数、截断、占位/拒绝语、复读、工程词泄漏、细纲照搬和 AI 句式，也可校验细纲必填字段与连续性台账结构；在设置中归入独立的「检查写作」能力，默认对写作 Agent 开启。
- Adds the read-only `check_writing` tool for visible word count, truncation, placeholders/refusal text, repetition, engineering-word leakage, large verbatim outline copying, and AI-style prose patterns, plus chapter-plan and continuity-ledger validation. It is a separate “Check Writing” capability in Settings and is enabled for the Writing Agent by default.
- Agent 上下文会列出最近的 `setting/research/` 研究简报路径，正文按需读取，不直接注入。
- Agent context lists recent research reports under `setting/research/` as paths; report bodies are read on demand.
- 首页当前书籍操作区新增「编辑信息」入口。
- The Home current-book actions add an Edit Info entry.

### Changed / 变更

- 写作侧栏默认宽度调整为 288px 以容纳第四个标签；未手动调整过宽度的用户会自动升级，已调整的宽度保持不变。
- The Writing sidebar default width grows to 288px for the fourth view tab. Untouched defaults migrate automatically; user-resized widths are preserved.
- 模型发现列表不再额外标记「最新」，所有可用模型仍按更新时间排列。
- Model discovery no longer adds a “Latest” badge; available models still rank by creation time.

### Fixed / 修复

- Claude 等模型返回 `refusal` 时不再误报“Agent completed without user-visible assistant content”，而会按内容过滤中断展示本地化原因。
- When Claude or another model returns `refusal`, InkDock now reports a localized content-filter interruption instead of “Agent completed without user-visible assistant content”.
- 思考内容在生成时自动展开、结束后收起；思考改用轻量文本渲染，长内容展开不再卡顿或先出现空白。
- Thinking expands while streaming and collapses when finished; thinking uses lightweight text rendering so long content opens without stalls or blank frames.
- Anthropic Messages 协议将同一助手轮次的并行工具结果合并为一条 user 消息，并按工具调用顺序排列，修复部分兼容服务返回的 `tool_use` / `tool_result` 配对与顺序 400 错误。
- Anthropic Messages groups parallel tool results from one assistant turn into a single user message ordered by tool calls, fixing 400 responses for unpaired or misordered `tool_use` / `tool_result` blocks from some compatible services.
- OpenAI Responses 兼容服务返回内置工具（联网搜索、文件检索、代码解释器、MCP、图像生成等）时不再中断流式响应；原始输出项仍保留在续写负载中。
- OpenAI Responses-compatible services no longer interrupt the stream when returning provider-side tools such as web search, file search, code interpreter, MCP, or image generation; raw output items stay in the continuation payload.
- 浏览器工具在运行结束后关闭页面或浏览器时不再误报 `context canceled`。
- Browser tools no longer report `context canceled` when closing pages or the browser after a successful run.
- 源码运行且未安装 ripgrep 时，启动会提示安装方式并从 PATH 回退；发布包仍要求并优先使用内置 `rg`。
- Source runs without ripgrep now print an actionable startup notice and fall back to `PATH`; release packages still require and prefer the bundled `rg`.
- 使用 Gemini 原生协议时，工具 schema 不再发送其不支持的 `$comment` 注解，修复写作等 Agent 因批处理工具标记返回 400 错误。
- With the native Gemini protocol, tool schemas no longer send the unsupported `$comment` annotation, fixing 400 errors from batch-tool markers in Writing and other Agents.
- 写作插画只插入当前打开章节的光标位置，修复多章节 Tab 下重复插入或误写入其他章节的问题；源码编辑支持原位插入与撤销。
- Writing illustrations insert at the caret in the current chapter, fixing duplicate inserts and unintended edits to other chapter tabs; source editing supports insertion at the caret and undo.
- 错误诊断不再把接口路径中的 `/v1/messages` 当作消息字段，保留服务商返回的错误详情。
- Error diagnostics no longer mistake the `/v1/messages` endpoint path for a message field, preserving the provider's error detail.
- 长篇对话中打开审批模式菜单不再卡顿。
- Opening the approval-mode menu in long conversations no longer stutters.

## [v1.0.0] - 2026-11-04

### Brief / 简要说明

#### 中文

- 更名为 InkDock（墨栈 AI）：应用、命令行与开发脚本统一使用 InkDock 命名；已有 `.denova` 数据继续可用，新安装使用 `.inkdock`。
- 游戏控制台可就地调整本故事的叙事规则；叙事风格目录支持兼容导入 SillyTavern 预设。
- 章节结构支持章、卷—章、卷—章—节；已有作品可在书籍管理中预览调整并备份恢复。
- 新增图表统计，可按项目和日期查看稿量变化、章节进度、游戏活动与模型用量。
- 新增内置互动游戏创作 Skill，并支持按「游戏」分类管理 Skills。
- 更新前后可查看本次升级跨越的全部版本变化。
- 内置 EliasApi 中转站连接：粘贴 API Key 后实时获取模型列表并推荐最新对话模型；注册仍在 EliasApi 站点完成。
- 修复写作 Agent 忽略书籍简介，以及部分兼容服务下并行工具调用参数为空的问题。

#### English

- Renamed to InkDock: the app, CLI, and development scripts share the InkDock naming; existing `.denova` data keeps working and new installs use `.inkdock`.
- Tune story-specific narrative rules in the Game console, and import SillyTavern presets from the narrative style directory.
- Choose chapters, volumes and chapters, or volumes, chapters and sections; preview and back up structure changes in Book Management.
- Adds chart-based statistics for manuscript changes, chapter progress, Game activity, and model usage by project and date.
- Adds a built-in interactive-game design Skill and a Game category for Skills.
- Shows changes across every version included in an upgrade, before and after installation.
- Adds a recommended EliasApi relay connection: paste an API key to discover models live and use the newest chat model; registration stays on the EliasApi site.
- Fixes writing Agents ignoring the book synopsis, and empty parallel tool-call arguments with some compatible services.

### Added / 新增

- 游戏控制台支持就地开关、编辑当前故事的叙事规则，可恢复预设或另存为共享风格。叙事风格目录兼容导入 SillyTavern 聊天补全预设：直接导入保留条目、顺序与默认启停，也可由 AI 整理；不执行宏或扩展脚本。
- The Game console can toggle and edit story-specific narrative rules, restore preset defaults, or save a reusable style. The narrative style directory imports SillyTavern Chat Completion presets with their text, order, and default activation states, or optional AI adaptation; macros and extension scripts are not executed.
- 新建和导入作品可选择一至三层章节结构，支持空卷、空章与分节编辑；已有作品可在书籍管理中预览转换，自动保存恢复版本，目录、Agent 阅读顺序与 TXT 导出保持一致。
- New and imported books support one to three manuscript levels, empty volumes and chapters, and section editing. Book Management previews conversions and saves recovery versions; navigation, Agent reading order and TXT exports share the hierarchy.
- 共通一级菜单新增「统计」，提供写作趋势、稿量分布、章节进度、游戏每日回合与故事线活动、模型调用与 Token 图表；稿量变化和模型用量从升级后开始记录，游戏活动从现存故事日志读取。可在常规设置中关闭本地统计，保留已有记录与创作内容。
- A shared Statistics destination adds writing trends, manuscript and chapter charts, daily Game turns and story activity, model calls, and token breakdowns. Manuscript changes and model usage are recorded after upgrading; Game activity comes from retained story journals. Local statistics can be disabled in General settings while keeping existing records and creative content.
- 内置互动游戏创作 Skill，支持游戏构思与核心循环、世界与角色资料库、状态系统、规则判定、游戏规划、开局回合和跑团诊断。
- A built-in interactive-game design Skill covers game premises and loops, world and character Lore, state systems, rules and checks, planning, openings and turns, and play diagnostics.
- 互动游戏创作 Skill 补充经典网文向的成长体系、爽点结构、资源与社会资本、世界阶梯、副本/规则恐怖/都市隐杀和外观沉浸感设计。
- The interactive-game design Skill now covers classic web-novel progression, payoff structure, resources and social capital, world tiers, instance/rule-horror/urban-undercover patterns, and sensory immersion.
- Skills 库新增「游戏」分类，可在创建、配置和筛选中使用。
- The Skills library adds a Game category for creation, configuration, and filtering.
- 设置 → 语言模型中新增 EliasApi 推荐连接。首次配置语言模型时点击「使用 EliasApi」会预填中转站地址和 Responses 协议；粘贴 API Key 后实时读取 `/v1/models`，按更新时间排列并标记最新对话模型。服务商返回的模型显示名会保留，模型列表无需随客户端升级更新。
- Settings → Language Models adds a recommended EliasApi connection. When no language model is configured, “Use EliasApi” prefills the relay address and Responses protocol; after pasting an API key, InkDock reads `/v1/models` live, sorts by creation time, and marks the newest chat model. Provider display names are preserved, and the model list updates without a client release.

### Changed / 变更

- 应用、命令行、开发脚本与配置键统一使用 InkDock 命名；升级后仍会读取已有的 `.denova` 数据目录、`denova_dir` 与 `DENOVA_*` 配置，新安装使用 `.inkdock`。
- The app, CLI, development scripts, and configuration keys now use InkDock naming. Upgrades still read existing `.denova` data directories, `denova_dir`, and `DENOVA_*` settings; new installs use `.inkdock`.
- 软件更新展示当前版本到目标版本的变化，按版本阅读简要说明和详情；更新完成后保留本次升级范围，并提供完整历史入口。
- Software updates show changes from the current version to the target, with per-version summaries and details. Completed upgrades retain their comparison range and link to the full history.

### Fixed / 修复

- 写作 Agent 每轮读取书名、作者与简介，生成内容遵循简介中的人物与设定；游戏不受影响。
- Writing Agents now read the book title, author, and synopsis each turn, so generated text follows the characters and premise in the synopsis. Game is unaffected.
- 使用 OpenAI Responses 兼容服务时，若服务未逐段返回工具参数，会从完成事件补回完整参数，不再因参数为空导致并行读取等工具调用失败。
- With OpenAI Responses-compatible services that omit streamed tool-argument deltas, complete arguments are recovered from completion events, so parallel reads and other tool calls no longer fail on empty arguments.
