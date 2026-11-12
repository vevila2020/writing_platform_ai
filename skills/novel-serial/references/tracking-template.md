# 连续性事务

`setting/continuity.json` 是书籍底账，不是 Agent 会话恢复文件。模型只提交一份语义变化，应用生成 `setting/progress.md`、`setting/character-states.md` 和 `setting/tracking.md`。不要分别手改，也不要新增另一套手写追踪。

## 调用

先 `prepare_advance`：

```json
{"path":"chapters/ch00001-opening.md","plan_path":"setting/chapter-plans/ch00001-opening.md"}
```

写前用于核对前章，最终改稿后再次调用以获取当前版本凭据。再 `commit_advance`（故事文本按作品语言填写，不照抄）：

```json
{
  "action":"commit",
  "path":"chapters/ch00001-opening.md",
  "plan_path":"setting/chapter-plans/ch00001-opening.md",
  "base_revision":"<latest preparation token>",
  "update":{
    "summary":"林晚拿到钥匙，决定独自走进无灯的楼梯间。",
    "context":{
      "language":"zh-CN",
      "position":"第一章结束，深夜，旧楼楼梯口。",
      "constraints":[],
      "active_characters":["林晚"],
      "active_threads":["F001"],
      "active_secrets":["S001"],
      "next_commitments":["接住楼梯间传来的脚步声。"],
      "risks":[]
    },
    "characters":[{"name":"林晚","location":"旧楼楼梯口","condition":"疲惫，未受伤","goal":"找到寄信人","possessions":["钥匙"],"relationships":[],"knowledge":["守夜人认得信封上的封印，原因未知。"]}],
    "threads":[{"id":"F001","label":"未署名的信","status":"active","detail":"封印被守夜人认出，寄信人尚未公开。"}],
    "secrets":[{"id":"S001","author_truth":"守夜人就是寄信人。","reader_knowledge":"守夜人认得封印。"}],
    "retired_context":[],
    "unresolved":[]
  }
}
```

## 更新规则

- `summary` 只记本章实际结果及后续影响，≤3072 UTF-8 字节。近三章由工具计算，不手填。
- 人物、伏笔、信息边界只传变化条目，但每条提交完整当前值；未传的保留。名称与 ID 保持稳定。
- `context` 是完整工作集：活跃人物 ≤20、伏笔 ≤8、信息边界 ≤12；约束、承诺与风险各 ≤6。退出活跃集合不会删除完整事实，需要时定点读取底账。
- 人物记录当前位置、状况、目标、物品、关系、所知/误判；稳定小传和规则留在 Lore。
- 伏笔只用 `active` / `resolved` / `abandoned`；回收有正文依据，弃用说明原因。未来安排留在规划。
- 作者真相和读者所知分开；未揭示时 `reader_knowledge` 为空，只揭示一部分就只写那部分。角色所知进入人物快照。
- 撤下旧约束、风险和承诺时，`retired_context` 写“原文：已解决/兑现/取消的原因”，不能只给原文。
- `context.language` 为 `zh-CN` 或 `en-US`，决定生成文档标题语言，随书稿保留。
- 台账 ≤12 KiB，单人物快照 ≤8 KiB；超限先压短冗余或明确退出工作集，不随意截断、遗忘未解决事实。底账 ≤16 MiB，生成进度/人物视图各 ≤2 MiB；超限先整理再重试。

## 已有书稿首次启用

1. 准备最新已写章；读实际前文、原进度、人物状态、台账与相关 Lore，只提取已确认事实。
2. 用准备凭据调用 `commit_advance action=initialize`，`path` 指向最新非空正文；提交当前完整人物、活跃伏笔、信息边界与工作集。空集合不编造条目。
3. 不要求给所有旧章补细纲，不重写正文。原进度作为历史基线保留；原人物/台账的有效事实纳入初始化语义，替换前原文由工作区变更历史保存，可整组撤销。
4. 工具登记已有正文版本；后续先修已改变的旧章，再新增。导入章事实修订须提交修正后的 `baseline_progress`，保留仍有效历史。

## 修订与恢复

- 改稿后重新准备并提交，同一章更新记录，不重复追加。
- 摘要对应改稿章，快照与工作集对应最新已写位置。后文与新事实冲突时在 `unresolved` 记录路径和原因，不自动改后文；解决前不新增章节。
- `view_conflicts` 表示生成文件被人工改动。先读取修改并纳入正确语义，不能机械用旧底账覆盖作者修正；重新提交仍保留前内容。
- 多文件写入复用工作区事务：先验证版本并持久化整组意图与前内容，再写入。中断后恢复同一事务，第三方改动产生冲突而不覆盖。Agent 恢复仍只有产品 journal。
- 成功响应丢失可重试相同请求；输入已变则重新准备。撤销/版本恢复还原底账与视图后，下一次准备重新核对正文版本。
