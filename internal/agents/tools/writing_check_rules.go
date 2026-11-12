package tools

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/vevila2020/writing_platform_ai/internal/book"
)

var (
	planTargetPattern = regexp.MustCompile(`(?mi)^[ \t>]*(?:[-*+][ \t]*)?(?:\*\*)?(?:字数目标|word target)(?:\*\*)?[ \t]*[:：][ \t]*(?:约|大约)?[ \t]*([0-9][0-9,，]*)[ \t]*字?`)
	planRangePattern  = regexp.MustCompile(`(?mi)^[ \t>]*(?:[-*+][ \t]*)?(?:\*\*)?(?:字数范围|word range)(?:\*\*)?[ \t]*[:：][ \t]*(?:约|大约)?[ \t]*([0-9][0-9,，]*)[ \t]*字?[ \t]*(?:-|~|～|—|–|－|至|到)[ \t]*([0-9][0-9,，]*)[ \t]*字?`)

	quotedSpanPattern  = regexp.MustCompile(`「[^」]*」|『[^』]*』|“[^”]*”|"[^"]*"`)
	fieldPrefixPattern = regexp.MustCompile(`^[^：:\n]{1,20}[：:]\s*`)

	foreshadowIDPattern = regexp.MustCompile(`F\d{3,}`)

	emDashPattern         = regexp.MustCompile(`——`)
	notIsPattern          = regexp.MustCompile(`不是[^。！？!?\n]{1,40}(?:而是|，是)`)
	voiceContrastPattern  = regexp.MustCompile(`声音(?:不大|不高|不响)[，,]?[^。！？!?\n]{0,20}却`)
	negationParadePattern = regexp.MustCompile(`(?:没有|没|不)[^。！？!?\n]{1,12}[，,；;][\t ]*(?:没有|没|不)`)
	clichePattern         = regexp.MustCompile(`缓缓|淡淡|微微|轻轻|默默|静静|不禁|下意识`)
	stockReactionPattern  = regexp.MustCompile(`瞳孔[^。！？!?\n]{0,6}(?:缩|骤缩)|倒吸一口凉气|全场寂静|鸦雀无声|心中一凛|指节泛白|喉咙发紧`)
	metaphorPattern       = regexp.MustCompile(`像|好像|仿佛|如同|犹如|宛若`)
	reasoningChainPattern = regexp.MustCompile(`这意味着|也就是说|换句话说|他知道，这代表|这说明`)
	abstractEndingPattern = regexp.MustCompile(`这一刻[^。！？!?\n]{0,24}(?:终于|才)(?:明白|意识到)|才刚刚开始|命运的齿轮|从这一刻开始`)
)

var (
	blockingPlaceholderPatterns = []struct {
		pattern *regexp.Regexp
		kind    string
		message string
	}{
		{regexp.MustCompile(`(?i)TODO|placeholder|未完待续`), "placeholder", "Placeholder or unfinished-text marker found in prose; replace it with completed prose."},
		{regexp.MustCompile(`此处省略|以下省略|[（(]\s*省略\s*[）)]`), "omitted_content", "Omitted-content marker found in prose; write the missing scene instead of leaving a marker."},
		{regexp.MustCompile(`\[待补充\]|【待补充】`), "unfilled_placeholder", "Unfilled placeholder found in prose; complete the marked content."},
		{regexp.MustCompile(`�`), "replacement_character", "Replacement character found; repair the encoding or rewrite the affected text."},
	}
	advisoryPlaceholderPatterns = []struct {
		pattern *regexp.Regexp
		kind    string
		message string
	}{
		{regexp.MustCompile(`(?i)作为\s*(?:一个)?\s*(?:AI|人工智能|大?语言模型|智能助手)`), "ai_self_reference", "AI self-reference found; remove meta text that is not part of the story."},
		{regexp.MustCompile(`我无法(?:继续|生成|创作|续写)`), "refusal_text", "Possible refusal text found; verify it is dialogue and not a model-facing refusal."},
		{regexp.MustCompile(`(?i)I (?:cannot|can't|am unable to) (?:continue|write|generate)`), "refusal_text", "Possible refusal text found; verify it is dialogue and not a model-facing refusal."},
		{regexp.MustCompile(`(?i)As an AI`), "ai_self_reference", "AI self-reference found; remove meta text that is not part of the story."},
	}
)

