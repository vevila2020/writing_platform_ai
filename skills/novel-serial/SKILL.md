---
name: novel-serial
description: Advance 长篇创作工作流。用于分阶段开书、卷/章组/逐章规划、正文续写、连续多章和改稿。按需召回创作技法、研究与作者偏好，写前校验细纲与前章状态，审稿修订后通过结构化章节事务提交连续性。
category: writing
capabilities:
  - writing-workflow
agent: ide
---

# Advance

让长篇创作从读者期待走到实际兑现，并在每章结束后留下可校验、可接续的当前事实。系统约束、作者本轮明确要求和已确认的书稿事实优先；研究、题材经验和默认建议只补缺项。

## 先确定范围

| 作者意图 | 本轮交付与停点 |
| --- | --- |
| 讨论、诊断、比较 | 对话中回答，不自动改书稿 |
| 开书 | 按方向 → 稳定设定 → 卷/章组 → 首批细纲分阶段交接；已有授权不重复确认，未授权正文不写 |
| 全书、卷、章组、细纲 | 只交付点名层级，不自行展开下一层或正文 |
| 写/续写第 N 章 | 完整单章闭环，结束后停 |
| 继续写、接着写 | 默认下一章；“继续/按这个来”沿用上一轮对象，不把规划确认变成正文授权 |
| 日更、连续多章 | 按指定数量串行；未给数量先建议 2–3 章并确认范围，不暗中扩大 |
| 修改/重写 | 只改点名内容，核对后文影响，不自动重写相邻章 |

片段、标点和格式修改遵守作者范围，不启动无关的整章创作。

## 按任务加载

叙事正文或相关规划开始前，用 `skill` 加载 `novel-craft`。章节读 `skill://novel-craft/references/scene-and-payoff.md`；连续性与读者契约补读 `skill://novel-craft/references/long-form.md` 的相关小节；规划读 `skill://novel-craft/references/premise-and-outline.md`。纯标点修改不加载整套技法。

只读当前需要的参考，不整套灌入：

- 规划：`skill://novel-serial/references/workflow-plan.md`
- 正文：`skill://novel-serial/references/workflow-chapter.md`
- 连续多章：`skill://novel-serial/references/workflow-daily.md`
- 改稿：`skill://novel-serial/references/workflow-revision.md`
- 写前材料、研究与偏好：`skill://novel-serial/references/context-and-recall.md`
- 细纲：`skill://novel-serial/references/chapter-plan-template.md`
- 连续性提交：`skill://novel-serial/references/tracking-template.md`
- 检查与审稿：`skill://novel-serial/references/continuity-and-quality.md`、`skill://novel-serial/references/review-rubric.md`

## 单章闭环

```text
确定范围与本章职责
→ 校验细纲，prepare_advance 核对前章
→ 定点读取前文、Lore、技法、研究和作者约束
→ 写一章正文
→ check_writing + 只读审稿
→ 修必要问题，重跑检查
→ 再次 prepare_advance 取得最终版本凭据
→ commit_advance 一次提交本章变化
→ 读回结果，向作者汇报
```

1. **细纲先行**：实际填写核心事件、目标情绪、主角目标/关键选择、章尾钩子、字数预算；先 `check_writing action=plan`。缺内容按既有方向补齐，重大剧情或人物决定未授权则先提案。
2. **逐章串行**：上一章正文、审稿和连续性提交完成后才开始下一章。不并发写章，不让审稿者改文件。
3. **提交前审稿**：本 Skill 要求整章定稿前完成只读审稿。宿主提供 `send` / `await` / `list_agents` 时，委派一个 `general-purpose` 审稿者，等待实际结果。委派不可用或作者要求独立完成时按同一规则自审，不能声称做过独立审稿。不使用不存在的 `task` 工具。
4. **匹配最终正文**：正文或细纲修改会使旧准备凭据过期。修订后重新准备，按真实变化提交，不重放旧版状态。

## 文件与工具的职责

- 稳定身份、背景、规则、地点、势力、能力体系 → Lore。用 `query_lore_items` 读取 `detail=full`，用 `write_lore_items` 更新稳定事实。
- 未来剧情 → `setting/outline.md`、`setting/chapter-groups/`、`setting/chapter-plans/`。计划、猜测和作者真相不得冒充已发生事实。
- 当前连续性 → `commit_advance` 的语义更新。`setting/continuity.json` 是书籍底账；进度、人物状态、台账是生成视图，不分别 `write/edit`，不新增另一套手写追踪。
- `prepare_advance` 返回活跃人物、伏笔、作者真相/读者所知、近三章和冲突。完整事实保留在底账，只有当前需要的集合进入写前材料。
- 已有书稿首次启用按追踪参考初始化；原进度保留，替换前内容由工作区变更历史保存，可整组撤销。不把初始化当作重新创作。

这些工具是应用层书稿能力，不管理 Agent 的 Goal/Todo/执行恢复。会话恢复沿用产品 journal；其他运行时按实际宿主能力调用，不猜委派协议。

工具缺失时可讨论、规划或交付明确标记为草稿的正文，说明连续性尚未提交。不得用 shell 手改底账、伪造回执或自动进入下一章。检查每次调用的成功和失败；受理不等于完成。

## 质量与作者参与

先修截断、乱码、占位、复读、工程文本泄漏、细纲照抄、严重字数越界与事实矛盾。风格建议按阅读效果判断，保留作者有意的口语、短句、留白和重复台词，不机械替换。

作者本轮字数范围优先于细纲目标。带外先说明差额，修订或取得作者接受后再提交，不自行注水或硬删。硬伤不能用长度接受绕过。不设置固定修订轮数或总时长；作者可取消，缺少决定或材料时说明具体阻碍。

一次性细节可直接写；可复用事实需登记；新增主线、力量规则、重大反转、提前揭晓或改变已定人物决定，先说明影响并取得选择。范围内执行不重复索取授权。

## 汇报

只讲故事：哪章写了/改了什么、人物与局势怎样变化、停在哪里、哪些事需要作者决定。检查结论一句即可，不默认展示工具、事务字段、内部编号或完整审稿报告。失败时说“草稿已保存，连续性尚未提交”，不能说本章已完成。
