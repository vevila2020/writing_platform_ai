package advance

type documentText struct {
	Progress, Characters, Tracking, Imported, Position, Constraints, Threads, AuthorTruth, ReaderKnowledge, Commitments, Risks, Recent, None string
	Location, Condition, Goal, Possessions, Relationships, Knowledge, Resolved, Abandoned, Conflicts                                         string
}

// Generated document labels are independent from story content. Their language
// is stored with the book's working context and does not change with UI theme.
var textEN = documentText{
	Progress: "Writing progress", Characters: "Character states", Tracking: "Continuity ledger", Imported: "Earlier progress (imported)", Position: "Current position", Constraints: "Long-term constraints", Threads: "Active foreshadowing", AuthorTruth: "Author truth", ReaderKnowledge: "Reader knowledge", Commitments: "Next-chapter commitments", Risks: "Continuity risks", Recent: "Last three chapters", None: "None.",
	Location: "Location", Condition: "Condition", Goal: "Current goal", Possessions: "Possessions", Relationships: "Relationships", Knowledge: "Knowledge and beliefs", Resolved: "Resolved", Abandoned: "Abandoned", Conflicts: "Unresolved prose conflicts",
}
var textZH = documentText{
	Progress: "写作进度", Characters: "人物当前状态", Tracking: "连续性台账", Imported: "此前进度（导入保留）", Position: "当前位置", Constraints: "长期约束", Threads: "活跃伏笔", AuthorTruth: "作者真相", ReaderKnowledge: "读者已知", Commitments: "下一章承诺", Risks: "连贯性风险", Recent: "近三章速记", None: "无。",
	Location: "所在地", Condition: "当前状况", Goal: "短期目标", Possessions: "持有物", Relationships: "关系变化", Knowledge: "已知与误判", Resolved: "已回收", Abandoned: "已弃用", Conflicts: "后文待修正冲突",
}