var (
	engineeringTier1 = []string{"细纲", "情节点", "卷纲", "章首钩子", "章尾钩子", "字数目标", "目标情绪", "情节细化", "排纲"}
	engineeringTier2 = []string{"本章", "下一章", "上一章", "前文", "后文", "伏笔", "读者", "任务描述"}
)

func checkChapterText(content, plan string, input checkWritingInput, result *writingCheckResult) (wordCount, target, minChars, maxChars int) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		addWritingFinding(result, "blocking", "empty", 1, 1, "", "The chapter file is empty.")
		return 0, input.WordTarget, input.WordMin, input.WordMax
	}
	starts := writingLineStarts(content)
	checkTruncation(content, starts, result)
	checkPlaceholders(content, result)
	checkRepetition(content, result)
	checkEngineeringWords(content, result)
	if plan != "" {
		checkOutlineCopy(content, plan, starts, result)
	}
	checkAIPatterns(content, starts, result)

	target, minChars, maxChars = input.WordTarget, input.WordMin, input.WordMax
	if plan != "" {
		planTarget, planMin, planMax := parsePlanWordBudget(plan)
		if target == 0 {
			target = planTarget
		}
		if minChars == 0 && maxChars == 0 && input.WordTarget == 0 {
			minChars, maxChars = planMin, planMax
		}
	}
	wordCount = book.CountWritingWords(content)
	checkWordCount(result, wordCount, target, minChars, maxChars)
	return wordCount, target, minChars, maxChars
}

func checkChapterPlan(content string, result *writingCheckResult) {
	required := []struct {
		name   string
		labels []string
	}{
		{name: "core event", labels: []string{"核心事件", "Core event"}},
		{name: "target emotion", labels: []string{"目标情绪", "Target emotion"}},
		{name: "protagonist goal / key choice", labels: []string{"主角目标 / 关键选择", "主角目标/关键选择", "主角目标 /关键选择", "主角目标/ 关键选择", "主角目标", "关键选择", "Protagonist goal / key choice", "Protagonist goal", "Key choice"}},
		{name: "chapter-end hook", labels: []string{"章尾钩子", "Chapter-end hook"}},
	}
	for _, field := range required {
		if value, ok := planFieldValue(content, field.labels...); !ok {
			addWritingFinding(result, "blocking", "plan_missing_field", 0, 0, "", fmt.Sprintf("Required plan field %q is missing or empty.", field.name))
		} else if placeholder := strings.ToLower(strings.TrimSpace(value)); placeholder == "无" || placeholder == "待定" || placeholder == "待补充" || placeholder == "none" || placeholder == "tbd" || placeholder == "todo" || placeholder == "n/a" || strings.Contains(value, "[待补充]") || (strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}")) {
			addWritingFinding(result, "blocking", "plan_unfilled_field", 0, 0, value, fmt.Sprintf("Plan field %q contains a placeholder; describe the actual story beat.", field.name))
		}
	}
	if value, ok := planFieldValue(content, "目标情绪"); ok && utf8.RuneCountInString(value) < 8 {
		addWritingFinding(result, "advisory", "plan_target_emotion_weak", 0, 0, value, "Target emotion is very short; describe the emotional movement rather than a single label.")
	}
	target, minChars, maxChars := parsePlanWordBudget(content)
	if target == 0 && minChars == 0 && maxChars == 0 {
		kind := "plan_word_budget_missing"
		message := "No valid 字数目标 or 字数范围 was found; add a word target before drafting."
		if strings.Contains(content, "字数目标") || strings.Contains(content, "字数范围") {
			kind = "plan_word_budget_invalid"
			message = "The 字数目标 or 字数范围 line could not be parsed; use a value such as 字数目标：2200 字 or 字数范围：2000-2600."
		}
		addWritingFinding(result, "blocking", kind, 0, 0, "", message)
	}
}

