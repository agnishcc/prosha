package ui

import (
	"fmt"
	"strings"

	"github.com/agnishcc/worktree-tui/internal/types"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	switch m.state {
	case types.StateNoGit:
		return m.viewNoGit()
	case types.StateShellSetup:
		return m.viewShellSetup()
	default:
		return m.viewMain()
	}
}

// ── Full-screen states ────────────────────────────────────────────────────────

func (m Model) viewNoGit() string {
	header := m.renderHeader()
	body := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(header)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center,
			dimStyle.Render("No git repository found."),
			"",
			dimStyle.Render("Would you like to initialise one?"),
			"",
			m.renderHints("i  init", "q  quit"),
		))
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func (m Model) viewShellSetup() string {
	header := m.renderHeader()
	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		accentStyle.Render("⚡ Add shell integration for cd-on-exit?"),
		"",
		dimStyle.Render("This adds a wt() function to your shell rc file."),
		dimStyle.Render("Invoke wt instead of worktree-tui to use it."),
		"",
		m.renderHints("y  add it", "n  skip"),
	))
	body := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(header)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(modal)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

// ── Main list + detail view ───────────────────────────────────────────────────

func (m Model) viewMain() string {
	switch m.state {
	case types.StateNewWorktree:
		return m.centerModal(m.renderNewModal())
	case types.StateEditWorktree:
		return m.centerModal(m.renderEditModal())
	case types.StateDeleteConfirm:
		return m.centerModal(m.renderDeleteModal())
	case types.StateCommitDetail:
		return m.centerModal(m.renderCommitDetailOverlay())
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)

	paneOuterH := m.height - headerH - footerH - 2
	if paneOuterH < 3 {
		paneOuterH = 3
	}
	leftOuterW := m.width / 4
	if leftOuterW < 22 {
		leftOuterW = 22
	}
	rightOuterW := m.width - leftOuterW - 2

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderLeftPane(leftOuterW, paneOuterH),
		"  ",
		m.renderRightPane(rightOuterW, paneOuterH),
	)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", panes, "", footer)
}

func (m Model) centerModal(modal string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// ── Header ────────────────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	innerW := m.width - 4
	if innerW < 4 {
		innerW = 4
	}

	sep := dimStyle.Render(" · ")
	sepW := lipgloss.Width(sep)

	appName := headerTextStyle.Render("⎇  worktree")

	// Line-1 candidates (excludes fetchedAgo, which is always on line 2).
	var candidates []string
	if m.remoteURL != "" {
		candidates = append(candidates, dimStyle.Render(m.remoteURL))
	}
	if n := len(m.worktrees); n > 0 {
		candidates = append(candidates, dimStyle.Render(fmt.Sprintf("%d worktrees", n)))
	}
	if m.stashCount > 0 {
		candidates = append(candidates, warningStyle.Render(fmt.Sprintf("✦ %d stashed", m.stashCount)))
	}

	// Greedily fit sections onto line 1; overflow moves to line 2 as whole units.
	used := lipgloss.Width(appName)
	var line1Sections []string
	var overflowSections []string
	overflowing := false
	for _, c := range candidates {
		needed := sepW + lipgloss.Width(c)
		if !overflowing && used+needed <= innerW {
			line1Sections = append(line1Sections, c)
			used += needed
		} else {
			overflowing = true
			overflowSections = append(overflowSections, c)
		}
	}

	// Build line 1: appName on left, fitted sections on right.
	right1 := strings.Join(line1Sections, sep)
	gap1 := innerW - lipgloss.Width(appName) - lipgloss.Width(right1)
	if gap1 < 0 {
		gap1 = 0
	}
	line1 := appName + strings.Repeat(" ", gap1) + right1

	// Build line 2: overflow sections on left, fetchedAgo always on right.
	fetchStr := ""
	if m.fetchedAgo != "" {
		fetchStr = dimStyle.Render("fetched " + m.fetchedAgo)
	}

	var line2 string
	if len(overflowSections) > 0 || fetchStr != "" {
		left2 := strings.Join(overflowSections, sep)
		if left2 != "" && fetchStr != "" {
			gap2 := innerW - lipgloss.Width(left2) - lipgloss.Width(fetchStr)
			if gap2 < 1 {
				gap2 = 1
			}
			line2 = left2 + strings.Repeat(" ", gap2) + fetchStr
		} else if fetchStr != "" {
			gap2 := innerW - lipgloss.Width(fetchStr)
			if gap2 < 0 {
				gap2 = 0
			}
			line2 = strings.Repeat(" ", gap2) + fetchStr
		} else {
			line2 = left2
		}
	}

	content := line1
	if line2 != "" {
		content = line1 + "\n" + line2
	}
	return headerBoxStyle.Width(innerW).Render(content)
}

