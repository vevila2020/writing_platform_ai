package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/vevila2020/writing_platform_ai/config"
	"github.com/vevila2020/writing_platform_ai/internal/book"

	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
	agenttool "github.com/vevila2020/writing_platform_ai/agent/tool"
)

const (
	writingCheckSchema       = "inkdock-writing-check/v1"
	writingCheckMaxFileBytes = 2 << 20
	writingCheckMaxFindings  = 50
	writingCheckExcerptRunes = 120
)

type checkWritingInput struct {
	Action     string `json:"action" jsonschema:"enum=chapter,enum=plan,enum=tracking" jsonschema_description:"What to check. chapter checks prose; plan checks a per-chapter plan; tracking checks the continuity ledger."`
	Path       string `json:"path" jsonschema_description:"Workspace-relative path to the file to check."`
	PlanPath   string `json:"plan_path,omitempty" jsonschema_description:"Optional workspace-relative chapter plan path. Used to detect large verbatim outline copying and to read the word budget when word_target/word_min/word_max are omitted. When omitted for a chapter, the tool looks for a single setting/chapter-plans file with the same ch order prefix."`
	WordTarget int    `json:"word_target,omitempty" jsonschema_description:"Optional target visible-character count. Overrides the plan's 字数目标."`
	WordMin    int    `json:"word_min,omitempty" jsonschema_description:"Optional minimum visible-character count. Overrides the plan's 字数范围 lower bound."`
	WordMax    int    `json:"word_max,omitempty" jsonschema_description:"Optional maximum visible-character count. Overrides the plan's 字数范围 upper bound."`
}