func checkTrackingLedger(content string, result *writingCheckResult) {
	size := len([]byte(content))
	if size > 20<<10 {
		addWritingFinding(result, "blocking", "tracking_too_large", 0, 0, "", fmt.Sprintf("Continuity ledger is %d bytes; compact it below 12KB.", size))
	} else if size > 12<<10 {
		addWritingFinding(result, "advisory", "tracking_large", 0, 0, "", fmt.Sprintf("Continuity ledger is %d bytes; compact it toward 12KB or less.", size))
	}

	sections := parseLedgerSections(content)
	required := []string{"当前位置", "长期约束", "活跃伏笔", "作者真相", "读者已知", "下一章承诺", "连贯性风险", "近三章速记"}
	for _, name := range required {
		if _, ok := findLedgerSection(sections, name); !ok {
			addWritingFinding(result, "blocking", "tracking_missing_section", 0, 0, "", fmt.Sprintf("Required ledger section %q is missing.", name))
		}
	}

	if section, ok := findLedgerSection(sections, "长期约束"); ok && countLedgerBullets(section.lines) > 6 {
		addWritingFinding(result, "advisory", "tracking_too_many_constraints", 0, 0, "", "More than 6 long-term constraints; merge or retire the least important ones.")
	}

	if section, ok := findLedgerSection(sections, "活跃伏笔"); ok {
		ids := make(map[string]bool)
		foreshadows := 0
		for _, line := range section.lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || !strings.HasPrefix(trimmed, "-") {
				continue
			}
			if value := strings.TrimSpace(strings.TrimPrefix(trimmed, "-")); value == "无" || value == "无。" || strings.EqualFold(value, "None.") {
				continue
			}
			foreshadows++
			id := foreshadowIDPattern.FindString(trimmed)
			if id == "" {
				addWritingFinding(result, "advisory", "tracking_foreshadow_format", 0, 0, trimmed, "Active foreshadowing entry has no F-number; add a stable F-id.")
				continue
			}
			if ids[id] {
				addWritingFinding(result, "blocking", "tracking_duplicate_foreshadow", 0, 0, trimmed, fmt.Sprintf("Foreshadowing id %s appears more than once in the active list.", id))
			}
			ids[id] = true
		}
		if foreshadows > 8 {
			addWritingFinding(result, "advisory", "tracking_too_many_foreshadows", 0, 0, "", "More than 8 active foreshadowing entries; resolve or merge the least important ones.")
		}
	}

	if section, ok := findLedgerSection(sections, "下一章承诺"); ok && countLedgerBullets(section.lines) == 0 {
		addWritingFinding(result, "advisory", "tracking_empty_commitment", 0, 0, "", "The next-chapter commitment section is empty; record what the next chapter must pick up.")
	}
}

func parsePlanWordBudget(plan string) (target, minChars, maxChars int) {
	if matches := planTargetPattern.FindStringSubmatch(plan); len(matches) == 2 {
		target = parsePlanNumber(matches[1])
	}
	if matches := planRangePattern.FindStringSubmatch(plan); len(matches) == 3 {
		minChars = parsePlanNumber(matches[1])
		maxChars = parsePlanNumber(matches[2])
		if minChars > maxChars {
			minChars, maxChars = maxChars, minChars
		}
	}
	return target, minChars, maxChars
}

