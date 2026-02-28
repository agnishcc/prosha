package ui

import (
	"errors"

	"github.com/agnishcc/worktree-tui/internal/git"
	"github.com/agnishcc/worktree-tui/internal/types"
	tea "github.com/charmbracelet/bubbletea"
)

// prCacheEntry stores the result of a gh pr view call.
// A nil *PRInfo means the branch has no open PR; a missing key means not yet fetched.
type prCacheEntry = *types.PRInfo

// Model is the root Bubbletea model.
type Model struct {
	state     types.AppState
	worktrees []types.Worktree
	repoName  string
	curBranch string
	cursor    int // 0 = "+ new worktree", 1..n = worktrees[cursor-1]
	width     int
	height    int

	// Repo-global header fields (refreshed on every loadWorktrees).
	remoteURL     string
	stashCount    int
	fetchedAgo    string
	defaultBranch string

	// PR badge cache: absent key = not fetched; nil value = no PR.
	ghAvailable bool
	prCache     map[string]prCacheEntry

	// hasCommits is false for a freshly-initialised repo with no commits yet.
	hasCommits bool

	// New worktree modal.
	newTypeIdx      int    // index into branchTypes
	newTypeListOpen bool   // whether the type-picker overlay is showing
	newDisplayName  string // shown in the list, allows spaces
	newBranch       string // git branch (auto-derived from type+name, then editable)
	newDescription  string // optional free-text description
	newActiveField  int    // 0=type, 1=name, 2=branch, 3=description
	newBranchEdited bool   // true once the user manually edits the branch field

	// Edit modal
	editName string

	// Commit drill-down (Levels 2 & 3).
	selectedCommitIndex int              // which commit is highlighted in Level 2
	commitDetailScroll  int              // vertical scroll offset for Level 3
	activeCommit        types.CommitDetail // full data shown in the Level 3 overlay

	// Transient error
	errMsg string
}

// InitialModel returns the starting model before any data is loaded.
func InitialModel() Model {
	return Model{state: types.StateNoGit}
}

// Init sends the initial git-detection command.
func (m Model) Init() tea.Cmd {
	return checkGitRepo
}

// ── Async messages ────────────────────────────────────────────────────────────

type gitCheckMsg struct{ isGit bool }

type worktreesLoadedMsg struct {
	worktrees     []types.Worktree
	repoName      string
	curBranch     string
	remoteURL     string
	stashCount    int
	fetchedAgo    string
	defaultBranch string
	ghAvailable   bool
	hasCommits    bool
	err           error
}

type gitInitMsg struct{ err error }
type worktreeCreatedMsg struct{ err error }
type worktreeDeletedMsg struct{ err error }
type worktreeRenamedMsg struct{ err error }

type prFetchedMsg struct {
	branch string
	info   *types.PRInfo // nil = no PR
}

type commitDetailLoadedMsg struct {
	detail *types.CommitDetail
	err    error
}

// ── Commands ──────────────────────────────────────────────────────────────────

func checkGitRepo() tea.Msg {
	return gitCheckMsg{isGit: git.IsGitRepo()}
}

func loadWorktrees() tea.Cmd {
	return func() tea.Msg {
		root, _ := git.GetRepoRoot()
		wts, err := git.ListWorktrees()
		if err != nil {
			return worktreesLoadedMsg{err: err}
		}
		name, branch, _ := git.GetRepoInfo()
		remoteURL, _ := git.GetRemoteURL()
		stashCount, _ := git.GetStashCount()
		fetchedAgo, _ := git.GetFetchedAgo()
		return worktreesLoadedMsg{
			worktrees:     wts,
			repoName:      name,
			curBranch:     branch,
			remoteURL:     remoteURL,
			stashCount:    stashCount,
			fetchedAgo:    fetchedAgo,
			defaultBranch: git.GetDefaultBranch(),
			ghAvailable:   git.IsGHAvailable(),
			hasCommits:    git.HasCommits(root),
		}
	}
}

func loadCommitDetail(worktreePath, sha string) tea.Cmd {
	return func() tea.Msg {
		detail, err := git.GetCommitDetail(worktreePath, sha)
		return commitDetailLoadedMsg{detail: detail, err: err}
	}
}

func fetchPR(branch string) tea.Cmd {
	return func() tea.Msg {
		info, _ := git.GetPRInfo(branch)
		return prFetchedMsg{branch: branch, info: info}
	}
}

func initGitRepo() tea.Msg {
	return gitInitMsg{err: git.InitRepo()}
}

// resetNewModal zeroes all new-worktree modal state.
func (m *Model) resetNewModal() {
	m.newTypeIdx = 0
	m.newTypeListOpen = false
	m.newDisplayName = ""
	m.newBranch = ""
	m.newDescription = ""
	m.newActiveField = 0
	m.newBranchEdited = false
}

func createWorktree(displayName, branch, path, description string) tea.Cmd {
	return func() tea.Msg {
		root, _ := git.GetRepoRoot()
		if !git.HasCommits(root) {
			return worktreeCreatedMsg{err: errors.New("repo has no commits yet — make an initial commit on main before creating worktrees")}
		}
		if err := git.AddWorktree(branch, path); err != nil {
			return worktreeCreatedMsg{err: err}
		}
		_ = git.SaveWorktreeMeta(branch, displayName, description)
		return worktreeCreatedMsg{}
	}
}

func deleteWorktree(branch, path string) tea.Cmd {
	return func() tea.Msg {
		_ = git.DeleteWorktreeMeta(branch)
		return worktreeDeletedMsg{err: git.RemoveWorktree(path)}
	}
}

func renameWorktree(oldName, newName string) tea.Cmd {
	return func() tea.Msg { return worktreeRenamedMsg{err: git.RenameBranch(oldName, newName)} }
}