// ── Panes ─────────────────────────────────────────────────────────────────────

func (m Model) renderLeftPane(outerW, outerH int) string {
	innerW := outerW - 2
	innerH := outerH - 2

	rows := []string{m.renderItem(0, "+ new worktree", innerW, true)}
	for i, wt := range m.worktrees {
		rows = append(rows, m.renderItem(i+1, wt.Name, innerW, false))
	}

	content := strings.Join(rows, "\n")
	lines := strings.Count(content, "\n") + 1
	for i := lines; i < innerH; i++ {
		content += "\n"
	}

	// Dim the left border when right pane is focused.
	style := activePaneStyle
	if m.state == types.StateRightPaneFocused {
		style = inactivePaneStyle
	}
	return style.Width(innerW).Height(innerH).Render(content)
}

func (m Model) renderItem(idx int, name string, innerW int, isNewRow bool) string {
	selected := m.cursor == idx
	maxNameW := innerW - 2
	text := truncate(name, maxNameW)

	if isNewRow {
		if !m.hasCommits {
			// No commits yet — dim the row and add an inline hint.
			label := padRight(text, maxNameW-18) + "  no commits yet"
			if selected {
				return "  " + dimStyle.Render(label)
			}
			return "  " + dimStyle.Render(label)
		}
		if selected {
			return selectedAccentStyle.Render("▌") + " " + newItemActiveStyle.Render(padRight(text, maxNameW))
		}
		return "  " + newItemFaintStyle.Render(padRight(text, maxNameW))
	}
	if selected {
		return selectedAccentStyle.Render("▌") + " " + selectedItemStyle.Render(padRight(text, maxNameW))
	}
	return "  " + normalItemStyle.Render(padRight(text, maxNameW))
}

func (m Model) renderRightPane(outerW, outerH int) string {
	innerW := outerW - 2
	innerH := outerH - 2

	var content string
	if m.cursor == 0 {
		if !m.hasCommits {
			content = lipgloss.JoinVertical(lipgloss.Left,
				dimStyle.Render("Worktrees require at least one commit."),
				"",
				dimStyle.Render("Run  git commit  on the main branch first,"),
				dimStyle.Render("then worktrees can be created here."),
			)
		} else {
			content = dimStyle.Render(
				"Select \"+ new worktree\" and press enter to create\n" +
					"or press  n  from anywhere.",
			)
		}
	} else if idx := m.cursor - 1; idx < len(m.worktrees) {
		content = m.renderDetail(m.worktrees[idx], innerW)
	}

	// Activate the right border when focus shifts here (Level 2).
	style := inactivePaneStyle
	if m.state == types.StateRightPaneFocused {
		style = activeRightPaneStyle
	}
	return style.Width(innerW).Height(innerH).Render(content)
}