func parsePlanNumber(raw string) int {
	raw = strings.NewReplacer(",", "", "，", "").Replace(raw)
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func planFieldValue(content string, labels ...string) (string, bool) {
	for _, label := range labels {
		pattern := regexp.MustCompile(`(?mi)^[ \t>]*(?:[-*+][ \t]*)?(?:\*\*)?` + regexp.QuoteMeta(label) + `(?:\*\*)?[ \t]*[:：][ \t]*(.*)$`)
		matches := pattern.FindStringSubmatch(content)
		if len(matches) != 2 {
			continue
		}
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func checkWordCount(result *writingCheckResult, count, target, minChars, maxChars int) {
	switch {
	case minChars > 0 || maxChars > 0:
		if (minChars > 0 && count < minChars/2) || (maxChars > 0 && count > maxChars*2) {
			addWritingFinding(result, "blocking", "word_count_severe", 0, 0, "", fmt.Sprintf("Visible characters: %d; requested range: %d-%d. The chapter is far outside the range.", count, minChars, maxChars))
		} else if (minChars > 0 && count < minChars) || (maxChars > 0 && count > maxChars) {
			addWritingFinding(result, "advisory", "word_count", 0, 0, "", fmt.Sprintf("Visible characters: %d; requested range: %d-%d.", count, minChars, maxChars))
		}
	case target > 0:
		lower := target * 85 / 100
		upper := target * 115 / 100
		if count < target/2 || count > target*2 {
			addWritingFinding(result, "blocking", "word_count_severe", 0, 0, "", fmt.Sprintf("Visible characters: %d; target: %d. The chapter is outside half-to-double the target.", count, target))
		} else if count < lower || count > upper {
			addWritingFinding(result, "advisory", "word_count", 0, 0, "", fmt.Sprintf("Visible characters: %d; target: %d (suggested %d-%d).", count, target, lower, upper))
		}
	}
}

func checkTruncation(content string, starts []int, result *writingCheckResult) {
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if line == "---" || line == "***" || line == "＊" {
			return
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "-") {
			return
		}
		runes := []rune(line)
		last := runes[len(runes)-1]
		if isClosingWritingRune(last) && isDialogueLine(line) {
			return
		}
		for len(runes) > 0 && isClosingWritingRune(runes[len(runes)-1]) {
			runes = runes[:len(runes)-1]
		}
		if len(runes) == 0 {
			return
		}
		if isTerminalWritingRune(runes[len(runes)-1]) {
			return
		}
		lineNumber := i + 1
		if lineNumber > len(starts) {
			lineNumber = len(starts)
		}
		addWritingFinding(result, "blocking", "truncation", lineNumber, 1, line, "The chapter ends without terminal punctuation; complete or rewrite the ending.")
		return
	}
}

func isClosingWritingRune(r rune) bool {
	switch r {
	case '」', '』', '”', '’', '）', ')', '》', '】', ']', '}', '"', '\'':
		return true
	}
	return false
}

func isTerminalWritingRune(r rune) bool {
	switch r {
	case '。', '！', '？', '!', '?', '…', '～', '~', '.':
		return true
	}
	return false
}

func checkPlaceholders(content string, result *writingCheckResult) {
	for index, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, item := range blockingPlaceholderPatterns {
			if match := item.pattern.FindStringIndex(line); match != nil {
				addWritingFinding(result, "blocking", item.kind, index+1, writingColumn(line, match[0]), line, item.message)
			}
		}
		for _, item := range advisoryPlaceholderPatterns {
			if match := item.pattern.FindStringIndex(line); match != nil {
				addWritingFinding(result, "advisory", item.kind, index+1, writingColumn(line, match[0]), line, item.message)
			}
		}
	}
}

func checkRepetition(content string, result *writingCheckResult) {
	lines := strings.Split(content, "\n")
	reported := make(map[string]bool)
	previous := ""
	adjacent := 0

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || isDialogueLine(trimmed) {
			previous = ""
			continue
		}
		if utf8.RuneCountInString(trimmed) >= 8 && trimmed == previous {
			addWritingFinding(result, "blocking", "repetition", index+1, 1, trimmed, "Adjacent identical narration lines repeat; keep one functional occurrence.")
			if adjacent++; adjacent >= 3 {
				return
			}
		}
		previous = trimmed
	}

	counts := make(map[string]int)
	firstLines := make(map[string]int)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || isDialogueLine(trimmed) {
			continue
		}
		for _, sentence := range splitNarrativeSentences(trimmed) {
			if utf8.RuneCountInString(sentence) < 12 {
				continue
			}
			counts[sentence]++
			if firstLines[sentence] == 0 {
				firstLines[sentence] = index + 1
			}
		}
		if utf8.RuneCountInString(trimmed) >= 12 {
			counts[trimmed]++
			if firstLines[trimmed] == 0 {
				firstLines[trimmed] = index + 1
			}
		}
	}
	for text, count := range counts {
		if count < 3 || reported[text] {
			continue
		}
		reported[text] = true
		addWritingFinding(result, "blocking", "repetition", firstLines[text], 1, text, "The same narration span repeats three or more times; keep one functional occurrence or vary the wording.")
		if len(reported) >= 3 {
			return
		}
	}
}

