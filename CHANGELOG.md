# Changelog

InkDock 仅在此记录用户可感知的重大功能、重要不兼容或数据变更、安全更新，以及影响核心流程的关键修复。完整记录可查看 [Git history](https://github.com/vevila2020/writing_platform_ai/commits/master)。

InkDock records major user-visible features, important compatibility or data changes, security updates, and fixes affecting core workflows. See [Git history](https://github.com/vevila2020/writing_platform_ai/commits/master) for the full history.

`Unreleased` 以最近一个已发布版本为基线，只描述升级用户最终可感知的净变化，不记录内部重构或未发布中间格式。

`Unreleased` describes the final user-visible changes since the latest release, excluding internal refactors and unpublished intermediate formats.

格式参考 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)。

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
