package ui

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/agnishcc/worktree-tui/internal/git"
	"github.com/agnishcc/worktree-tui/internal/types"
	tea "github.com/charmbracelet/bubbletea"
)

// branchTypes is the full list shown in the type-picker overlay.
var branchTypes = []string{
	"feat", "fix", "chore", "docs", "refactor",
	"test", "style", "ci", "perf", "release",
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case gitCheckMsg:
		if !msg.isGit {
			m.state = types.StateNoGit
			return m, nil
		}
		if git.IsShellIntegrated() {
			m.state = types.StateList
			return m, loadWorktrees()
		}
		m.state = types.StateShellSetup
		return m, nil

	case worktreesLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.worktrees = msg.worktrees
		m.repoName = msg.repoName
		m.curBranch = msg.curBranch
		m.remoteURL = msg.remoteURL
		m.stashCount = msg.stashCount
		m.fetchedAgo = msg.fetchedAgo
		m.defaultBranch = msg.defaultBranch
		m.ghAvailable = msg.ghAvailable
		if m.prCache == nil {
			m.prCache = make(map[string]prCacheEntry)
		}
		m.state = types.StateList
		if m.cursor > len(m.worktrees) {
			m.cursor = len(m.worktrees)
		}
		return m, m.maybeFetchPR()

	case prFetchedMsg:
		if m.prCache == nil {
			m.prCache = make(map[string]prCacheEntry)
		}
		m.prCache[msg.branch] = msg.info
		return m, nil

	case commitDetailLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		if msg.detail != nil {
			m.activeCommit = *msg.detail
		}
		return m, nil

	case gitInitMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.state = types.StateList
		return m, loadWorktrees()

	case worktreeCreatedMsg:
		m.state = types.StateList
		m.resetNewModal()
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		}
		return m, loadWorktrees()

	case worktreeDeletedMsg:
		m.state = types.StateList
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		}
		if m.cursor > 0 {
			m.cursor--
		}
		return m, loadWorktrees()

	case worktreeRenamedMsg:
		m.state = types.StateList
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		}
		return m, loadWorktrees()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.errMsg != "" {
		m.errMsg = ""
		return m, nil
	}
	switch m.state {
	case types.StateNoGit:
		return m.handleNoGit(msg)
	case types.StateShellSetup:
		return m.handleShellSetup(msg)
	case types.StateList:
		return m.handleList(msg)
	case types.StateNewWorktree:
		return m.handleNewWorktree(msg)
	case types.StateEditWorktree:
		return m.handleEditWorktree(msg)
	case types.StateDeleteConfirm:
		return m.handleDeleteConfirm(msg)
	case types.StateRightPaneFocused:
		return m.handleRightPaneFocused(msg)
	case types.StateCommitDetail:
		return m.handleCommitDetail(msg)
	}
	return m, nil
}

func (m Model) handleNoGit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		return m, initGitRepo
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleShellSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		_ = git.SetupShellIntegration()
		_ = git.MarkShellIntegrated()
		m.state = types.StateList
		return m, loadWorktrees()
	case "n", "esc", "q":
		_ = git.MarkShellIntegrated()
		m.state = types.StateList
		return m, loadWorktrees()
	}
	return m, nil
}

func (m Model) handleList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	total := len(m.worktrees) + 1
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, m.maybeFetchPR()
	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
		}
		return m, m.maybeFetchPR()
	case "enter":
		if m.cursor == 0 {
			m.openNewModal()
		} else if m.cursor-1 < len(m.worktrees) {
			m.selectedCommitIndex = 0
			m.state = types.StateRightPaneFocused
		}
	case "n":
		m.openNewModal()
	case "d":
		if m.cursor > 0 && !m.worktrees[m.cursor-1].IsMain {
			m.state = types.StateDeleteConfirm
		}
	case "e":
		if m.cursor > 0 {
			m.editName = m.worktrees[m.cursor-1].Branch
			m.state = types.StateEditWorktree
		}
	case "c":
		if m.cursor > 0 {
			_ = git.WriteCDPath(m.worktrees[m.cursor-1].Path)
			return m, tea.Quit
		}
	}
	return m, nil
}

// maybeFetchPR fires a PR fetch for the currently selected worktree if it
// hasn't been fetched yet and gh is available.
func (m Model) maybeFetchPR() tea.Cmd {
	if !m.ghAvailable || m.cursor == 0 || m.cursor-1 >= len(m.worktrees) {
		return nil
	}
	wt := m.worktrees[m.cursor-1]
	if wt.IsMain {
		return nil
	}
	if _, cached := m.prCache[wt.Branch]; cached {
		return nil
	}
	return fetchPR(wt.Branch)
}

func (m *Model) openNewModal() {
	m.resetNewModal()
	m.state = types.StateNewWorktree
}