func (m Model) renderDetail(wt types.Worktree, innerW int) string {
	var sb strings.Builder

	// ── Title line with optional PR badge ─────────────────────────────────────
	title := detailTitleStyle.Render(wt.Name)
	badge := ""
	if !wt.IsMain {
		badge = m.prBadge(wt.Branch)
	}
	if badge != "" {
		gap := innerW - lipgloss.Width(title) - lipgloss.Width(badge)
		if gap < 1 {
			gap = 1
		}
		sb.WriteString(title + strings.Repeat(" ", gap) + badge)
	} else {
		sb.WriteString(title)
	}
	sb.WriteString("\n\n")

	// ── Metadata rows ──────────────────────────────────────────────────────────
	ind := detailIndicatorStyle.Render("◎")
	row := func(label, value string) {
		sb.WriteString(fmt.Sprintf("%s  %s  %s\n",
			ind,
			detailLabelStyle.Render(fmt.Sprintf("%-8s", label)),
			value,
		))
	}

	row("Branch", detailValueStyle.Render(wt.Branch))
	row("Path", detailValueStyle.Render(truncate(wt.Path, innerW-22)))
	row("Updated", detailValueStyle.Render(wt.UpdatedAt))

	// HEAD sha — Flamingo color.
	if wt.HeadSHA != "" {
		row("HEAD", lipgloss.NewStyle().Foreground(clrFlamingo).Render(wt.HeadSHA))
	}

	// Status — dirty / clean.
	if wt.StatusChanged > 0 || wt.StatusUntracked > 0 {
		var parts []string
		if wt.StatusChanged > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(clrRed).Render("●")+
				detailValueStyle.Render(fmt.Sprintf(" %d changed", wt.StatusChanged)))
		}
		if wt.StatusUntracked > 0 {
			parts = append(parts, detailValueStyle.Render(fmt.Sprintf("%d untracked", wt.StatusUntracked)))
		}
		row("Status", strings.Join(parts, dimStyle.Render("  ")))
	} else {
		row("Status", lipgloss.NewStyle().Foreground(clrGreen).Render("✓ clean"))
	}

	// Sync — ahead/behind default branch (skip for main worktree).
	if !wt.IsMain {
		def := m.defaultBranch
		if def == "" {
			def = "main"
		}
		switch {
		case wt.Ahead > 0 && wt.Behind > 0:
			row("Sync", lipgloss.NewStyle().Foreground(clrYellow).Render(
				fmt.Sprintf("↑%d ↓%d diverged from %s", wt.Ahead, wt.Behind, def)))
		case wt.Ahead > 0:
			row("Sync", detailValueStyle.Render(fmt.Sprintf("↑%d ahead of %s", wt.Ahead, def)))
		case wt.Behind > 0:
			row("Sync", lipgloss.NewStyle().Foreground(clrYellow).Render(
				fmt.Sprintf("↓%d behind %s", wt.Behind, def)))
		default:
			row("Sync", lipgloss.NewStyle().Foreground(clrGreen).Render(fmt.Sprintf("✓ up to date with %s", def)))
		}

		if wt.CreatedFrom != "" {
			row("Created", detailValueStyle.Render("from "+wt.CreatedFrom))
		}
	}

	// ── Description ────────────────────────────────────────────────────────────
	if wt.Description != "" {
		sb.WriteString("\n")
		divW := innerW - 14
		if divW < 3 {
			divW = 3
		}
		sb.WriteString(sectionDividerStyle.Render("Description " + strings.Repeat("─", divW)))
		sb.WriteString("\n\n")
		for _, line := range wrapWords(wt.Description, innerW) {
			sb.WriteString(dimStyle.Render(line) + "\n")
		}
	}

	// ── Commits ────────────────────────────────────────────────────────────────
	if len(wt.Commits) > 0 {
		sb.WriteString("\n")
		divW := innerW - 10
		if divW < 3 {
			divW = 3
		}
		hint := ""
		if m.state == types.StateRightPaneFocused {
			hint = "  " + dimStyle.Render("enter to view")
		}
		sb.WriteString(sectionDividerStyle.Render("Commits "+strings.Repeat("─", divW)) + hint)
		sb.WriteString("\n\n")
		for i, c := range wt.Commits {
			maxMsg := innerW - 28
			if maxMsg < 10 {
				maxMsg = 10
			}
			selected := m.state == types.StateRightPaneFocused && i == m.selectedCommitIndex
			if selected {
				sb.WriteString(fmt.Sprintf("%s %s  %s  %s\n",
					selectedAccentStyle.Render("▌"),
					lipgloss.NewStyle().Foreground(clrFlamingo).Render(c.Hash),
					selectedItemStyle.Render(truncate(c.Message, maxMsg)),
					commitTimeStyle.Render(c.RelTime),
				))
			} else {
				sb.WriteString(fmt.Sprintf("%s %s  %s  %s\n",
					commitDotStyle.Render("●"),
					commitHashStyle.Render(c.Hash),
					commitMsgStyle.Render(truncate(c.Message, maxMsg)),
					commitTimeStyle.Render(c.RelTime),
				))
			}
		}
	}

	return sb.String()
}