func splitNarrativeSentences(line string) []string {
	line = quotedSpanPattern.ReplaceAllString(line, "")
	parts := regexp.MustCompile(`[。！？!?]+`).Split(line, -1)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func isDialogueLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, prefix := range []string{"「", "『", "“", "\"", "'"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	if quotedSpanPattern.MatchString(trimmed) {
		return strings.TrimSpace(quotedSpanPattern.ReplaceAllString(trimmed, "")) == ""
	}
	return false
}

func checkEngineeringWords(content string, result *writingCheckResult) {
	reported := 0
	for index, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, word := range engineeringTier1 {
			if position := strings.Index(line, word); position >= 0 {
				addWritingFinding(result, "blocking", "engineering_word", index+1, writingColumn(line, position), line, fmt.Sprintf("Engineering word %q leaked into prose; replace it with an in-story event, action, or time anchor.", word))
				if reported++; reported >= 10 {
					return
				}
			}
		}
		for _, word := range engineeringTier2 {
			if position := strings.Index(line, word); position >= 0 {
				addWritingFinding(result, "advisory", "engineering_word", index+1, writingColumn(line, position), line, fmt.Sprintf("Possible engineering word %q in prose; verify it is an in-story term, not workflow leakage.", word))
				if reported++; reported >= 10 {
					return
				}
			}
		}
	}
}

func checkOutlineCopy(content, plan string, starts []int, result *writingCheckResult) {
	seen := make(map[string]bool)
	reported := 0
	for _, sentence := range planSentences(plan) {
		if seen[sentence] || utf8.RuneCountInString(sentence) < 15 {
			continue
		}
		seen[sentence] = true
		index := strings.Index(content, sentence)
		if index < 0 {
			continue
		}
		severity := "advisory"
		if utf8.RuneCountInString(sentence) >= 30 {
			severity = "blocking"
		}
		line, column := writingLineColumn(content, starts, index)
		addWritingFinding(result, severity, "outline_copy", line, column, sentence, "The chapter copies a long span verbatim from the chapter plan; rewrite it as scene action, dialogue, or consequence.")
		if reported++; reported >= 5 {
			return
		}
	}
}

func planSentences(plan string) []string {
	lines := strings.Split(plan, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		value := strings.TrimSpace(line)
		value = strings.TrimLeft(value, "#>*-+ \t")
		value = strings.ReplaceAll(value, "|", " ")
		value = strings.ReplaceAll(value, "**", "")
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = fieldPrefixPattern.ReplaceAllString(value, "")
		for _, sentence := range regexp.MustCompile(`[。！？!?]+`).Split(value, -1) {
			sentence = strings.TrimSpace(sentence)
			if utf8.RuneCountInString(sentence) >= 15 {
				result = append(result, sentence)
			}
		}
	}
	return result
}

func checkAIPatterns(content string, starts []int, result *writingCheckResult) {
	addDensityFinding(result, content, starts, notIsPattern, 2, "not_is_comparison", "Repeated 不是……而是…… contrast; state the positive term directly or show it through action.")
	addDensityFinding(result, content, starts, voiceContrastPattern, 1, "voice_contrast", "Voice-contrast pattern (声音不大……却……); rewrite the contrast through a visible reaction.")
	addDensityFinding(result, content, starts, negationParadePattern, 3, "negation_parade", "Repeated negation pairs (没有……也没有……); use direct statements or concrete action.")
	addDensityFinding(result, content, starts, emDashPattern, 3, "em_dash", "Many em-dashes; replace most with commas, periods, or a rewritten sentence.")
	addDensityFinding(result, content, starts, clichePattern, 8, "cliche_density", "High density of stock adverbs; replace some with specific actions or sensory details.")
	addDensityFinding(result, content, starts, stockReactionPattern, 4, "stock_reaction", "Repeated stock reactions; give characters specific, differentiated actions.")
	addDensityFinding(result, content, starts, metaphorPattern, 10, "metaphor_density", "High metaphor-marker density; keep the functional comparisons and cut the rest.")
	addDensityFinding(result, content, starts, reasoningChainPattern, 3, "reasoning_chain", "Repeated explanation chains; trust the scene and remove narrator explanation.")
	checkAbstractEnding(content, starts, result)
	checkParagraphUniformity(content, result)
}

