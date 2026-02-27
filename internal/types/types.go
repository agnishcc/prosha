package types

// AppState represents the current screen/mode of the application.
type AppState int

const (
	StateNoGit             AppState = iota // no .git found
	StateShellSetup                        // first-run shell integration prompt
	StateList                              // main list + detail view
	StateNewWorktree                       // modal: create new worktree
	StateEditWorktree                      // modal: rename branch
	StateDeleteConfirm                     // modal: confirm delete
	StateRightPaneFocused                  // Level 2 — commit list navigable in right pane
	StateCommitDetail                      // Level 3 — commit detail overlay
)

// Worktree holds metadata for a single git worktree.
type Worktree struct {
	Name        string   // user-defined display name (from metadata, or branch-derived)
	Path        string   // absolute filesystem path
	Branch      string   // git branch name, e.g. "feat/auth-refactor"
	IsMain      bool     // true for the primary worktree
	UpdatedAt   string   // human-readable relative time, e.g. "2 hours ago"
	Description string   // user-defined description (from metadata)
	CreatedFrom string   // short SHA of HEAD at creation time (from metadata)
	Ahead       int      // commits ahead of the default branch
	Behind      int      // commits behind the default branch
	IsMerged    bool     // whether branch is merged into the default branch
	Commits     []Commit // last 10 commits

	// Detail pane extras.
	HeadSHA         string // short SHA of current HEAD
	StatusChanged   int    // count of modified/deleted/renamed files
	StatusUntracked int    // count of untracked files
}

// PRInfo holds the result of a gh pr view call.
type PRInfo struct {
	State  string // "OPEN", "MERGED", "CLOSED"
	Number int
	URL    string
}

// Commit is a single git commit displayed in the detail pane.
type Commit struct {
	Hash    string // short hash, 7 chars
	Message string // subject line
	RelTime string // relative time, e.g. "3h ago"
}

// CommitDetail holds the full data for the commit detail overlay (Level 3).
type CommitDetail struct {
	ShortHash string
	Subject   string
	Body      string
	RelTime   string
	Files     []CommitFile
	Diff      []DiffLine
	Loaded    bool // false until the async fetch completes
}

// CommitFile is a single file entry in the "files changed" section.
type CommitFile struct {
	Status string // "M", "A", "D", "R"
	Path   string
}

// DiffLine is one line of the patch, categorised by type.
type DiffLine struct {
	// Type: "+" added, "-" removed, " " context, "@@" hunk header,
	// "diff" file header, "meta" (---, +++, index, etc.)
	Type    string
	Content string
}
