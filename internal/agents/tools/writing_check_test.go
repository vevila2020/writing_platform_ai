package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/config"

	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
)

func TestWritingCheckAuthorRangeOverridesPlanTarget(t *testing.T) {
	root := t.TempDir()
	writeWritingCheckFixture(t, root, "chapters/ch00001-night.md", strings.Repeat("夜", 2400)+"。")
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00001-night.md", "- 字数目标：2400 字\n")
	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "chapter", Path: "chapters/ch00001-night.md", WordMin: 3000, WordMax: 3500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !writingCheckFindingTypes(result)["word_count"] {
		t.Fatalf("author range was ignored: %#v", result)
	}
}

func TestWritingCheckPlanRangeOverridesTarget(t *testing.T) {
	result := writingCheckResult{}
	_, _, minChars, maxChars := checkChapterText("完整的一章。", "- 字数目标：100 字\n- 字数范围：5-10\n", checkWritingInput{}, &result)
	if minChars != 5 || maxChars != 10 || writingCheckFindingTypes(result)["word_count_severe"] {
		t.Fatalf("explicit plan range was ignored: %#v", result)
	}
}

func TestWritingCheckEnglishPlanAndUnfilledFields(t *testing.T) {
	valid := writingCheckResult{}
	checkChapterPlan(advanceTestPlan, &valid)
	finalizeWritingCheck(&valid)
	if valid.Blocking != 0 {
		t.Fatalf("English plan rejected: %#v", valid)
	}
	for _, placeholder := range []string{"TBD", "none", "[待补充]", "{actual event}"} {
		result := writingCheckResult{}
		checkChapterPlan(strings.Replace(advanceTestPlan, "A traveller enters the sealed stairway.", placeholder, 1), &result)
		if !writingCheckFindingTypes(result)["plan_unfilled_field"] {
			t.Fatalf("accepted placeholder %q", placeholder)
		}
	}
}

func TestDiscoverChapterPlanHandlesNestedOrdersAndAmbiguity(t *testing.T) {
	for _, tc := range []struct {
		name, path      string
		plans, chapters []string
		want            string
	}{
		{"exact order", "chapters/ch00001-night.md", []string{"ch000010-other.md", "ch00001-night.md"}, nil, "ch00001-night.md"},
		{"duplicate", "chapters/ch00001-night.md", []string{"ch00001-night.md", "ch00001-door.md"}, nil, ""},
		{"volume qualifier", "chapters/v00002-book/ch00001-night.md", []string{"ch00001-night.md", "v00002-ch00001-night.md"}, nil, "v00002-ch00001-night.md"},
		{"section qualifier", "chapters/v00002-book/ch00001-night/s00001-door.md", []string{"v00002-ch00001-s00001-door.md"}, nil, "v00002-ch00001-s00001-door.md"},
		{"ambiguous legacy", "chapters/v00002-book/ch00001-night.md", []string{"ch00001-night.md"}, []string{"chapters/v00001-book/ch00001-night.md"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range tc.plans {
				writeWritingCheckFixture(t, root, "setting/chapter-plans/"+name, advanceTestPlan)
			}
			for _, name := range tc.chapters {
				writeWritingCheckFixture(t, root, name, "Existing chapter.")
			}
			want := tc.want
			if want != "" {
				want = "setting/chapter-plans/" + want
			}
			if got := discoverChapterPlanPath(root, tc.path); got != want {
				t.Fatalf("discovered %q, want %q", got, want)
			}
		})
	}
}

func writeWritingCheckFixture(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", relativePath, err)
	}
}

func writingCheckFindingTypes(result writingCheckResult) map[string]bool {
	types := make(map[string]bool, len(result.Findings))
	for _, finding := range result.Findings {
		types[finding.Type] = true
	}
	return types
}