type writingFinding struct {
	Severity string `json:"severity"`
	Type     string `json:"type"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
	Message  string `json:"message"`
}

type writingCheckResult struct {
	Schema     string           `json:"schema"`
	Action     string           `json:"action"`
	Path       string           `json:"path"`
	OK         bool             `json:"ok"`
	Summary    string           `json:"summary"`
	WordCount  int              `json:"word_count,omitempty"`
	WordTarget int              `json:"word_target,omitempty"`
	WordMin    int              `json:"word_min,omitempty"`
	WordMax    int              `json:"word_max,omitempty"`
	Blocking   int              `json:"blocking"`
	Advisory   int              `json:"advisory"`
	Findings   []writingFinding `json:"findings"`
	Truncated  bool             `json:"truncated,omitempty"`
}

const checkWritingDescription = "Check a writing workspace file for deterministic chapter, plan, or continuity-ledger problems. action=chapter checks visible word count, truncation, placeholders/refusal text, repetition, engineering-word leakage, large verbatim outline copying, and AI-style prose patterns. action=plan checks required plan fields and word-budget syntax. action=tracking checks required ledger sections, active-foreshadowing format, and size/count caps. Blocking findings are deterministic hard failures; advisory findings are style or workflow warnings. The tool never modifies files."

func newWritingCheckTool(workspace string) (agenttool.ToolDefinition, error) {
	tool, err := agenttool.InferTool("check_writing", checkWritingDescription, func(ctx context.Context, input checkWritingInput) (string, error) {
		result, err := runWritingCheck(ctx, workspace, input)
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("encode writing check result: %w", err)
		}
		return string(data), nil
	})
	if err != nil {
		return agenttool.ToolDefinition{}, err
	}
	return defineTool(tool, boundedReadDescriptor(agenttool.ToolSourceOther, config.AgentToolWritingCheck, agentschema.ToolResultRecoveryRerun))
}

func runWritingCheck(ctx context.Context, workspace string, input checkWritingInput) (writingCheckResult, error) {
	if err := ctx.Err(); err != nil {
		return writingCheckResult{}, err
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	switch action {
	case "chapter", "plan", "tracking":
	default:
		return writingCheckResult{}, fmt.Errorf("action must be chapter, plan, or tracking")
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return writingCheckResult{}, fmt.Errorf("path is required")
	}
	content, validUTF8, err := readWritingWorkspaceText(workspace, path)
	if err != nil {
		return writingCheckResult{}, err
	}
	result := writingCheckResult{Schema: writingCheckSchema, Action: action, Path: path}
	if input.WordTarget < 0 || input.WordMin < 0 || input.WordMax < 0 || (input.WordMin > 0 && input.WordMax > 0 && input.WordMin > input.WordMax) {
		return writingCheckResult{}, fmt.Errorf("word budget must be nonnegative and word_min must not exceed word_max")
	}
	if !validUTF8 {
		addWritingFinding(&result, "blocking", "invalid_utf8", 1, 1, "", "File is not valid UTF-8 text; repair the encoding or rewrite the affected text.")
	}

	switch action {
	case "chapter":
		plan, planNote := chapterPlanForCheck(ctx, workspace, path, input.PlanPath)
		if planNote != "" {
			addWritingFinding(&result, "advisory", "plan_unavailable", 0, 0, "", planNote)
		}
		if plan != "" {
			checkChapterPlan(plan, &result)
		}
		result.WordCount, result.WordTarget, result.WordMin, result.WordMax = checkChapterText(content, plan, input, &result)
	case "plan":
		checkChapterPlan(content, &result)
	case "tracking":
		checkTrackingLedger(content, &result)
	}
	finalizeWritingCheck(&result)
	return result, nil
}

func readWritingWorkspaceText(workspace, relativePath string) (string, bool, error) {
	if strings.TrimSpace(workspace) == "" {
		return "", false, fmt.Errorf("workspace is unavailable")
	}
	abs, err := book.SafePath(workspace, relativePath)
	if err != nil {
		return "", false, fmt.Errorf("resolve %s: %w", relativePath, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", relativePath, err)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s is not a regular file", relativePath)
	}
	if info.Size() > writingCheckMaxFileBytes {
		return "", false, fmt.Errorf("%s is too large to check (%d bytes)", relativePath, info.Size())
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", relativePath, err)
	}
	valid := utf8.Valid(data)
	text := string(data)
	text = strings.TrimPrefix(text, "\ufeff")
	if !valid {
		text = strings.ToValidUTF8(text, "�")
	}
	return text, valid, nil
}

// chapterPlanForCheck resolves the plan used for outline-copy and word-budget
// checks. An explicit path wins; otherwise a unique chapter-plans file with the
// same ch order prefix is used.
func chapterPlanForCheck(ctx context.Context, workspace, chapterPath, explicit string) (string, string) {
	if err := ctx.Err(); err != nil {
		return "", ""
	}
	planPath := strings.TrimSpace(explicit)
	if planPath == "" {
		planPath = discoverChapterPlanPath(workspace, chapterPath)
	}
	if planPath == "" {
		return "", "No chapter plan was supplied or uniquely discovered; outline-copy and plan word-budget checks were skipped."
	}
	content, valid, err := readWritingWorkspaceText(workspace, planPath)
	if err != nil {
		return "", "The chapter plan could not be read; outline-copy and plan word-budget checks were skipped."
	}
	if !valid {
		return "", "The chapter plan is not valid UTF-8; repair it before using it for chapter validation."
	}
	return content, ""
}

var chapterOrderPrefixPattern = regexp.MustCompile(`(?i)^(?:ch|s)(\d+)(?:[-_ .]|$)`)
var manuscriptOrderPartPattern = regexp.MustCompile(`(?i)^(?:v|ch|s)\d+`)

func discoverChapterPlanPath(workspace, chapterPath string) string {
	matches := chapterOrderPrefixPattern.FindStringSubmatch(filepath.Base(chapterPath))
	if len(matches) != 2 {
		return ""
	}
	prefix := strings.ToLower(manuscriptOrderPartPattern.FindString(filepath.Base(chapterPath)))
	parts := []string{}
	for _, part := range strings.Split(filepath.ToSlash(chapterPath), "/") {
		if key := manuscriptOrderPartPattern.FindString(part); key != "" {
			parts = append(parts, strings.ToLower(key))
		}
	}
	qualified := strings.Join(parts, "-")
	dir := filepath.Join(workspace, "setting", "chapter-plans")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	found := ""
	qualifiedFound := ""
	ambiguous := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" && ext != ".txt" {
			continue
		}
		base := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if base == qualified || strings.HasPrefix(base, qualified+"-") || strings.HasPrefix(base, qualified+"_") {
			if qualifiedFound != "" {
				return ""
			}
			qualifiedFound = entry.Name()
			continue
		}
		if base != prefix && !strings.HasPrefix(base, prefix+"-") && !strings.HasPrefix(base, prefix+"_") {
			continue
		}
		if found != "" {
			ambiguous = true
		}
		found = entry.Name()
	}
	if qualifiedFound != "" {
		found = qualifiedFound
	} else if ambiguous {
		return ""
	}
	if qualifiedFound == "" && qualified != prefix && found != "" {
		// Older books may use unqualified plan names. They remain usable only
		// when another volume/section cannot refer to that same leaf order.
		_ = filepath.WalkDir(filepath.Join(workspace, "chapters"), func(abs string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(workspace, abs)
			if err == nil && filepath.ToSlash(rel) != filepath.ToSlash(chapterPath) && strings.EqualFold(manuscriptOrderPartPattern.FindString(entry.Name()), prefix) {
				ambiguous = true
			}
			return nil
		})
		if ambiguous {
			return ""
		}
	}
	if found == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join("setting", "chapter-plans", found))
}

func addWritingFinding(result *writingCheckResult, severity, findingType string, line, column int, excerpt, message string) {
	if result == nil {
		return
	}
	severity = strings.TrimSpace(strings.ToLower(severity))
	if severity != "blocking" {
		severity = "advisory"
	}
	result.Findings = append(result.Findings, writingFinding{
		Severity: severity, Type: strings.TrimSpace(findingType),
		Line: line, Column: column,
		Excerpt: truncateWritingExcerpt(excerpt), Message: strings.TrimSpace(message),
	})
}

func truncateWritingExcerpt(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\t", " "))
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= writingCheckExcerptRunes {
		return value
	}
	return string(runes[:writingCheckExcerptRunes]) + "…"
}

func finalizeWritingCheck(result *writingCheckResult) {
	if result == nil {
		return
	}
	sort.SliceStable(result.Findings, func(i, j int) bool {
		left, right := result.Findings[i], result.Findings[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) < severityRank(right.Severity)
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		return left.Type < right.Type
	})
	for _, finding := range result.Findings {
		if finding.Severity == "blocking" {
			result.Blocking++
		} else {
			result.Advisory++
		}
	}
	if result.Findings == nil {
		result.Findings = []writingFinding{}
	}
	if len(result.Findings) > writingCheckMaxFindings {
		result.Findings = result.Findings[:writingCheckMaxFindings]
		result.Truncated = true
	}
	result.OK = result.Blocking == 0
	if len(result.Findings) == 0 {
		result.Summary = "No findings."
		return
	}
	result.Summary = fmt.Sprintf("%d blocking, %d advisory findings", result.Blocking, result.Advisory)
	if result.Truncated {
		result.Summary += " (output truncated)"
	}
}

func severityRank(severity string) int {
	if severity == "blocking" {
		return 0
	}
	return 1
}
