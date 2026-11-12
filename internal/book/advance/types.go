// Package advance owns long-form fiction continuity. Its state is a versioned
// book artifact; Agent execution and recovery remain in the product journal.
package advance

import workspacechange "github.com/vevila2020/writing_platform_ai/internal/workspace/change"

const StatePath = "setting/continuity.json"
const Schema = "inkdock.advance/v1"

var viewPaths = []string{"setting/progress.md", "setting/character-states.md", "setting/tracking.md"}

// Context selects the bounded working set for the next chapter. Full retired
// facts remain in State; removing something here does not delete its history.
type Context struct {
	Language         string   `json:"language" jsonschema:"enum=zh-CN,enum=en-US" jsonschema_description:"Language of generated book documents; preserve the existing book language."`
	Position         string   `json:"position" jsonschema_description:"Current story time, location and situation after the latest written chapter, not future plot."`
	Constraints      []string `json:"constraints" jsonschema_description:"At most 6 current long-term creative constraints."`
	ActiveCharacters []string `json:"active_characters" jsonschema_description:"Names of at most 20 existing character snapshots needed next."`
	ActiveThreads    []string `json:"active_threads" jsonschema_description:"IDs of at most 8 active foreshadowing threads needed next."`
	ActiveSecrets    []string `json:"active_secrets" jsonschema_description:"IDs of at most 12 information boundaries needed next."`
	NextCommitments  []string `json:"next_commitments" jsonschema_description:"At most 6 concrete promises the next chapter must address."`
	Risks            []string `json:"risks" jsonschema_description:"At most 6 current continuity risks; preserve unresolved risks."`
}

type Character struct {
	Name          string   `json:"name"`
	Location      string   `json:"location"`
	Condition     string   `json:"condition"`
	Goal          string   `json:"goal"`
	Possessions   []string `json:"possessions"`
	Relationships []string `json:"relationships"`
	Knowledge     []string `json:"knowledge" jsonschema_description:"Only what this character knows or believes; distinguish guesses from facts."`
}

type Thread struct {
	ID     string `json:"id" jsonschema_description:"Stable F-number, for example F001. Reuse an existing ID when updating."`
	Label  string `json:"label"`
	Status string `json:"status" jsonschema:"enum=active,enum=resolved,enum=abandoned"`
	Detail string `json:"detail" jsonschema_description:"What has actually happened; resolution or abandonment needs a reason. Future reveal plans belong in outlines."`
}

type Secret struct {
	ID              string `json:"id"`
	AuthorTruth     string `json:"author_truth"`
	ReaderKnowledge string `json:"reader_knowledge" jsonschema_description:"What the prose has actually revealed; empty when unrevealed. Do not copy hidden truth here."`
}

// Update is one semantic chapter transaction. Character/thread/secret lists
// upsert complete affected records; Context replaces only the working set.
type Update struct {
	Summary          string      `json:"summary" jsonschema_description:"Compact actual outcome and consequences of this chapter, at most 3072 UTF-8 bytes."`
	Context          Context     `json:"context"`
	Characters       []Character `json:"characters"`
	Threads          []Thread    `json:"threads"`
	Secrets          []Secret    `json:"secrets"`
	RetiredContext   []string    `json:"retired_context" jsonschema_description:"Explicitly name removed constraints, risks and unfulfilled commitments, with the reason they no longer apply."`
	Unresolved       []Issue     `json:"unresolved" jsonschema_description:"Complete current list of later prose conflicts after a revision. New chapters are blocked until these are resolved."`
	BaselineProgress *string     `json:"baseline_progress,omitempty" jsonschema_description:"Only when revising an imported chapter: corrected historical progress through the import checkpoint."`
}

type Issue struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Chapter struct {
	Path           string   `json:"path"`
	Revision       string   `json:"revision"`
	PlanPath       string   `json:"plan_path,omitempty"`
	PlanRevision   string   `json:"plan_revision,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	RetiredContext []string `json:"retired_context,omitempty"`
}

type State struct {
	Schema           string               `json:"schema"`
	Revision         int                  `json:"revision"`
	Context          Context              `json:"context"`
	Chapters         []Chapter            `json:"chapters"`
	Characters       map[string]Character `json:"characters"`
	Threads          map[string]Thread    `json:"threads"`
	Secrets          map[string]Secret    `json:"secrets"`
	Unresolved       []Issue              `json:"unresolved"`
	ImportedThrough  string               `json:"imported_through,omitempty"`
	BaselineProgress string               `json:"baseline_progress,omitempty"`
	LastSubmission   string               `json:"last_submission"`
}

// Preparation binds a future commit to the exact prose, plan and state/view
// revisions read during preparation. It exposes only the selected working set.
type Preparation struct {
	BaseRevision        string      `json:"base_revision"`
	Initialized         bool        `json:"initialized"`
	Committed           bool        `json:"committed"`
	Path                string      `json:"path"`
	PlanPath            string      `json:"plan_path,omitempty"`
	Context             Context     `json:"context"`
	Characters          []Character `json:"characters"`
	Threads             []Thread    `json:"threads"`
	Secrets             []Secret    `json:"secrets"`
	RecentChapters      []Chapter   `json:"recent_chapters"`
	Unresolved          []Issue     `json:"unresolved"`
	ViewConflicts       []string    `json:"view_conflicts"`
	NeedsInitialization bool        `json:"needs_initialization"`
	Uncommitted         []string    `json:"uncommitted"`
}

// CommitRequest must be prepared again after the final prose edit. Initialize
// adopts existing prose without rewriting it; commit validates final prose.
type CommitRequest struct {
	Action       string
	Path         string
	PlanPath     string
	BaseRevision string
	Update       Update
	Metadata     workspacechange.ChangeMetadata
}

type Receipt struct {
	Mutation      *workspacechange.ToolReceipt `json:"-"`
	Schema        string                       `json:"schema"`
	Status        string                       `json:"status"`
	Path          string                       `json:"path"`
	StateRevision int                          `json:"state_revision"`
	ChangeGroupID string                       `json:"change_group_id,omitempty"`
	ChangedPaths  []string                     `json:"changed_paths"`
}