// handleNewWorktree dispatches to the type-list handler when the overlay is
// open, otherwise manages the four-field form.
func (m Model) handleNewWorktree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.newTypeListOpen {
		return m.handleTypeList(msg)
	}

	switch msg.Type {

	case tea.KeyEsc:
		m.state = types.StateList
		m.resetNewModal()

	// Tab and Down both advance to the next field.
	case tea.KeyTab, tea.KeyDown:
		m.newActiveField = (m.newActiveField + 1) % 4

	case tea.KeyUp:
		m.newActiveField = (m.newActiveField + 3) % 4 // wraps backward

	case tea.KeyEnter:
		if m.newActiveField == 0 {
			// Open the type picker.
			m.newTypeListOpen = true
		} else if m.newDisplayName != "" && m.newBranch != "" {
			root, _ := git.GetRepoRoot()
			safePath := strings.ReplaceAll(m.newBranch, "/", "-")
			wtPath := filepath.Join(root, ".wt", safePath)
			return m, createWorktree(m.newDisplayName, m.newBranch, wtPath, m.newDescription)
		}

	case tea.KeySpace:
		m.appendRunes([]rune{' '})

	case tea.KeyBackspace:
		m.deleteChar()

	case tea.KeyRunes:
		m.appendRunes(msg.Runes)
	}

	return m, nil
}

// handleTypeList handles key input while the type-picker overlay is visible.
func (m Model) handleTypeList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.newTypeIdx > 0 {
			m.newTypeIdx--
		}
	case "down", "j":
		if m.newTypeIdx < len(branchTypes)-1 {
			m.newTypeIdx++
		}
	case "enter":
		m.newTypeListOpen = false
		m.recalcBranch()
	case "esc":
		m.newTypeListOpen = false
	}
	return m, nil
}

func (m Model) handleRightPaneFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var commits []types.Commit
	if m.cursor > 0 && m.cursor-1 < len(m.worktrees) {
		commits = m.worktrees[m.cursor-1].Commits
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.state = types.StateList
	case "up", "k":
		if m.selectedCommitIndex > 0 {
			m.selectedCommitIndex--
		}
	case "down", "j":
		if m.selectedCommitIndex < len(commits)-1 {
			m.selectedCommitIndex++
		}
	case "enter":
		if len(commits) > 0 && m.selectedCommitIndex < len(commits) {
			c := commits[m.selectedCommitIndex]
			wt := m.worktrees[m.cursor-1]
			// Pre-populate with what we already know; full data arrives async.
			m.activeCommit = types.CommitDetail{
				ShortHash: c.Hash,
				Subject:   c.Message,
				RelTime:   c.RelTime,
			}
			m.commitDetailScroll = 0
			m.state = types.StateCommitDetail
			return m, loadCommitDetail(wt.Path, c.Hash)
		}
	}
	return m, nil
}

func (m Model) handleCommitDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = types.StateRightPaneFocused
	case "up", "k":
		if m.commitDetailScroll > 0 {
			m.commitDetailScroll--
		}
	case "down", "j":
		m.commitDetailScroll++
	}
	return m, nil
}

// deleteChar removes the last rune from the currently active field.
func (m *Model) deleteChar() {
	switch m.newActiveField {
	case 1:
		m.newDisplayName = dropLast(m.newDisplayName)
		m.recalcBranch()
	case 2:
		m.newBranch = dropLast(m.newBranch)
		m.newBranchEdited = true
	case 3:
		m.newDescription = dropLast(m.newDescription)
	}
	// Field 0 (type) ignores backspace — use the type picker instead.
}

// appendRunes adds typed characters to the active field.
// The Name field allows any characters including spaces.
// The Branch field is locked to lowercase with hyphens (auto or manual).
func (m *Model) appendRunes(runes []rune) {
	switch m.newActiveField {
	case 0:
		// Type is chosen via picker, not typed.
	case 1: // Name — full free text, spaces allowed
		m.newDisplayName += string(runes)
		m.recalcBranch()
	case 2: // Branch — user is taking manual control; spaces become hyphens
		for _, r := range runes {
			if unicode.IsSpace(r) {
				r = '-'
			}
			m.newBranch += string(r)
		}
		m.newBranchEdited = true
	case 3: // Description — full free text
		m.newDescription += string(runes)
	}
}

// recalcBranch rebuilds the branch name from type + slugified display name,
// unless the user has manually edited it.
func (m *Model) recalcBranch() {
	if m.newBranchEdited {
		return
	}
	slug := slugify(m.newDisplayName)
	if slug == "" {
		m.newBranch = branchTypes[m.newTypeIdx]
	} else {
		m.newBranch = branchTypes[m.newTypeIdx] + "/" + slug
	}
}

func (m Model) handleEditWorktree(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = types.StateList
		m.editName = ""
	case tea.KeyEnter:
		if m.cursor > 0 && m.editName != "" {
			wt := m.worktrees[m.cursor-1]
			if wt.Branch != m.editName {
				return m, renameWorktree(wt.Branch, m.editName)
			}
		}
		m.state = types.StateList
	case tea.KeyBackspace:
		m.editName = dropLast(m.editName)
	case tea.KeySpace:
		m.editName += " "
	case tea.KeyRunes:
		m.editName += string(msg.Runes)
	}
	return m, nil
}

func (m Model) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.cursor > 0 {
			wt := m.worktrees[m.cursor-1]
			return m, deleteWorktree(wt.Branch, wt.Path)
		}
	case "n", "esc":
		m.state = types.StateList
	}
	return m, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func dropLast(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

// slugify converts a display name to a lowercase hyphenated git branch suffix.
// Spaces, underscores and existing hyphens all become a single hyphen.
// The slash character is preserved so "feat/something" round-trips correctly.
func slugify(s string) string {
	var b strings.Builder
	prevSep := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevSep = false
		case r == '/':
			b.WriteRune('/')
			prevSep = false
		case unicode.IsSpace(r) || r == '_' || r == '-':
			if !prevSep && b.Len() > 0 {
				b.WriteRune('-')
				prevSep = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-/")
}