func TestCheckWritingToolDefinitionIsReadOnlyAndCapabilityScoped(t *testing.T) {
	definition, err := newWritingCheckTool(t.TempDir())
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}
	info, err := definition.Tool.Info(context.Background())
	if err != nil {
		t.Fatalf("read tool info: %v", err)
	}
	if info == nil || info.Name != "check_writing" {
		t.Fatalf("tool info = %#v, want check_writing", info)
	}
	if definition.Descriptor.Capability != config.AgentToolWritingCheck {
		t.Fatalf("capability = %q, want %q", definition.Descriptor.Capability, config.AgentToolWritingCheck)
	}
	if definition.Descriptor.MutationScope != agenttool.ToolMutationNone {
		t.Fatalf("mutation scope = %q, want none", definition.Descriptor.MutationScope)
	}
}

func TestWritingCheckCapabilityDefaultsOnForIDEOnly(t *testing.T) {
	ide := config.ResolveAgentTools(&config.Config{}, config.AgentKindIDE)
	if !ide.Allows(config.AgentToolWritingCheck) {
		t.Fatal("writing_check should be enabled for the IDE agent by default")
	}
	general := config.ResolveAgentTools(&config.Config{}, config.AgentKindGeneral)
	if general.Allows(config.AgentToolWritingCheck) {
		t.Fatal("writing_check should not be enabled for the General agent")
	}
}

func TestIDEToolsExposeWritingCheckOnlyWhenEnabled(t *testing.T) {
	cfg := &config.Config{Workspace: t.TempDir()}
	factory := ideToolsFactory(cfg)

	disabled, err := factory(config.ResolvedAgentToolSettings{})
	if err != nil {
		t.Fatalf("build disabled tools: %v", err)
	}
	for _, definition := range disabled {
		info, err := definition.Tool.Info(context.Background())
		if err != nil {
			t.Fatalf("read disabled tool info: %v", err)
		}
		if info != nil && info.Name == "check_writing" {
			t.Fatal("check_writing was assembled while its capability was disabled")
		}
	}

	enabled, err := factory(config.ResolvedAgentToolSettings{config.AgentToolWritingCheck: true})
	if err != nil {
		t.Fatalf("build enabled tools: %v", err)
	}
	for _, definition := range enabled {
		info, err := definition.Tool.Info(context.Background())
		if err != nil {
			t.Fatalf("read enabled tool info: %v", err)
		}
		if info != nil && info.Name == "check_writing" {
			return
		}
	}
	t.Fatal("check_writing was not assembled while its capability was enabled")
}

func TestRunWritingCheckChapterFindsDeterministicHardFailures(t *testing.T) {
	root := t.TempDir()
	plan := "## 细纲（第 1 章）\n\n### 第 1 章：夜行\n\n" +
		"- 核心事件：林晚深夜到值班室找守夜人，拿到楼梯间的钥匙。\n" +
		"- 字数目标：2000 字\n" +
		"- 目标情绪：从警惕走到克制的决心。\n" +
		"- 主角目标 / 关键选择：拿到钥匙，但不能让守夜人起疑。\n" +
		"- 章尾钩子：楼梯上没有灯。\n"
	repeated := "他走进房间，看见桌上放着一封信。\n"
	prose := "林晚深夜到值班室找守夜人，拿到楼梯间的钥匙。\n" + repeated + repeated + repeated +
		"细纲里写着下一步要下楼。\nTODO\n她走出门，回头看了一眼"
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00001-night.md", plan)
	writeWritingCheckFixture(t, root, "chapters/ch00001-night.md", prose)

	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "chapter", Path: "chapters/ch00001-night.md",
	})
	if err != nil {
		t.Fatalf("check chapter: %v", err)
	}
	types := writingCheckFindingTypes(result)
	for _, want := range []string{"truncation", "placeholder", "repetition", "engineering_word", "outline_copy", "word_count_severe"} {
		if !types[want] {
			t.Fatalf("missing finding %q; got %#v\n%s", want, types, result.Summary)
		}
	}
	if result.Blocking == 0 || result.OK {
		t.Fatalf("expected blocking findings, got %#v", result)
	}
}

