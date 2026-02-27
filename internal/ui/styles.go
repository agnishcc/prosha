package ui

import "github.com/charmbracelet/lipgloss"

// General UI colors — ANSI 16-color so they inherit the terminal's palette.
var (
	clrAccent = lipgloss.Color("5") // magenta/purple
	clrDim    = lipgloss.Color("8") // bright-black
	clrGreen  = lipgloss.Color("2")
	clrYellow = lipgloss.Color("3")
	clrRed    = lipgloss.Color("1")
	clrBlue   = lipgloss.Color("4")
)

// Semantic fixed colors for specific indicators (Catppuccin Mocha values).
// These carry intentional meaning so they stay pinned to specific hues.
var (
	clrFlamingo = lipgloss.Color("#f2cdcd") // HEAD sha
	clrPROpen   = lipgloss.Color("#94e2d5") // PR open   — Teal
	clrPRMerged = lipgloss.Color("#cba6f7") // PR merged — Mauve
	clrPRClosed = lipgloss.Color("#f38ba8") // PR closed — Red
	clrPRNone   = lipgloss.Color("#a6adc8") // no PR     — Subtext0

	// Commit detail overlay colors.
	clrCommitTitle   = lipgloss.Color("#cdd6f4") // Text     — commit subject, file paths
	clrCommitBody    = lipgloss.Color("#bac2de") // Subtext1 — commit body
	clrCommitContext = lipgloss.Color("#a6adc8") // Subtext0 — context diff lines, reltime
	clrDiffAdded     = lipgloss.Color("#a6e3a1") // Green    — added diff lines
	clrDiffRemoved   = lipgloss.Color("#f38ba8") // Red      — removed diff lines
	clrFileAdded     = lipgloss.Color("#a6e3a1") // Green    — "A" status
	clrFileModified  = lipgloss.Color("#f9e2af") // Yellow   — "M" status
	clrFileDeleted   = lipgloss.Color("#f38ba8") // Red      — "D" status
	clrFileRenamed   = lipgloss.Color("#cba6f7") // Mauve    — "R" status
)

var (
	// ── Header ───────────────────────────────────────────────────────────────
	headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrDim).
			Padding(0, 1)

	headerBranchStyle = lipgloss.NewStyle().Foreground(clrAccent)
	headerTextStyle   = lipgloss.NewStyle() // default foreground

	// ── Panes ─────────────────────────────────────────────────────────────────
	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrAccent)

	inactivePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrDim)

	// activeRightPaneStyle is used when Level 2 focus shifts to the right pane.
	activeRightPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(clrPRMerged) // Mauve

	// ── List items ────────────────────────────────────────────────────────────
	selectedAccentStyle = lipgloss.NewStyle().Foreground(clrAccent)
	selectedItemStyle   = lipgloss.NewStyle().Bold(true)
	normalItemStyle     = lipgloss.NewStyle().Foreground(clrDim)
	newItemActiveStyle  = lipgloss.NewStyle().Foreground(clrAccent)
	newItemFaintStyle   = lipgloss.NewStyle().Foreground(clrAccent).Faint(true)

	// ── Detail pane ───────────────────────────────────────────────────────────
	detailTitleStyle     = lipgloss.NewStyle().Bold(true)
	detailIndicatorStyle = lipgloss.NewStyle().Foreground(clrGreen)
	detailLabelStyle     = lipgloss.NewStyle().Foreground(clrDim)
	detailValueStyle     = lipgloss.NewStyle()

	commitDotStyle  = lipgloss.NewStyle().Foreground(clrBlue)
	commitHashStyle = lipgloss.NewStyle().Foreground(clrBlue)
	commitMsgStyle  = lipgloss.NewStyle()
	commitTimeStyle = lipgloss.NewStyle().Foreground(clrDim)

	sectionDividerStyle = lipgloss.NewStyle().Foreground(clrDim)
	dimStyle            = lipgloss.NewStyle().Foreground(clrDim)

	// ── Footer ────────────────────────────────────────────────────────────────
	footerStyle    = lipgloss.NewStyle().Foreground(clrDim)
	footerKeyStyle = lipgloss.NewStyle().Foreground(clrAccent).Bold(true)

	// ── Modals ────────────────────────────────────────────────────────────────
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrAccent).
			Padding(1, 2)

	modalTitleStyle   = lipgloss.NewStyle().Bold(true)
	modalLabelStyle   = lipgloss.NewStyle().Foreground(clrDim)
	modalInputStyle   = lipgloss.NewStyle()
	modalPreviewStyle = lipgloss.NewStyle().Foreground(clrAccent)

	selectedTypeStyle   = lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
	unselectedTypeStyle = lipgloss.NewStyle().Foreground(clrDim)

	dangerStyle  = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(clrYellow)

	// ── Shell setup ───────────────────────────────────────────────────────────
	accentStyle = lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
)