// prBadge returns the styled PR badge string for a branch, or "" if hidden.
func (m Model) prBadge(branch string) string {
	if !m.ghAvailable {
		return ""
	}
	info, cached := m.prCache[branch]
	if !cached {
		return "" // still fetching — badge appears when result arrives
	}
	if info == nil {
		return lipgloss.NewStyle().Foreground(clrPRNone).Render("no PR")
	}
	switch strings.ToUpper(info.State) {
	case "OPEN":
		return lipgloss.NewStyle().Foreground(clrPROpen).Render(fmt.Sprintf("● open  #%d", info.Number))
	case "MERGED":
		return lipgloss.NewStyle().Foreground(clrPRMerged).Render(fmt.Sprintf("✓ merged  #%d", info.Number))
	case "CLOSED":
		return lipgloss.NewStyle().Foreground(clrPRClosed).Render(fmt.Sprintf("✗ closed  #%d", info.Number))
	}
	return ""
}

// ── Modals ────────────────────────────────────────────────────────────────────

// renderNewModal switches between the type-picker overlay and the main form.
func (m Model) renderNewModal() string {
	if m.newTypeListOpen {
		return m.renderTypeListModal()
	}
	return m.renderNewFormModal()
}

// renderTypeListModal renders the branch-type selection overlay.
func (m Model) renderTypeListModal() string {
	var rows []string
	for i, t := range branchTypes {
		if i == m.newTypeIdx {
			rows = append(rows, selectedAccentStyle.Render("▌")+" "+selectedItemStyle.Render(t))
		} else {
			rows = append(rows, "  "+dimStyle.Render(t))
		}
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		modalTitleStyle.Render("Select Type"),
		"",
		strings.Join(rows, "\n"),
		"",
		m.renderHints("↑↓  navigate", "enter  select", "esc  close"),
	)
	return modalStyle.Render(content)
}

// renderNoCommitsModal is shown instead of the create form when the repo has no commits.
func (m Model) renderNoCommitsModal() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		modalTitleStyle.Render("New Worktree"),
		"",
		dangerStyle.Render("✗  Cannot create worktree"),
		"",
		dimStyle.Render("No commits on main yet."),
		dimStyle.Render("Make an initial commit first."),
		"",
		m.renderHints("esc  close"),
	)
	return modalStyle.Render(content)
}

// renderNewFormModal renders the four-field create form.
func (m Model) renderNewFormModal() string {
	if !m.hasCommits {
		return m.renderNoCommitsModal()
	}
	fieldLabel := func(label string, idx int) string {
		if m.newActiveField == idx {
			return accentStyle.Render(label)
		}
		return modalLabelStyle.Render(label)
	}

	// Type field (not a text input — uses picker).
	typeVal := branchTypes[m.newTypeIdx]
	var typeDisplay string
	if m.newActiveField == 0 {
		typeDisplay = selectedItemStyle.Render(typeVal) + "  " + dimStyle.Render("↵ change")
	} else {
		typeDisplay = dimStyle.Render(typeVal)
	}

	// Hints depend on which field is focused.
	var hints string
	if m.newActiveField == 0 {
		hints = m.renderHints("enter  change type", "tab/↑↓  navigate", "esc  cancel")
	} else {
		hints = m.renderHints("enter  create", "tab/↑↓  navigate", "esc  cancel")
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		modalTitleStyle.Render("New Worktree"),
		"",
		fieldLabel("Type", 0),
		typeDisplay,
		"",
		fieldLabel("Name", 1),
		m.fieldInput(m.newDisplayName, m.newActiveField == 1),
		"",
		fieldLabel("Branch", 2),
		m.fieldInput(m.newBranch, m.newActiveField == 2),
		"",
		fieldLabel("Description", 3),
		m.fieldInput(m.newDescription, m.newActiveField == 3),
		"",
		hints,
	)
	return modalStyle.Render(content)
}