func TestRunWritingCheckCleanChapterHasNoBlockingFindings(t *testing.T) {
	root := t.TempDir()
	plan := "## 细纲（第 1 章）\n\n- 核心事件：林晚到值班室拿钥匙。\n- 字数目标：150 字\n- 目标情绪：从警惕走到克制的决心。\n- 主角目标 / 关键选择：拿到钥匙，但不能让守夜人起疑。\n- 章尾钩子：楼梯上没有灯。\n"
	prose := "林晚推开门，风从走廊尽头灌进来。她把信按在胸口，脚步没有停。\n\n" +
		"值班室亮着一盏灯。她认得桌上的那只铜杯，也认得杯底那道缺口。\n\n" +
		"“这么晚？”守夜人抬头。\n\n" +
		"“来找人。”她把信放在桌上，没有松手。\n\n" +
		"守夜人看了她一眼，把钥匙推到桌边。她拿起钥匙，转身走向楼梯。楼梯上没有灯，只有窗外一点微光。\n"
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00001-night.md", plan)
	writeWritingCheckFixture(t, root, "chapters/ch00001-night.md", prose)

	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "chapter", Path: "chapters/ch00001-night.md",
	})
	if err != nil {
		t.Fatalf("check chapter: %v", err)
	}
	if result.Blocking != 0 || !result.OK {
		t.Fatalf("clean chapter should have no blocking findings: %#v", result.Findings)
	}
	if result.WordTarget != 150 {
		t.Fatalf("word target = %d, want parsed plan target 150", result.WordTarget)
	}
}

func TestRunWritingCheckPlanRequiresCoreFields(t *testing.T) {
	root := t.TempDir()
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00003-empty.md", "# 细纲\n\n- 字数目标：1200 字\n")

	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "plan", Path: "setting/chapter-plans/ch00003-empty.md",
	})
	if err != nil {
		t.Fatalf("check plan: %v", err)
	}
	types := writingCheckFindingTypes(result)
	for _, want := range []string{"plan_missing_field"} {
		if !types[want] {
			t.Fatalf("missing finding %q; got %#v", want, types)
		}
	}
	if result.Blocking < 4 {
		t.Fatalf("expected four missing core fields, got %d blocking", result.Blocking)
	}

	valid := "## 细纲（第 1 章）\n\n- 核心事件：林晚到值班室拿钥匙。\n- 字数目标：150 字\n- 目标情绪：从警惕走到克制的决心。\n- 主角目标 / 关键选择：拿到钥匙，但不能让守夜人起疑。\n- 章尾钩子：楼梯上没有灯。\n"
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00004-valid.md", valid)
	validResult, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "plan", Path: "setting/chapter-plans/ch00004-valid.md",
	})
	if err != nil {
		t.Fatalf("check valid plan: %v", err)
	}
	if validResult.Blocking != 0 {
		t.Fatalf("valid plan should have no blocking findings: %#v", validResult.Findings)
	}
}