func addDensityFinding(result *writingCheckResult, content string, starts []int, pattern *regexp.Regexp, threshold int, kind, message string) {
	matches := pattern.FindAllStringIndex(content, -1)
	if len(matches) < threshold {
		return
	}
	line, column := writingLineColumn(content, starts, matches[0][0])
	addWritingFinding(result, "advisory", kind, line, column, writingLineAt(content, line), message)
}

func checkAbstractEnding(content string, starts []int, result *writingCheckResult) {
	runes := []rune(content)
	start := 0
	if len(runes) > 600 {
		start = len(runes) - 600
	}
	tail := string(runes[start:])
	match := abstractEndingPattern.FindStringIndex(tail)
	if match == nil {
		return
	}
	offset := len(string(runes[:start])) + match[0]
	line, column := writingLineColumn(content, starts, offset)
	addWritingFinding(result, "advisory", "abstract_ending", line, column, writingLineAt(content, line), "The chapter may end on an abstract summary or trailer line; land on a concrete action, choice, or consequence instead.")
}

func checkParagraphUniformity(content string, result *writingCheckResult) {
	paragraphs := strings.Split(content, "\n\n")
	lengths := make([]int, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		value := strings.TrimSpace(paragraph)
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		lengths = append(lengths, utf8.RuneCountInString(value))
	}
	if len(lengths) < 8 {
		return
	}
	total := 0
	for _, length := range lengths {
		total += length
	}
	mean := float64(total) / float64(len(lengths))
	if mean <= 0 {
		return
	}
	variance := 0.0
	for _, length := range lengths {
		delta := float64(length) - mean
		variance += delta * delta
	}
	stddev := variance / float64(len(lengths))
	if stddev < mean*mean*0.0144 {
		addWritingFinding(result, "advisory", "paragraph_uniformity", 1, 1, "", "Paragraph lengths are unusually uniform; vary scene length, sentence rhythm, and information density.")
	}
}

func parseLedgerSections(content string) []ledgerSection {
	lines := strings.Split(content, "\n")
	sections := make([]ledgerSection, 0, 8)
	current := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			name := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			sections = append(sections, ledgerSection{name: name})
			current = len(sections) - 1
			continue
		}
		if current >= 0 {
			sections[current].lines = append(sections[current].lines, line)
		}
	}
	return sections
}

type ledgerSection struct {
	name  string
	lines []string
}

func findLedgerSection(sections []ledgerSection, name string) (ledgerSection, bool) {
	english := map[string]string{"当前位置": "Current position", "长期约束": "Long-term constraints", "活跃伏笔": "Active foreshadowing", "作者真相": "Author truth", "读者已知": "Reader knowledge", "下一章承诺": "Next-chapter commitments", "连贯性风险": "Continuity risks", "近三章速记": "Last three chapters"}
	for _, section := range sections {
		if strings.Contains(section.name, name) || (english[name] != "" && strings.EqualFold(section.name, english[name])) {
			return section, true
		}
	}
	return ledgerSection{}, false
}

func countLedgerBullets(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			count++
		}
	}
	return count
}

func writingLineStarts(content string) []int {
	starts := []int{0}
	for index := 0; index < len(content); index++ {
		if content[index] == '\n' {
			starts = append(starts, index+1)
		}
	}
	return starts
}

func writingLineColumn(content string, starts []int, offset int) (line, column int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(content) {
		offset = len(content)
	}
	index := sort.Search(len(starts), func(i int) bool { return starts[i] > offset }) - 1
	if index < 0 {
		index = 0
	}
	return index + 1, utf8.RuneCountInString(content[starts[index]:offset]) + 1
}

func writingColumn(line string, byteOffset int) int {
	if byteOffset <= 0 {
		return 1
	}
	if byteOffset > len(line) {
		byteOffset = len(line)
	}
	return utf8.RuneCountInString(line[:byteOffset]) + 1
}

func writingLineAt(content string, line int) string {
	if line <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if line > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line-1])
}