func (m Model) renderEditModal() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		modalTitleStyle.Render("Edit Worktree"),
		"",
		modalLabelStyle.Render("Branch name"),
		m.fieldInput(m.editName, true),
		"",
		m.renderHints("enter  save", "esc  cancel"),
	)
	return modalStyle.Render(content)
}

func (m Model) renderDeleteModal() string {
	name := ""
	if m.cursor > 0 && m.cursor-1 < len(m.worktrees) {
		name = m.worktrees[m.cursor-1].Name
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		dangerStyle.Render("Delete "+name+"?"),
		"",
		dimStyle.Render("This cannot be undone."),
		"",
		m.renderHints("y  confirm", "n / esc  cancel"),
	)
	return modalStyle.Render(content)
}

// fieldInput renders an input line. When active it shows a block cursor.
func (m Model) fieldInput(value string, active bool) string {
	if active {
		return modalInputStyle.Render(value) + accentStyle.Render("█")
	}
	return dimStyle.Render(value + " ")
}

// renderCommitDetailOverlay renders the Level 3 centered modal.
func (m Model) renderCommitDetailOverlay() string {
	outerW := m.width * 80 / 100
	outerH := m.height * 80 / 100
	if outerW < 40 {
		outerW = 40
	}
	if outerH < 10 {
		outerH = 10
	}
	// Border (1 each side) + Padding (2 left/right, 1 top/bottom).
	innerW := outerW - 6
	innerH := outerH - 4

	// Reserve 2 lines at the bottom for blank line + footer hints.
	scrollH := innerH - 2
	if scrollH < 1 {
		scrollH = 1
	}

	cd := m.activeCommit
	var lines []string

	// ── Header: hash + reltime ─────────────────────────────────────────────
	hashStr := lipgloss.NewStyle().Foreground(clrFlamingo).Render(cd.ShortHash)
	timeStr := lipgloss.NewStyle().Foreground(clrCommitContext).Render(cd.RelTime)
	gap := innerW - lipgloss.Width(hashStr) - lipgloss.Width(timeStr)
	if gap < 1 {
		gap = 1
	}
	lines = append(lines, hashStr+strings.Repeat(" ", gap)+timeStr)
	lines = append(lines, "")

	// ── Subject ────────────────────────────────────────────────────────────
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(clrCommitTitle).
		Render(truncate(cd.Subject, innerW)))

	// ── Body (optional) ────────────────────────────────────────────────────
	if cd.Body != "" {
		lines = append(lines, "")
		for _, bl := range wrapWords(cd.Body, innerW) {
			lines = append(lines, lipgloss.NewStyle().Foreground(clrCommitBody).Render(bl))
		}
	}

	if !cd.Loaded {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Loading…"))
	} else {
		// ── Files changed ──────────────────────────────────────────────────
		if len(cd.Files) > 0 {
			lines = append(lines, "")
			hdr := fmt.Sprintf("Files changed (%d) ", len(cd.Files))
			divW := innerW - lipgloss.Width(hdr)
			if divW < 0 {
				divW = 0
			}
			lines = append(lines, sectionDividerStyle.Render(hdr+strings.Repeat("─", divW)))
			lines = append(lines, "")
			for _, f := range cd.Files {
				var sc lipgloss.Color
				switch f.Status {
				case "A":
					sc = clrFileAdded
				case "D":
					sc = clrFileDeleted
				case "R":
					sc = clrFileRenamed
				default:
					sc = clrFileModified
				}
				lines = append(lines, fmt.Sprintf("%s  %s  %s",
					commitDotStyle.Render("●"),
					lipgloss.NewStyle().Foreground(sc).Render(f.Status),
					lipgloss.NewStyle().Foreground(clrCommitTitle).Render(f.Path),
				))
			}
		}

		// ── Diff ───────────────────────────────────────────────────────────
		if len(cd.Diff) > 0 {
			lines = append(lines, "")
			diffHdr := "Diff "
			divW := innerW - lipgloss.Width(diffHdr)
			if divW < 0 {
				divW = 0
			}
			lines = append(lines, sectionDividerStyle.Render(diffHdr+strings.Repeat("─", divW)))
			lines = append(lines, "")
			for _, dl := range cd.Diff {
				var rendered string
				switch dl.Type {
				case "+":
					rendered = lipgloss.NewStyle().Foreground(clrDiffAdded).Render(truncate(dl.Content, innerW))
				case "-":
					rendered = lipgloss.NewStyle().Foreground(clrDiffRemoved).Render(truncate(dl.Content, innerW))
				case "@@":
					rendered = lipgloss.NewStyle().Foreground(clrAccent).Render(truncate(dl.Content, innerW))
				case "diff":
					rendered = lipgloss.NewStyle().Bold(true).Render(truncate(dl.Content, innerW))
				case "meta":
					rendered = dimStyle.Render(truncate(dl.Content, innerW))
				default:
					rendered = lipgloss.NewStyle().Foreground(clrCommitContext).Render(truncate(dl.Content, innerW))
				}
				lines = append(lines, rendered)
			}
		}
	}

	// ── Apply scroll ───────────────────────────────────────────────────────
	total := len(lines)
	maxScroll := total - scrollH
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.commitDetailScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	visible := lines
	if scroll > 0 && scroll < len(lines) {
		visible = lines[scroll:]
	}
	if len(visible) > scrollH {
		visible = visible[:scrollH]
	}
	for len(visible) < scrollH {
		visible = append(visible, "")
	}

	// ── Scroll indicator ───────────────────────────────────────────────────
	// Append a simple N/M indicator when content overflows.
	scrollInfo := ""
	if total > scrollH {
		scrollInfo = "  " + dimStyle.Render(fmt.Sprintf("%d/%d", scroll+1, total))
	}

	hints := m.renderHints("↑↓  scroll", "esc  close") + scrollInfo
	body := strings.Join(visible, "\n") + "\n\n" + hints

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrAccent).
		Padding(1, 2).
		Width(innerW).
		Render(body)
}