func TestRunWritingCheckTrackingRequiresSectionsAndUniqueForeshadowIDs(t *testing.T) {
	root := t.TempDir()
	ledger := "# 连续性台账\n\n## 当前位置\n\n- 当前章：第 1 章\n\n" +
		"## 长期约束\n\n- 不提前揭开信的内容。\n\n" +
		"## 活跃伏笔\n\n- F001｜那封信的寄信人｜埋设第 1 章｜预计回收第 3 章｜重要度高｜状态已埋\n- F001｜重复编号｜埋设第 2 章｜预计回收第 4 章｜重要度中｜状态已埋\n\n" +
		"## 作者真相 / 读者已知\n\n### 作者真相（尚未揭示）\n\n- 信是守夜人旧友寄的。\n\n### 读者已知\n\n- 林晚拿到钥匙。\n\n" +
		"## 下一章承诺\n\n- 林晚打开楼梯间的门。\n\n## 连贯性风险\n\n- 不能提前说出寄信人。\n\n## 近三章速记\n\n- 第 1 章：林晚到值班室拿钥匙。\n"
	writeWritingCheckFixture(t, root, "setting/tracking.md", ledger)

	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "tracking", Path: "setting/tracking.md",
	})
	if err != nil {
		t.Fatalf("check tracking: %v", err)
	}
	if !writingCheckFindingTypes(result)["tracking_duplicate_foreshadow"] {
		t.Fatalf("missing duplicate-foreshadow finding: %#v", result.Findings)
	}

	writeWritingCheckFixture(t, root, "setting/tracking.md", "# 连续性台账\n\n## 当前位置\n\n- 当前章：第 1 章\n")
	missing, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "tracking", Path: "setting/tracking.md",
	})
	if err != nil {
		t.Fatalf("check incomplete tracking: %v", err)
	}
	if missing.Blocking < 7 {
		t.Fatalf("expected missing-section blocking findings, got %d", missing.Blocking)
	}
}

func TestRunWritingCheckValidTrackingLedgerHasNoBlocking(t *testing.T) {
	root := t.TempDir()
	ledger := "# 连续性台账\n\n## 当前位置\n\n- 当前章：第 1 章\n- 故事时间：夜里\n- 场景：值班室\n\n" +
		"## 长期约束\n\n- 不提前揭开信的内容。\n\n" +
		"## 活跃伏笔\n\n- F001｜那封信的寄信人｜埋设第 1 章｜预计回收第 3 章｜重要度高｜状态已埋\n\n" +
		"## 作者真相 / 读者已知\n\n### 作者真相（尚未揭示）\n\n- 信是守夜人旧友寄的。\n\n### 读者已知\n\n- 林晚拿到钥匙。\n\n" +
		"## 下一章承诺\n\n- 林晚打开楼梯间的门。\n\n## 连贯性风险\n\n- 不能提前说出寄信人。\n\n## 近三章速记\n\n- 第 1 章：林晚到值班室拿钥匙。\n"
	writeWritingCheckFixture(t, root, "setting/tracking.md", ledger)

	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "tracking", Path: "setting/tracking.md",
	})
	if err != nil {
		t.Fatalf("check tracking: %v", err)
	}
	if result.Blocking != 0 || !result.OK {
		t.Fatalf("valid ledger should have no blocking findings: %#v", result.Findings)
	}
}

func TestRunWritingCheckDiscoversChapterPlanByOrderPrefix(t *testing.T) {
	root := t.TempDir()
	plan := "## 细纲（第 2 章）\n\n- 核心事件：林晚在楼梯间遇到一个提灯的人，对方知道她手里的信是谁寄的。\n- 字数目标：800 字\n- 目标情绪：从警惕走到克制的决心。\n- 主角目标 / 关键选择：问出寄信人，但不能暴露自己害怕。\n- 章尾钩子：提灯的人叫出了她的旧姓。\n"
	prose := "林晚在楼梯间遇到一个提灯的人，对方知道她手里的信是谁寄的。\n\n她停下脚步，没有后退。\n"
	writeWritingCheckFixture(t, root, "setting/chapter-plans/ch00002-stairs.md", plan)
	writeWritingCheckFixture(t, root, "chapters/ch00002-stairs.md", prose)

	result, err := runWritingCheck(context.Background(), root, checkWritingInput{
		Action: "chapter", Path: "chapters/ch00002-stairs.md",
	})
	if err != nil {
		t.Fatalf("check chapter: %v", err)
	}
	if result.WordTarget != 800 {
		t.Fatalf("word target = %d, want discovered plan target 800", result.WordTarget)
	}
	if !writingCheckFindingTypes(result)["outline_copy"] {
		t.Fatalf("missing outline-copy finding: %#v", result.Findings)
	}
}