// ── Footer ────────────────────────────────────────────────────────────────────

func (m Model) renderFooter() string {
	if m.errMsg != "" {
		return dangerStyle.Render("error: "+m.errMsg) + footerStyle.Render("    (any key to dismiss)")
	}
	switch m.state {
	case types.StateList:
		if m.cursor == 0 {
			return m.renderHints("n  new", "↑↓  navigate", "q  quit")
		}
		if m.cursor-1 < len(m.worktrees) && m.worktrees[m.cursor-1].IsMain {
			return m.renderHints("n  new", "↑↓  navigate", "q  quit")
		}
		return m.renderHints("n  new", "d  delete", "e  edit", "c  cd", "enter  focus", "↑↓  navigate", "q  quit")
	case types.StateRightPaneFocused:
		return m.renderHints("↑↓  navigate commits", "enter  view", "esc  back", "q  quit")
	default:
		return m.renderHints("q  quit")
	}
}

func (m Model) renderHints(hints ...string) string {
	var parts []string
	for _, h := range hints {
		if idx := strings.Index(h, "  "); idx != -1 {
			parts = append(parts, footerKeyStyle.Render(h[:idx])+footerStyle.Render(h[idx:]))
		} else {
			parts = append(parts, footerStyle.Render(h))
		}
	}
	return strings.Join(parts, footerStyle.Render("    "))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// wrapWords breaks s into lines of at most width characters on word boundaries.
func wrapWords(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		line := ""
		for _, w := range words {
			if line == "" {
				line = w
			} else if len(line)+1+len(w) <= width {
				line += " " + w
			} else {
				lines = append(lines, line)
				line = w
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
